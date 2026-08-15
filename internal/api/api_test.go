package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloudWeave/internal/auth"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

const testAdminKey = "default-admin-key"

func TestAPIHandler_PutAndGetSingleNode(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewDiskStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}

	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	metaStore := metadata.NewStore()
	apiHandler := NewAPIHandler(metaStore, adapter, 10)
	router := NewRouter(apiHandler, transport.NewServer(store).Handler(), nil)

	ts := httptest.NewServer(router)
	defer ts.Close()

	payload := []byte("The quick brown fox jumps over the lazy dog. 1234567890!")
	fileID := "test-file-1"

	// 1. PUT /files/test-file-1
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/files/"+fileID, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create PUT request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+testAdminKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201 Created, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Verify metadata recorded
	m, found := metaStore.Lookup(fileID)
	if !found {
		t.Fatalf("metadata manifest not recorded")
	}
	if m.Size != int64(len(payload)) {
		t.Errorf("manifest size mismatch: got %d, want %d", m.Size, len(payload))
	}
	if len(m.ChunkIDs) <= 1 {
		t.Errorf("expected payload to be split into multiple chunks, got %d chunk(s)", len(m.ChunkIDs))
	}

	// 2. GET /files/test-file-1
	getReq, err := http.NewRequest(http.MethodGet, ts.URL+"/files/"+fileID, nil)
	if err != nil {
		t.Fatalf("failed to create GET request: %v", err)
	}
	getReq.Header.Set("Authorization", "Bearer "+testAdminKey)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", getResp.StatusCode)
	}

	downloaded, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("failed to read GET response body: %v", err)
	}

	if !bytes.Equal(downloaded, payload) {
		t.Fatalf("byte mismatch!\nGot:  %s\nWant: %s", string(downloaded), string(payload))
	}
}

type mockGCRunner struct{}

func (m *mockGCRunner) CollectGarbage() (int, error) {
	return 0, nil
}

func TestAPIHandler_AuthRejection(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewDiskStore(tempDir)
	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	authenticator := auth.NewAuthenticator([]auth.Credential{
		{KeyHash: auth.HashKey("admin-key"), Namespaces: []string{"*"}, IsAdmin: true},
		{KeyHash: auth.HashKey("tenant1-key"), Namespaces: []string{"tenant1"}, IsAdmin: false},
	})

	metaStore := metadata.NewStore()
	apiHandler := NewAPIHandler(metaStore, adapter, 100)
	router := NewRouter(apiHandler, nil, &mockGCRunner{}, authenticator)
	ts := httptest.NewServer(router)
	defer ts.Close()

	// 1. Missing Key -> 401 Unauthorized
	reqNoKey, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/tenant1/file.txt", bytes.NewReader([]byte("hello")))
	respNoKey, err := http.DefaultClient.Do(reqNoKey)
	if err != nil || respNoKey.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing key, got %v (status %d)", err, respNoKey.StatusCode)
	}
	respNoKey.Body.Close()

	// 2. Invalid Key -> 401 Unauthorized
	reqBadKey, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/tenant1/file.txt", bytes.NewReader([]byte("hello")))
	reqBadKey.Header.Set("X-API-Key", "invalid-key")
	respBadKey, err := http.DefaultClient.Do(reqBadKey)
	if err != nil || respBadKey.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid key, got %v (status %d)", err, respBadKey.StatusCode)
	}
	respBadKey.Body.Close()

	// 3. Forbidden Namespace -> 403 Forbidden (tenant1 key trying tenant2)
	reqForbidden, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/tenant2/file.txt", bytes.NewReader([]byte("hello")))
	reqForbidden.Header.Set("X-API-Key", "tenant1-key")
	respForbidden, err := http.DefaultClient.Do(reqForbidden)
	if err != nil || respForbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for unassigned namespace, got %v (status %d)", err, respForbidden.StatusCode)
	}
	respForbidden.Body.Close()

	// 4. Admin GC without admin key -> 403 Forbidden
	reqGCForbidden, _ := http.NewRequest(http.MethodPost, ts.URL+"/admin/gc", nil)
	reqGCForbidden.Header.Set("X-API-Key", "tenant1-key")
	respGCForbidden, err := http.DefaultClient.Do(reqGCForbidden)
	if err != nil || respGCForbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for admin endpoint, got %v (status %d)", err, respGCForbidden.StatusCode)
	}
	respGCForbidden.Body.Close()
}

func TestAPIHandler_NamespaceIsolation(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewDiskStore(tempDir)
	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	authenticator := auth.NewAuthenticator([]auth.Credential{
		{KeyHash: auth.HashKey("userA-key"), Namespaces: []string{"tenant-a"}},
		{KeyHash: auth.HashKey("userB-key"), Namespaces: []string{"tenant-b"}},
	})

	metaStore := metadata.NewStore()
	apiHandler := NewAPIHandler(metaStore, adapter, 100)
	router := NewRouter(apiHandler, nil, nil, authenticator)
	ts := httptest.NewServer(router)
	defer ts.Close()

	key := "shared-object-key"
	contentA := []byte("Tenant A Data")
	contentB := []byte("Tenant B Data")

	// User A PUT /files/tenant-a/shared-object-key
	putA, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/tenant-a/"+key, bytes.NewReader(contentA))
	putA.Header.Set("Authorization", "Bearer userA-key")
	respA, _ := http.DefaultClient.Do(putA)
	if respA.StatusCode != http.StatusCreated {
		t.Fatalf("User A PUT failed with status %d", respA.StatusCode)
	}
	respA.Body.Close()

	// User B PUT /files/tenant-b/shared-object-key
	putB, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/tenant-b/"+key, bytes.NewReader(contentB))
	putB.Header.Set("Authorization", "Bearer userB-key")
	respB, _ := http.DefaultClient.Do(putB)
	if respB.StatusCode != http.StatusCreated {
		t.Fatalf("User B PUT failed with status %d", respB.StatusCode)
	}
	respB.Body.Close()

	// User A GET should receive contentA
	getA, _ := http.NewRequest(http.MethodGet, ts.URL+"/files/tenant-a/"+key, nil)
	getA.Header.Set("Authorization", "Bearer userA-key")
	respGetA, _ := http.DefaultClient.Do(getA)
	if respGetA.StatusCode != http.StatusOK {
		t.Fatalf("User A GET failed with status %d", respGetA.StatusCode)
	}
	bodyA, _ := io.ReadAll(respGetA.Body)
	respGetA.Body.Close()
	if !bytes.Equal(bodyA, contentA) {
		t.Errorf("User A got '%s', want '%s'", string(bodyA), string(contentA))
	}

	// User B GET should receive contentB
	getB, _ := http.NewRequest(http.MethodGet, ts.URL+"/files/tenant-b/"+key, nil)
	getB.Header.Set("Authorization", "Bearer userB-key")
	respGetB, _ := http.DefaultClient.Do(getB)
	if respGetB.StatusCode != http.StatusOK {
		t.Fatalf("User B GET failed with status %d", respGetB.StatusCode)
	}
	bodyB, _ := io.ReadAll(respGetB.Body)
	respGetB.Body.Close()
	if !bytes.Equal(bodyB, contentB) {
		t.Errorf("User B got '%s', want '%s'", string(bodyB), string(contentB))
	}
}

func TestAPIHandler_ObjectMetadata(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewDiskStore(tempDir)
	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	metaStore := metadata.NewStore()
	apiHandler := NewAPIHandler(metaStore, adapter, 100)
	router := NewRouter(apiHandler, nil, nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	fileID := "metadata-file.json"
	payload := []byte(`{"message": "hello world"}`)

	// PUT with Content-Type and custom metadata headers
	putReq, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/"+fileID, bytes.NewReader(payload))
	putReq.Header.Set("Authorization", "Bearer "+testAdminKey)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("X-Meta-Environment", "production")
	putReq.Header.Set("X-Meta-Owner", "TeamAlpha")

	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil || putResp.StatusCode != http.StatusCreated {
		t.Fatalf("PUT failed: %v, status %d", err, putResp.StatusCode)
	}
	putResp.Body.Close()

	// GET and verify response headers
	getReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/files/"+fileID, nil)
	getReq.Header.Set("Authorization", "Bearer "+testAdminKey)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil || getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET failed: %v, status %d", err, getResp.StatusCode)
	}
	defer getResp.Body.Close()

	if contentType := getResp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
	if envHeader := getResp.Header.Get("X-Meta-Environment"); envHeader != "production" {
		t.Errorf("expected X-Meta-Environment 'production', got '%s'", envHeader)
	}
	if ownerHeader := getResp.Header.Get("X-Meta-Owner"); ownerHeader != "TeamAlpha" {
		t.Errorf("expected X-Meta-Owner 'TeamAlpha', got '%s'", ownerHeader)
	}
}

func TestAPIHandler_RangeRequest(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewDiskStore(tempDir)
	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	metaStore := metadata.NewStore()
	apiHandler := NewAPIHandler(metaStore, adapter, 10)
	router := NewRouter(apiHandler, nil, nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	payload := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	fileID := "range-file"

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/"+fileID, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+testAdminKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("PUT failed: %v, status %d", err, resp.StatusCode)
	}
	resp.Body.Close()

	getReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/files/"+fileID, nil)
	getReq.Header.Set("Authorization", "Bearer "+testAdminKey)
	getReq.Header.Set("Range", "bytes=5-15")

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("Range GET failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected 206 Partial Content, got %d", getResp.StatusCode)
	}

	if contentRange := getResp.Header.Get("Content-Range"); contentRange != "bytes 5-15/36" {
		t.Fatalf("expected Content-Range 'bytes 5-15/36', got '%s'", contentRange)
	}

	rangeBody, _ := io.ReadAll(getResp.Body)
	expectedRange := payload[5:16]
	if !bytes.Equal(rangeBody, expectedRange) {
		t.Fatalf("range body mismatch! Got '%s', want '%s'", string(rangeBody), string(expectedRange))
	}
}

func TestAPIHandler_DeleteFile(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewDiskStore(tempDir)
	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	metaStore := metadata.NewStore()
	apiHandler := NewAPIHandler(metaStore, adapter, 100)
	router := NewRouter(apiHandler, nil, nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	fileID := "delete-me"
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/"+fileID, bytes.NewReader([]byte("temp content")))
	req.Header.Set("Authorization", "Bearer "+testAdminKey)
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/files/"+fileID, nil)
	delReq.Header.Set("Authorization", "Bearer "+testAdminKey)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	delResp.Body.Close()

	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on DELETE, got %d", delResp.StatusCode)
	}

	_, exists := metaStore.Lookup(fileID)
	if exists {
		t.Fatalf("expected file manifest to be deleted from store")
	}
}

func TestAPIHandler_DynamicKeyManagementAndClusterNodes(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewDiskStore(tempDir)
	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	metaStore := metadata.NewStore()
	authenticator := auth.NewDefaultAuthenticator()
	apiHandler := NewAPIHandler(metaStore, adapter, 100)
	router := NewRouter(apiHandler, nil, nil, authenticator)
	ts := httptest.NewServer(router)
	defer ts.Close()

	// 1. GET /cluster/nodes (unauthenticated -> 401)
	nodesReqNoAuth, _ := http.NewRequest(http.MethodGet, ts.URL+"/cluster/nodes", nil)
	nodesRespNoAuth, _ := http.DefaultClient.Do(nodesReqNoAuth)
	if nodesRespNoAuth.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for /cluster/nodes without key, got %d", nodesRespNoAuth.StatusCode)
	}
	nodesRespNoAuth.Body.Close()

	// 2. GET /cluster/nodes (authenticated)
	nodesReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/cluster/nodes", nil)
	nodesReq.Header.Set("Authorization", "Bearer "+testAdminKey)
	nodesResp, err := http.DefaultClient.Do(nodesReq)
	if err != nil || nodesResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /cluster/nodes failed: %v, status %d", err, nodesResp.StatusCode)
	}
	nodesResp.Body.Close()

	// 3. POST /admin/keys to issue a key
	keyBody := `{"namespaces": ["tenant-dyn"], "is_admin": false}`
	postKeyReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/admin/keys", bytes.NewReader([]byte(keyBody)))
	postKeyReq.Header.Set("Authorization", "Bearer "+testAdminKey)
	postKeyReq.Header.Set("Content-Type", "application/json")

	postKeyResp, err := http.DefaultClient.Do(postKeyReq)
	if err != nil || postKeyResp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /admin/keys failed: %v, status %d", err, postKeyResp.StatusCode)
	}
	defer postKeyResp.Body.Close()

	var issued struct {
		Key        string   `json:"key"`
		KeyHash    string   `json:"key_hash"`
		Namespaces []string `json:"namespaces"`
		IsAdmin    bool     `json:"is_admin"`
	}
	_ = json.NewDecoder(postKeyResp.Body).Decode(&issued)

	if !strings.HasPrefix(issued.Key, "cw_key_") {
		t.Fatalf("expected rawKey starting with cw_key_, got %s", issued.Key)
	}

	// 4. Test newly issued rawKey for PUT in tenant-dyn
	putReq, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/tenant-dyn/doc.txt", bytes.NewReader([]byte("dynamic data")))
	putReq.Header.Set("Authorization", "Bearer "+issued.Key)
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil || putResp.StatusCode != http.StatusCreated {
		t.Fatalf("PUT using dynamically issued key failed: %v, status %d", err, putResp.StatusCode)
	}
	putResp.Body.Close()

	// 5. Revoke key via DELETE /admin/keys
	delKeyReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/admin/keys?key_hash="+issued.KeyHash, nil)
	delKeyReq.Header.Set("Authorization", "Bearer "+testAdminKey)
	delKeyResp, err := http.DefaultClient.Do(delKeyReq)
	if err != nil || delKeyResp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE /admin/keys failed: %v, status %d", err, delKeyResp.StatusCode)
	}
	delKeyResp.Body.Close()

	// 6. Test revoked key -> 401 Unauthorized
	putReqRevoked, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/tenant-dyn/doc2.txt", bytes.NewReader([]byte("data")))
	putReqRevoked.Header.Set("Authorization", "Bearer "+issued.Key)
	putRevokedResp, _ := http.DefaultClient.Do(putReqRevoked)
	if putRevokedResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized after key revocation, got %d", putRevokedResp.StatusCode)
	}
	putRevokedResp.Body.Close()
}

