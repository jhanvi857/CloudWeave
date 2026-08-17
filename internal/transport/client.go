package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var (
	defaultTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	defaultPooledClient = &http.Client{
		Timeout:   10 * time.Second,
		Transport: defaultTransport,
	}
)

func DefaultPooledClient() *http.Client {
	return defaultPooledClient
}

func DefaultTransport() *http.Transport {
	return defaultTransport.Clone()
}

type Client struct {
	httpClient    *http.Client
	peerAddr      string
	clusterSecret string
}

func NewClient(peerAddr string) *Client {
	return NewClientWithHTTPClient(peerAddr, defaultPooledClient)
}

func NewClientWithHTTPClient(peerAddr string, httpClient *http.Client) *Client {
	return NewClientWithHTTPClientAndSecret(peerAddr, httpClient, "")
}

func NewClientWithSecret(peerAddr string, clusterSecret string) *Client {
	return NewClientWithHTTPClientAndSecret(peerAddr, defaultPooledClient, clusterSecret)
}

func NewClientWithHTTPClientAndSecret(peerAddr string, httpClient *http.Client, clusterSecret string) *Client {
	if httpClient == nil {
		httpClient = defaultPooledClient
	}
	return &Client{
		httpClient:    httpClient,
		peerAddr:      peerAddr,
		clusterSecret: clusterSecret,
	}
}

func (c *Client) SetClusterSecret(secret string) {
	c.clusterSecret = secret
}

func (c *Client) PutChunk(ctx context.Context, chunkID string, data []byte) error {
	url := fmt.Sprintf("%s/chunks/%s", c.peerAddr, chunkID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	if c.clusterSecret != "" {
		req.Header.Set("X-Cluster-Secret", c.clusterSecret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending chunk %s to %s: %w", chunkID, c.peerAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("peer %s rejected chunk %s: status %d", c.peerAddr, chunkID, resp.StatusCode)
	}
	return nil
}

func (c *Client) GetChunk(ctx context.Context, chunkID string) ([]byte, error) {
	url := fmt.Sprintf("%s/chunks/%s", c.peerAddr, chunkID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if c.clusterSecret != "" {
		req.Header.Set("X-Cluster-Secret", c.clusterSecret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching chunk %s from %s: %w", chunkID, c.peerAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer %s: chunk %s returned status %d", c.peerAddr, chunkID, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) DeleteChunk(ctx context.Context, chunkID string) error {
	url := fmt.Sprintf("%s/chunks/%s", c.peerAddr, chunkID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("building delete request: %w", err)
	}
	if c.clusterSecret != "" {
		req.Header.Set("X-Cluster-Secret", c.clusterSecret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deleting chunk %s from %s: %w", chunkID, c.peerAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("peer %s: delete chunk %s returned status %d", c.peerAddr, chunkID, resp.StatusCode)
	}
	return nil
}


