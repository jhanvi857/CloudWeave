package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ObjectInfo holds metadata information about a retrieved object.
type ObjectInfo struct {
	Size        int64             `json:"size"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata"`
}

// ObjectVersion holds metadata for a historical version of an object.
type ObjectVersion struct {
	VersionID   string            `json:"version_id"`
	Size        int64             `json:"size"`
	ContentType string            `json:"content_type,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Config holds initialization options for the CloudWeave client.
type Config struct {
	Endpoints           []string      // List of cluster node HTTP addresses (e.g. ["http://localhost:8080", "http://localhost:8081"])
	APIKey              string        // API key or token for authentication
	Namespace           string        // Default namespace for object keys (defaults to "default")
	HTTPClient          *http.Client  // Optional HTTP client instance
	MaxRetries          int           // Max retry attempts per request across endpoints (defaults to 3)
	Timeout             time.Duration // Per-request timeout (defaults to 10s)
	EnableAutoDiscovery bool          // Enable automatic background topology discovery (defaults to false)
	DiscoveryInterval   time.Duration // Interval for topology discovery (defaults to 30s)
	EncryptionPassphrase string       // Optional client-side AES-256-GCM encryption passphrase
}

// Client is the idiomatic Go client for interacting with CloudWeave object storage.
type Client struct {
	mu                   sync.RWMutex
	endpoints            []string
	apiKey               string
	namespace            string
	httpClient           *http.Client
	maxRetries           int
	rrCounter            uint64
	stopChan             chan struct{}
	encryptionPassphrase string
}

// PutOptions holds options for object upload operations.
type PutOptions struct {
	ContentType          string
	Metadata             map[string]string
	EncryptionPassphrase string
}

// New creates and validates a new CloudWeave Client instance.
func New(cfg Config) (*Client, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one cluster node endpoint must be specified")
	}

	cleanedEndpoints := make([]string, 0, len(cfg.Endpoints))
	for _, ep := range cfg.Endpoints {
		trimmed := strings.TrimRight(strings.TrimSpace(ep), "/")
		if trimmed != "" {
			if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
				trimmed = "http://" + trimmed
			}
			cleanedEndpoints = append(cleanedEndpoints, trimmed)
		}
	}

	if len(cleanedEndpoints) == 0 {
		return nil, fmt.Errorf("invalid endpoint list")
	}

	ns := cfg.Namespace
	if ns == "" {
		ns = "default"
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	hc := cfg.HTTPClient
	if hc == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		hc = &http.Client{Timeout: timeout}
	}

	cli := &Client{
		endpoints:            cleanedEndpoints,
		apiKey:               cfg.APIKey,
		namespace:            ns,
		httpClient:           hc,
		maxRetries:           maxRetries,
		stopChan:             make(chan struct{}),
		encryptionPassphrase: cfg.EncryptionPassphrase,
	}

	if cfg.EnableAutoDiscovery {
		interval := cfg.DiscoveryInterval
		if interval <= 0 {
			interval = 30 * time.Second
		}
		go cli.startDiscoveryLoop(interval)
	}

	return cli, nil
}

// Close stops background routines such as auto-discovery loops.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopChan != nil {
		close(c.stopChan)
		c.stopChan = nil
	}
}

// startDiscoveryLoop periodically refreshes active cluster topology.
func (c *Client) startDiscoveryLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = c.DiscoverNodes(ctx)
			cancel()
		case <-c.stopChan:
			return
		}
	}
}

// DiscoverNodes queries GET /cluster/nodes from active endpoints to dynamically refresh cluster topology.
func (c *Client) DiscoverNodes(ctx context.Context) error {
	resp, err := c.executeWithRetry(ctx, http.MethodGet, "/cluster/nodes", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node discovery failed (status %d): %s", resp.StatusCode, string(body))
	}

	var activeNodes []string
	if err := json.NewDecoder(resp.Body).Decode(&activeNodes); err != nil {
		return fmt.Errorf("failed to decode node list: %w", err)
	}

	if len(activeNodes) > 0 {
		cleaned := make([]string, 0, len(activeNodes))
		for _, ep := range activeNodes {
			trimmed := strings.TrimRight(strings.TrimSpace(ep), "/")
			if trimmed != "" {
				if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
					trimmed = "http://" + trimmed
				}
				cleaned = append(cleaned, trimmed)
			}
		}
		if len(cleaned) > 0 {
			c.mu.Lock()
			c.endpoints = cleaned
			c.mu.Unlock()
		}
	}

	return nil
}

// GetEndpoints returns a snapshot of active cluster node endpoints.
func (c *Client) GetEndpoints() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	eps := make([]string, len(c.endpoints))
	copy(eps, c.endpoints)
	return eps
}

// getNextEndpoint returns an endpoint using round-robin distribution.
func (c *Client) getNextEndpoint() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.endpoints) == 0 {
		return ""
	}
	idx := atomic.AddUint64(&c.rrCounter, 1) % uint64(len(c.endpoints))
	return c.endpoints[idx]
}

// executeWithRetry executes an HTTP request operation against available node endpoints with retry failover.
func (c *Client) executeWithRetry(ctx context.Context, method, path string, body []byte, reqHeaders map[string]string) (*http.Response, error) {
	var lastErr error

	attempts := c.maxRetries
	if attempts < len(c.endpoints) {
		attempts = len(c.endpoints)
	}

	for i := 0; i < attempts; i++ {
		endpoint := c.getNextEndpoint()
		reqURL := endpoint + path

		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Apply authentication and namespace headers
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
		if c.namespace != "" {
			req.Header.Set("X-Namespace", c.namespace)
		}

		// Apply custom headers
		for k, v := range reqHeaders {
			req.Header.Set(k, v)
		}

		resp, err := c.httpClient.Do(req)
		if err == nil {
			if resp.StatusCode < 500 {
				return resp, nil
			}
			// Server error (5xx) -> buffer body and retry on another node
			resp.Body.Close()
			lastErr = fmt.Errorf("node %s returned server error status %d", endpoint, resp.StatusCode)
		} else {
			lastErr = fmt.Errorf("failed to connect to node %s: %w", endpoint, err)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	return nil, fmt.Errorf("quorum retry failed across %d node attempts: %w", attempts, lastErr)
}

// SetEncryptionPassphrase sets or updates the client-side encryption passphrase.
func (c *Client) SetEncryptionPassphrase(passphrase string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.encryptionPassphrase = passphrase
}

// Put uploads an object stream to CloudWeave.
func (c *Client) Put(ctx context.Context, key string, data []byte, opts ...PutOptions) error {
	path := "/files/" + url.PathEscape(key)

	headers := make(map[string]string)

	c.mu.RLock()
	passphrase := c.encryptionPassphrase
	c.mu.RUnlock()

	if len(opts) > 0 {
		opt := opts[0]
		if opt.EncryptionPassphrase != "" {
			passphrase = opt.EncryptionPassphrase
		}
		if opt.ContentType != "" {
			headers["Content-Type"] = opt.ContentType
		}
		for k, v := range opt.Metadata {
			headers["X-Meta-"+k] = v
		}
	}

	if passphrase != "" {
		encryptedData, saltHex, nonceHex, err := EncryptPayload(passphrase, data, true)
		if err != nil {
			return fmt.Errorf("client-side encryption failed: %w", err)
		}
		data = encryptedData
		headers["X-Meta-encrypted"] = "true"
		headers["X-Meta-enc-salt"] = saltHex
		headers["X-Meta-enc-nonce"] = nonceHex
	}

	resp, err := c.executeWithRetry(ctx, http.MethodPut, path, data, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Get downloads an object from CloudWeave.
func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	path := "/files/" + url.PathEscape(key)

	resp, err := c.executeWithRetry(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, nil, fmt.Errorf("get object failed (status %d): %s", resp.StatusCode, string(body))
	}

	info := extractObjectInfo(resp)

	if info.Metadata["encrypted"] == "true" || info.Metadata["enc-salt"] != "" {
		c.mu.RLock()
		passphrase := c.encryptionPassphrase
		c.mu.RUnlock()

		ciphertext, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("reading encrypted body: %w", err)
		}

		saltHex := info.Metadata["enc-salt"]
		nonceHex := info.Metadata["enc-nonce"]

		plaintext, err := DecryptPayload(passphrase, saltHex, nonceHex, ciphertext)
		if err != nil {
			return nil, nil, err
		}

		return io.NopCloser(bytes.NewReader(plaintext)), info, nil
	}

	return resp.Body, info, nil
}

// GetVersion downloads a specific historical version of an object from CloudWeave.
func (c *Client) GetVersion(ctx context.Context, key, versionID string) (io.ReadCloser, *ObjectInfo, error) {
	path := "/files/" + url.PathEscape(key) + "?version_id=" + url.QueryEscape(versionID)

	resp, err := c.executeWithRetry(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, nil, fmt.Errorf("get object version failed (status %d): %s", resp.StatusCode, string(body))
	}

	info := extractObjectInfo(resp)

	if info.Metadata["encrypted"] == "true" || info.Metadata["enc-salt"] != "" {
		c.mu.RLock()
		passphrase := c.encryptionPassphrase
		c.mu.RUnlock()

		ciphertext, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("reading encrypted body: %w", err)
		}

		saltHex := info.Metadata["enc-salt"]
		nonceHex := info.Metadata["enc-nonce"]

		plaintext, err := DecryptPayload(passphrase, saltHex, nonceHex, ciphertext)
		if err != nil {
			return nil, nil, err
		}

		return io.NopCloser(bytes.NewReader(plaintext)), info, nil
	}

	return resp.Body, info, nil
}

// ListVersions lists all available versions of an object key.
func (c *Client) ListVersions(ctx context.Context, key string) ([]ObjectVersion, error) {
	path := "/files/" + url.PathEscape(key) + "?versions=true"

	resp, err := c.executeWithRetry(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list object versions failed (status %d): %s", resp.StatusCode, string(body))
	}

	var rawManifests []struct {
		VersionID   string            `json:"version_id"`
		Size        int64             `json:"size"`
		ContentType string            `json:"content_type"`
		Metadata    map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawManifests); err != nil {
		return nil, fmt.Errorf("failed to decode object versions: %w", err)
	}

	versions := make([]ObjectVersion, len(rawManifests))
	for i, m := range rawManifests {
		versions[i] = ObjectVersion{
			VersionID:   m.VersionID,
			Size:        m.Size,
			ContentType: m.ContentType,
			Metadata:    m.Metadata,
		}
	}
	return versions, nil
}

// RangeGet reads a specific byte range [start, end] of an object from CloudWeave.
func (c *Client) RangeGet(ctx context.Context, key string, start, end int64) (io.ReadCloser, *ObjectInfo, error) {
	path := "/files/" + url.PathEscape(key)
	headers := map[string]string{
		"Range": fmt.Sprintf("bytes=%d-%d", start, end),
	}

	resp, err := c.executeWithRetry(ctx, http.MethodGet, path, nil, headers)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, nil, fmt.Errorf("range get failed (status %d): %s", resp.StatusCode, string(body))
	}

	info := extractObjectInfo(resp)
	return resp.Body, info, nil
}

// Delete removes an object from CloudWeave.
func (c *Client) Delete(ctx context.Context, key string) error {
	path := "/files/" + url.PathEscape(key)

	resp, err := c.executeWithRetry(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete object failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// JoinCluster requests the cluster to dynamically register a new node address.
func (c *Client) JoinCluster(ctx context.Context, newNodeAddr string) error {
	path := "/admin/join?node_addr=" + url.QueryEscape(newNodeAddr)

	resp, err := c.executeWithRetry(ctx, http.MethodPost, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("join cluster failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// LeaveCluster requests the cluster to gracefully remove a node address.
func (c *Client) LeaveCluster(ctx context.Context, targetNodeAddr string) error {
	path := "/admin/leave?node_addr=" + url.QueryEscape(targetNodeAddr)

	resp, err := c.executeWithRetry(ctx, http.MethodPost, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("leave cluster failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// CollectGarbage triggers garbage collection on the cluster.
func (c *Client) CollectGarbage(ctx context.Context) (string, error) {
	path := "/admin/gc"

	resp, err := c.executeWithRetry(ctx, http.MethodPost, path, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GC failed (status %d): %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func extractObjectInfo(resp *http.Response) *ObjectInfo {
	info := &ObjectInfo{
		ContentType: resp.Header.Get("Content-Type"),
		Metadata:    make(map[string]string),
	}

	if contentLengthStr := resp.Header.Get("Content-Length"); contentLengthStr != "" {
		info.Size, _ = strconv.ParseInt(contentLengthStr, 10, 64)
	}

	for k, v := range resp.Header {
		lower := strings.ToLower(k)
		if strings.HasPrefix(lower, "x-meta-") {
			metaKey := strings.TrimPrefix(lower, "x-meta-")
			if len(v) > 0 {
				info.Metadata[metaKey] = v[0]
			}
		}
	}

	return info
}
