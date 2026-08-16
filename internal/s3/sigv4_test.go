package s3

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloudWeave/internal/auth"
)

func TestSigV4VerificationSuccess(t *testing.T) {
	authenticator := auth.NewDefaultAuthenticator()
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	dateStamp := time.Now().UTC().Format("20060102")
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	region := "us-east-1"
	service := "s3"

	req := httptest.NewRequest(http.MethodGet, "/test-bucket/hello.txt", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Host", "localhost:8080")
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", req.Host, req.Header.Get("X-Amz-Content-Sha256"), amzDate)
	canonicalRequest := fmt.Sprintf("GET\n/test-bucket/hello.txt\n\n%s\n%s\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", canonicalHeaders, signedHeaders)

	cReqHash := sha256.Sum256([]byte(canonicalRequest))
	hexCReqHash := hex.EncodeToString(cReqHash[:])

	scope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, scope, hexCReqHash)

	kDate := hmacSHA256([]byte("AWS4"+rawKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))

	sigBytes := hmacSHA256(kSigning, []byte(stringToSign))
	sigHex := hex.EncodeToString(sigBytes)

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", rawKey, scope, signedHeaders, sigHex)
	req.Header.Set("Authorization", authHeader)

	res, err := VerifySigV4(req, authenticator)
	if err != nil {
		t.Fatalf("expected SigV4 verification to succeed, got error: %v", err)
	}
	if res.AccessKeyID != rawKey {
		t.Errorf("expected AccessKeyID %s, got %s", rawKey, res.AccessKeyID)
	}
}

func TestSigV4VerificationInvalidSignature(t *testing.T) {
	authenticator := auth.NewDefaultAuthenticator()
	rawKey := "default-admin-key"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	amzDate := time.Now().UTC().Format("20060102T150405Z")
	req := httptest.NewRequest(http.MethodGet, "/test-bucket/hello.txt", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Host", "localhost:8080")
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/20260815/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=invalid_hex_signature", rawKey)
	req.Header.Set("Authorization", authHeader)

	_, err := VerifySigV4(req, authenticator)
	if err == nil || err.Error() != "SignatureDoesNotMatch" {
		t.Fatalf("expected SignatureDoesNotMatch error, got: %v", err)
	}
}

func TestSigV4VerificationWrongSecret(t *testing.T) {
	authenticator := auth.NewDefaultAuthenticator()
	realSecret := "valid-secret-key-123"
	wrongSecret := "wrong-secret-key-999"

	authenticator.AddRawKey(realSecret, []string{"*"}, true)

	dateStamp := time.Now().UTC().Format("20060102")
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	region := "us-east-1"
	service := "s3"

	req := httptest.NewRequest(http.MethodGet, "/test-bucket/hello.txt", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Host", "localhost:8080")
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", req.Host, req.Header.Get("X-Amz-Content-Sha256"), amzDate)
	canonicalRequest := fmt.Sprintf("GET\n/test-bucket/hello.txt\n\n%s\n%s\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", canonicalHeaders, signedHeaders)

	cReqHash := sha256.Sum256([]byte(canonicalRequest))
	hexCReqHash := hex.EncodeToString(cReqHash[:])

	scope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, scope, hexCReqHash)

	// Sign using WRONG secret key
	kDate := hmacSHA256([]byte("AWS4"+wrongSecret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))

	sigBytes := hmacSHA256(kSigning, []byte(stringToSign))
	sigHex := hex.EncodeToString(sigBytes)

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", realSecret, scope, signedHeaders, sigHex)
	req.Header.Set("Authorization", authHeader)

	_, err := VerifySigV4(req, authenticator)
	if err == nil || err.Error() != "SignatureDoesNotMatch" {
		t.Fatalf("expected SignatureDoesNotMatch error when signing with wrong secret, got: %v", err)
	}
}
