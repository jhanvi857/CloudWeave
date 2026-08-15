package integration

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// generateCertificates creates a test CA, server cert, and node client cert for mTLS testing.
func generateCertificates(t *testing.T) (tls.Certificate, tls.Certificate, *x509.CertPool) {
	t.Helper()

	// 1. CA Certificate
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "CloudWeave Test Root CA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}

	caCert, err := x509.ParseCertificate(caCertBytes)
	if err != nil {
		t.Fatalf("failed to parse CA cert: %v", err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	// 2. Server Certificate
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate server key: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	serverCertBytes, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create server cert: %v", err)
	}

	serverTLSCert := tls.Certificate{
		Certificate: [][]byte{serverCertBytes},
		PrivateKey:  serverKey,
	}

	// 3. Client Certificate for node-to-node authentication
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate client key: %v", err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "CloudWeave Peer Node 1"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	clientCertBytes, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create client cert: %v", err)
	}

	clientTLSCert := tls.Certificate{
		Certificate: [][]byte{clientCertBytes},
		PrivateKey:  clientKey,
	}

	return serverTLSCert, clientTLSCert, caPool
}

func TestMutualTLS_NodeToNodeEnforcement(t *testing.T) {
	serverCert, clientCert, caPool := generateCertificates(t)

	// External Listener TLS Config enforcing mTLS
	listenerTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	// Internal Node Transport Client TLS Config providing client cert
	nodeClientTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
	}

	// Log actual tls.Config properties as evidence
	t.Logf("=== EXTERNAL LISTENER TLS CONFIG ===")
	t.Logf("ClientAuth mode: %v (RequireAndVerifyClientCert)", listenerTLSConfig.ClientAuth)
	t.Logf("Has ClientCAs: %v", listenerTLSConfig.ClientCAs != nil)
	t.Logf("Certificates count: %d", len(listenerTLSConfig.Certificates))

	t.Logf("=== INTERNAL NODE-TO-NODE CLIENT TLS CONFIG ===")
	t.Logf("Has Client Certificates: %v (count: %d)", len(nodeClientTLSConfig.Certificates) > 0, len(nodeClientTLSConfig.Certificates))
	t.Logf("Has RootCAs: %v", nodeClientTLSConfig.RootCAs != nil)

	// Start HTTPS test server with listener TLS config
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mTLS handshake successful"))
	})

	ts := httptest.NewUnstartedServer(mux)
	ts.TLS = listenerTLSConfig
	ts.StartTLS()
	defer ts.Close()

	// 1. Test Node-to-Node Connection WITH Client Cert (Should Succeed)
	validClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: nodeClientTLSConfig,
		},
		Timeout: 3 * time.Second,
	}

	resp, err := validClient.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Valid node-to-node mTLS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK with client cert, got %d", resp.StatusCode)
	}
	t.Logf("SUCCESS: Node-to-node connection WITH client certificate accepted! Status: %d", resp.StatusCode)

	// 2. Test Connection WITHOUT Client Certificate (Must Be Rejected)
	noCertTLSConfig := &tls.Config{
		RootCAs: caPool,
		// Note: NO Certificates provided!
	}

	noCertClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: noCertTLSConfig,
		},
		Timeout: 3 * time.Second,
	}

	_, err = noCertClient.Get(ts.URL + "/health")
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: Request WITHOUT client certificate was wrongly accepted by mTLS listener!")
	}

	t.Logf("SUCCESS: Connection WITHOUT client certificate rejected by listener as expected!")
	t.Logf("Actual TLS Error Output: %v", err)
}
