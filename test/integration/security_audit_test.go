package integration

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloudWeave/internal/api"
	"cloudWeave/internal/auth"
	"cloudWeave/internal/coordinator"
	"cloudWeave/internal/erasure"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/s3"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

// TestSecurityAudit_AdminGateStrictIsAdmin tests finding #1:
// A non-admin key with wildcard namespace access ["*"] MUST NOT pass admin gates.
func TestSecurityAudit_AdminGateStrictIsAdmin(t *testing.T) {
	authenticator := auth.NewAuthenticator(nil)
	wildcardNonAdminKey := "cw_key_wildcard_tenant_non_admin"
	authenticator.AddRawKey(wildcardNonAdminKey, []string{"*"}, false) // IsAdmin = false

	metaStore := metadata.NewStore()
	r := ring.New()
	coord := coordinator.NewCoordinator(r, metaStore, "http://localhost:9000", nil, 1, 1, 1)
	apiHandler := api.NewAPIHandler(metaStore, coord, 1024*1024)
	transportServer := transport.NewServer(nil)
	dummyGC := &dummyGCRunner{}
	router := api.NewRouter(apiHandler, transportServer.Handler(), dummyGC, authenticator)

	ts := httptest.NewServer(router)
	defer ts.Close()

	adminEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/keys"},
		{http.MethodPost, "/admin/keys"},
		{http.MethodDelete, "/admin/keys?key_hash=abc"},
		{http.MethodPost, "/admin/join?node_addr=http://localhost:9001"},
		{http.MethodPost, "/admin/leave?node_addr=http://localhost:9001"},
		{http.MethodPost, "/admin/kill?node_addr=http://localhost:9001"},
		{http.MethodPost, "/admin/gc"},
	}

	client := ts.Client()
	for _, ep := range adminEndpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req, err := http.NewRequest(ep.method, ts.URL+ep.path, nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+wildcardNonAdminKey)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("expected 403 Forbidden for non-admin key on %s %s, got %d", ep.method, ep.path, resp.StatusCode)
			}
		})
	}
}

// TestSecurityAudit_SigV4ReplayProtection tests finding #3:
// SigV4 requests with timestamps older than 15 minutes are rejected.
func TestSecurityAudit_SigV4ReplayProtection(t *testing.T) {
	authenticator := auth.NewAuthenticator(nil)
	accessKey := "AKIAIOSFODNN7EXAMPLE"
	secretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	authenticator.AddRawKey(secretKey, []string{"*"}, true)

	metaStore := metadata.NewStore()
	r := ring.New()
	coord := coordinator.NewCoordinator(r, metaStore, "http://localhost:9000", nil, 1, 1, 1)
	s3Handler := s3.NewS3Handler(metaStore, coord, 1024*1024, authenticator)

	// Create request with expired timestamp (2 hours in the past)
	expiredDate := time.Now().UTC().Add(-2 * time.Hour)
	dateStamp := expiredDate.Format("20060102")
	amzDate := expiredDate.Format("20060102T150405Z")
	region := "us-east-1"
	service := "s3"
	scope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)

	// Derive signing key
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))

	canonicalHeaders := fmt.Sprintf("host:localhost:9000\nx-amz-date:%s\n", amzDate)
	signedHeaders := "host;x-amz-date"
	canonicalReq := fmt.Sprintf("GET\n/test-bucket\n\n%s\n%s\nUNSIGNED-PAYLOAD", canonicalHeaders, signedHeaders)
	reqHash := sha256.Sum256([]byte(canonicalReq))
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, scope, hex.EncodeToString(reqHash[:]))
	sig := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", accessKey, scope, signedHeaders, sig)

	req := httptest.NewRequest(http.MethodGet, "/test-bucket", nil)
	req.Host = "localhost:9000"
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Amz-Date", amzDate)

	w := httptest.NewRecorder()
	s3Handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for expired SigV4 request, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSecurityAudit_ClusterSecretEnforcement tests findings #5 and #6:
// Unauthenticated requests to /internal/* and /chunks/* are rejected when cluster secret is configured.
func TestSecurityAudit_ClusterSecretEnforcement(t *testing.T) {
	clusterSecret := "super-secure-cluster-secret-999"
	store, _ := storage.NewDiskStore(t.TempDir())
	metaStore := metadata.NewStore()
	r := ring.New()
	coord := coordinator.NewCoordinator(r, metaStore, "http://localhost:9000", store, 1, 1, 1)
	apiHandler := api.NewAPIHandler(metaStore, coord, 1024*1024)
	transportServer := transport.NewServer(store)
	router := api.NewRouterWithClusterSecret(apiHandler, transportServer.Handler(), nil, clusterSecret)

	ts := httptest.NewServer(router)
	defer ts.Close()
	client := ts.Client()

	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/internal/manifest"},
		{http.MethodPost, "/internal/join?node_addr=http://evil:9000"},
		{http.MethodPost, "/internal/leave?node_addr=http://node1:9000"},
		{http.MethodPost, "/internal/revoke-key?key_hash=123"},
		{http.MethodGet, "/chunks/a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"},
		{http.MethodPut, "/chunks/a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"},
		{http.MethodDelete, "/chunks/a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"},
	}

	for _, ep := range protectedEndpoints {
		t.Run("NoSecret_"+ep.method+"_"+ep.path, func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, ts.URL+ep.path, bytes.NewReader([]byte("{}")))
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("expected 403 Forbidden without cluster secret, got %d", resp.StatusCode)
			}
		})

		t.Run("ValidSecret_"+ep.method+"_"+ep.path, func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, ts.URL+ep.path, bytes.NewReader([]byte("{}")))
			req.Header.Set("X-Cluster-Secret", clusterSecret)
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusForbidden {
				t.Errorf("expected request with valid cluster secret not to return 403, got %d", resp.StatusCode)
			}
		})
	}
}

type dummyGCRunner struct{}

func (d *dummyGCRunner) CollectGarbage() (int, error) {
	return 0, nil
}

// TestSecurityAudit_PathTraversalProtection tests finding #8:
// Directory traversal patterns in chunk IDs and namespaces are rejected.
func TestSecurityAudit_PathTraversalProtection(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewDiskStore(tempDir)
	if err != nil {
		t.Fatalf("NewDiskStore: %v", err)
	}


	server := transport.NewServer(store)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	client := ts.Client()

	traversalChunkIDs := []string{
		"../../etc/passwd",
		"..\\..\\windows\\system32\\config",
		"chunk/subdir",
		"chunk\x00nullbyte",
		"..",
		".",
	}

	for _, badID := range traversalChunkIDs {
		t.Run("Validation_"+badID, func(t *testing.T) {
			if storage.ValidateChunkID(badID) {
				t.Errorf("ValidateChunkID(%q) should return false", badID)
			}
			req, err := http.NewRequest(http.MethodPut, ts.URL+"/chunks/"+badID, strings.NewReader("malicious payload"))
			if err != nil {
				// Malformed URL rejected at request creation time (e.g. null bytes) — safe
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
				t.Errorf("expected rejection for path traversal chunk ID %q, got %d", badID, resp.StatusCode)
			}
		})
	}
}


// TestSecurityAudit_AWSChunkedMaxSizeLimit tests finding #10:
// AWS chunked transfer encoding rejects absurdly large chunk sizes.
func TestSecurityAudit_AWSChunkedMaxSizeLimit(t *testing.T) {
	// 1. Send header with 128 MiB chunk size (0x8000000) which exceeds 64 MiB limit
	oversizedStream := strings.NewReader("8000000;chunk-signature=0000\r\n")
	chunkedReader := s3.NewAWSChunkedReader(oversizedStream)

	buf := make([]byte, 1024)
	_, err := chunkedReader.Read(buf)
	if err == nil {
		t.Fatal("expected error reading oversized chunk size, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum allowed") {
		t.Errorf("expected max allowed error, got %v", err)
	}

	// 2. Send header with int64 overflow (16 F's)
	overflowStream := strings.NewReader("FFFFFFFFFFFFFFFF;chunk-signature=0000\r\n")
	chunkedReader2 := s3.NewAWSChunkedReader(overflowStream)
	_, err2 := chunkedReader2.Read(buf)
	if err2 == nil {
		t.Fatal("expected error reading overflow chunk size, got nil")
	}
}


// TestSecurityAudit_TransportServerBodyLimit tests finding #13:
// Transport server rejects oversized chunk PUTs (>16 MiB).
func TestSecurityAudit_TransportServerBodyLimit(t *testing.T) {
	store, _ := storage.NewDiskStore(t.TempDir())
	server := transport.NewServer(store)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// 17 MiB payload (exceeds 16 MiB maxChunkBodySize)
	largeBody := bytes.Repeat([]byte("A"), 17*1024*1024)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/chunks/validchunk123456", bytes.NewReader(largeBody))
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 Request Entity Too Large for 17MB chunk, got %d", resp.StatusCode)
	}
}

// TestSecurityAudit_ErasureShardIntegrity tests finding #17:
// Corrupted/tampered shards are detected and dropped prior to reconstruction.
func TestSecurityAudit_ErasureShardIntegrity(t *testing.T) {
	enc, err := erasure.NewEncoder(3, 2)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	data := []byte("Sensitive data that must not be silently corrupted by tampered shards!")
	shards, err := enc.Encode(data)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Record legitimate checksums
	expectedChecksums := make(map[int]string)
	available := make(map[int][]byte)
	for _, s := range shards {
		expectedChecksums[s.Index] = s.Checksum
		available[s.Index] = append([]byte(nil), s.Data...)
	}

	// Tamper with shard 0 (flip bytes)
	available[0][0] ^= 0xFF

	// Normal Reconstruct would produce corrupted output
	// ReconstructVerified should detect the tampering, drop shard 0, and reconstruct cleanly from shards 1, 2, 3, 4
	reconstructed, err := enc.ReconstructVerified(available, expectedChecksums, len(data))
	if err != nil {
		t.Fatalf("ReconstructVerified failed: %v", err)
	}

	if !bytes.Equal(reconstructed, data) {
		t.Fatalf("reconstructed data mismatch: got %q, want %q", reconstructed, data)
	}
}

// TestSecurityAudit_GetAllCredentialsNoRawKeys tests finding #20:
// GetAllCredentials never exposes raw plaintext keys.
func TestSecurityAudit_GetAllCredentialsNoRawKeys(t *testing.T) {
	authenticator := auth.NewAuthenticator(nil)
	rawKey := "cw_key_secret_raw_key_12345"
	authenticator.AddRawKey(rawKey, []string{"tenant-1"}, false)

	creds := authenticator.GetAllCredentials()
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}

	if creds[0].RawKey != "" {
		t.Errorf("GetAllCredentials leaked raw key %q, expected empty string", creds[0].RawKey)
	}

	if creds[0].KeyHash != auth.HashKey(rawKey) {
		t.Errorf("expected key hash %s, got %s", auth.HashKey(rawKey), creds[0].KeyHash)
	}
}

// Helper for SigV4 HMAC
func hmacSHA256(key []byte, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
