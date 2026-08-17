package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type contextKey string

const CredentialContextKey contextKey = "auth_credential"

// Credential represents an API key permission record with SHA-256 hashed key storage.
type Credential struct {
	KeyHash    string   `json:"key_hash"`
	RawKey     string   `json:"raw_key,omitempty"`
	Namespaces []string `json:"namespaces"` // e.g. ["ns1", "ns2"] or ["*"] for all namespaces
	IsAdmin    bool     `json:"is_admin"`
}

// HashKey produces a SHA-256 hex string for a given raw API key.
func HashKey(rawKey string) string {
	if rawKey == "" {
		return ""
	}
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

// GenerateRandomKey generates a cryptographically random 24-byte API key string.
func GenerateRandomKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "cw_key_" + hex.EncodeToString(b), nil
}

// CanAccessNamespace checks if the credential has permission to access the given namespace.
func (c *Credential) CanAccessNamespace(ns string) bool {
	if c == nil {
		return false
	}
	if c.IsAdmin {
		return true
	}
	for _, allowed := range c.Namespaces {
		if allowed == "*" || allowed == ns {
			return true
		}
	}
	return false
}

// Authenticator maintains an in-memory thread-safe store of hashed API key credentials.
type Authenticator struct {
	mu   sync.RWMutex
	keys map[string]*Credential // keyed by SHA-256 KeyHash
}

// NewAuthenticator creates a new Authenticator initialized with the provided credentials.
func NewAuthenticator(creds []Credential) *Authenticator {
	a := &Authenticator{
		keys: make(map[string]*Credential),
	}
	for _, cred := range creds {
		c := cred
		if c.KeyHash == "" {
			// If empty KeyHash, assume Key field was populated with raw key (backward compatibility)
			c.KeyHash = HashKey(cred.KeyHash)
		}
		a.keys[c.KeyHash] = &c
	}
	return a
}

// NewDefaultAuthenticator creates an authenticator initialized with test keys for unit testing.
func NewDefaultAuthenticator() *Authenticator {
	auth := &Authenticator{
		keys: make(map[string]*Credential),
	}
	auth.AddRawKey("default-admin-key", []string{"*"}, true)
	auth.AddRawKey("default-test-key", []string{"*"}, false)
	return auth
}

// AddRawKey registers a raw key by computing its SHA-256 hash.
func (a *Authenticator) AddRawKey(rawKey string, namespaces []string, isAdmin bool) Credential {
	keyHash := HashKey(rawKey)
	cred := Credential{
		KeyHash:    keyHash,
		RawKey:     rawKey,
		Namespaces: namespaces,
		IsAdmin:    isAdmin,
	}
	a.AddCredentialByHash(cred)
	return cred
}

// AddCredentialByHash registers a pre-hashed credential struct.
func (a *Authenticator) AddCredentialByHash(cred Credential) {
	a.mu.Lock()
	defer a.mu.Unlock()
	c := cred
	a.keys[cred.KeyHash] = &c
}

// RevokeCredentialByHash removes a credential by its SHA-256 KeyHash.
func (a *Authenticator) RevokeCredentialByHash(keyHash string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.keys[keyHash]; exists {
		delete(a.keys, keyHash)
		return true
	}
	return false
}

// RevokeRawKey hashes a raw key and revokes its credential.
func (a *Authenticator) RevokeRawKey(rawKey string) bool {
	return a.RevokeCredentialByHash(HashKey(rawKey))
}

// ValidateKey hashes the incoming key and verifies whether a matching credential exists.
func (a *Authenticator) ValidateKey(rawKey string) (*Credential, bool) {
	if rawKey == "" {
		return nil, false
	}
	keyHash := HashKey(rawKey)

	a.mu.RLock()
	defer a.mu.RUnlock()

	cred, exists := a.keys[keyHash]
	if !exists {
		// Fallback check for raw key match (backwards compatibility)
		cred, exists = a.keys[rawKey]
	}
	if !exists {
		return nil, false
	}
	return cred, true
}

// LookupSecretKey retrieves the secret key and credential associated with an access key ID.
func (a *Authenticator) LookupSecretKey(accessKeyID string) (string, *Credential, bool) {
	if accessKeyID == "" {
		return "", nil, false
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// 1. Direct hash lookup (AccessKeyID is the raw key)
	keyHash := HashKey(accessKeyID)
	if cred, exists := a.keys[keyHash]; exists {
		secretKey := cred.RawKey
		if secretKey == "" {
			secretKey = accessKeyID
		}
		return secretKey, cred, true
	}

	// 2. Fallback check (AccessKeyID stored as key in map directly or matches raw key)
	if cred, exists := a.keys[accessKeyID]; exists {
		secretKey := cred.RawKey
		if secretKey == "" {
			secretKey = accessKeyID
		}
		return secretKey, cred, true
	}

	// 3. Scan for cred.RawKey matching accessKeyID
	for _, cred := range a.keys {
		if cred.RawKey == accessKeyID {
			return cred.RawKey, cred, true
		}
	}

	return "", nil, false
}

// GetAllCredentials returns a snapshot of all registered credentials (with SHA-256 hashes, no raw keys).
func (a *Authenticator) GetAllCredentials() []Credential {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]Credential, 0, len(a.keys))
	for _, c := range a.keys {
		safe := *c
		safe.RawKey = "" // Never expose raw keys via listing (finding #20)
		result = append(result, safe)
	}
	return result
}

// ExtractKey extracts the API key from HTTP request headers.
// Supports "Authorization: Bearer <key>", "Authorization: <key>", and "X-API-Key: <key>".
func ExtractKey(r *http.Request) string {
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return strings.TrimSpace(apiKey)
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(authHeader)
}

// ExtractNamespaceAndFileID parses the request to determine the namespace and fileID.
func ExtractNamespaceAndFileID(r *http.Request) (string, string) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/files/")
	rawPath = strings.TrimPrefix(rawPath, "/")

	nsHeader := r.Header.Get("X-Namespace")
	if nsHeader != "" {
		return strings.TrimSpace(nsHeader), rawPath
	}

	if idx := strings.Index(rawPath, "/"); idx != -1 {
		ns := rawPath[:idx]
		fileID := rawPath[idx+1:]
		if ns != "" && fileID != "" {
			return ns, fileID
		}
	}

	return "default", rawPath
}

// CredentialFromContext retrieves the authenticated Credential from the request context.
func CredentialFromContext(ctx context.Context) (*Credential, bool) {
	cred, ok := ctx.Value(CredentialContextKey).(*Credential)
	return cred, ok
}
