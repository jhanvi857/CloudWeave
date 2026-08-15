package auth

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthenticator_ValidateAndAccess(t *testing.T) {
	auth := NewDefaultAuthenticator()

	// Add dynamic raw key
	rawKey, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}
	if !strings.HasPrefix(rawKey, "cw_key_") {
		t.Errorf("expected cw_key_ prefix, got %s", rawKey)
	}

	cred := auth.AddRawKey(rawKey, []string{"tenant-a", "tenant-b"}, false)
	if cred.KeyHash != HashKey(rawKey) {
		t.Errorf("expected KeyHash %s, got %s", HashKey(rawKey), cred.KeyHash)
	}

	// Validate with raw key
	gotCred, ok := auth.ValidateKey(rawKey)
	if !ok || gotCred == nil {
		t.Fatalf("expected rawKey to validate")
	}
	if !gotCred.CanAccessNamespace("tenant-a") {
		t.Errorf("should access tenant-a")
	}
	if gotCred.CanAccessNamespace("tenant-c") {
		t.Errorf("should NOT access tenant-c")
	}

	// Revoke key
	if !auth.RevokeRawKey(rawKey) {
		t.Errorf("expected RevokeRawKey to return true")
	}
	if _, ok := auth.ValidateKey(rawKey); ok {
		t.Errorf("revoked key should not validate")
	}
}

func TestExtractKey(t *testing.T) {
	req1 := httptest.NewRequest("GET", "/files/x", nil)
	req1.Header.Set("X-API-Key", "my-header-key")
	if key := ExtractKey(req1); key != "my-header-key" {
		t.Errorf("expected my-header-key, got %s", key)
	}

	req2 := httptest.NewRequest("GET", "/files/x", nil)
	req2.Header.Set("Authorization", "Bearer my-bearer-token")
	if key := ExtractKey(req2); key != "my-bearer-token" {
		t.Errorf("expected my-bearer-token, got %s", key)
	}
}

func TestExtractNamespaceAndFileID(t *testing.T) {
	req1 := httptest.NewRequest("GET", "/files/tenant-1/my-file.txt", nil)
	req1.Header.Set("X-Namespace", "override-ns")
	ns, fileID := ExtractNamespaceAndFileID(req1)
	if ns != "override-ns" || fileID != "tenant-1/my-file.txt" {
		t.Errorf("expected (override-ns, tenant-1/my-file.txt), got (%s, %s)", ns, fileID)
	}

	req2 := httptest.NewRequest("GET", "/files/tenant-2/doc.pdf", nil)
	ns, fileID = ExtractNamespaceAndFileID(req2)
	if ns != "tenant-2" || fileID != "doc.pdf" {
		t.Errorf("expected (tenant-2, doc.pdf), got (%s, %s)", ns, fileID)
	}

	req3 := httptest.NewRequest("GET", "/files/simplefile", nil)
	ns, fileID = ExtractNamespaceAndFileID(req3)
	if ns != "default" || fileID != "simplefile" {
		t.Errorf("expected (default, simplefile), got (%s, %s)", ns, fileID)
	}
}
