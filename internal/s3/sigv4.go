package s3

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"cloudWeave/internal/auth"
)

// SigV4AuthResult holds the result of a SigV4 verification.
type SigV4AuthResult struct {
	AccessKeyID   string
	Credential    *auth.Credential
	SecretKey     string
	SigningKey    []byte
	SeedSignature string
	AmzDate       string
	Scope         string
}

// VerifySigV4 verifies AWS Signature Version 4 on an incoming HTTP request.
func VerifySigV4(r *http.Request, authenticator *auth.Authenticator) (*SigV4AuthResult, error) {
	if authenticator == nil {
		return nil, fmt.Errorf("authenticator not configured")
	}

	authHeader := r.Header.Get("Authorization")
	queryAlg := r.URL.Query().Get("X-Amz-Algorithm")

	if authHeader == "" && queryAlg == "" {
		return nil, fmt.Errorf("missing authorization header or query parameters")
	}

	var accessKeyID, dateStamp, region, service, signedHeadersStr, signature string
	var amzDate string

	if authHeader != "" {
		if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256") {
			return nil, fmt.Errorf("unsupported authorization scheme")
		}

		parts := strings.Split(strings.TrimPrefix(authHeader, "AWS4-HMAC-SHA256"), ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			val := strings.Trim(strings.TrimSpace(kv[1]), `"`)

			switch key {
			case "Credential":
				credParts := strings.Split(val, "/")
				if len(credParts) >= 5 {
					accessKeyID = credParts[0]
					dateStamp = credParts[1]
					region = credParts[2]
					service = credParts[3]
				}
			case "SignedHeaders":
				signedHeadersStr = val
			case "Signature":
				signature = val
			}
		}
		amzDate = r.Header.Get("X-Amz-Date")
		if amzDate == "" {
			amzDate = r.Header.Get("Date")
		}
	} else if queryAlg == "AWS4-HMAC-SHA256" {
		credParam := r.URL.Query().Get("X-Amz-Credential")
		credParts := strings.Split(credParam, "/")
		if len(credParts) >= 5 {
			accessKeyID = credParts[0]
			dateStamp = credParts[1]
			region = credParts[2]
			service = credParts[3]
		}
		signedHeadersStr = r.URL.Query().Get("X-Amz-SignedHeaders")
		signature = r.URL.Query().Get("X-Amz-Signature")
		amzDate = r.URL.Query().Get("X-Amz-Date")
	}

	if accessKeyID == "" || signature == "" {
		return nil, fmt.Errorf("invalid authorization header or parameters")
	}

	secretKey, cred, found := authenticator.LookupSecretKey(accessKeyID)
	if !found {
		return nil, fmt.Errorf("InvalidAccessKeyId")
	}

	if amzDate != "" && len(amzDate) >= 8 && dateStamp == "" {
		dateStamp = amzDate[:8]
	}

	// Replay protection: validate request timestamp is within ±15 minutes (finding #3)
	if amzDate != "" {
		var reqTime time.Time
		var parseErr error
		if len(amzDate) >= 16 {
			reqTime, parseErr = time.Parse("20060102T150405Z", amzDate)
		} else if len(amzDate) == 8 {
			reqTime, parseErr = time.Parse("20060102", amzDate)
		}
		if parseErr == nil && !reqTime.IsZero() {
			skew := math.Abs(time.Since(reqTime).Minutes())
			if skew > 15 {
				return nil, fmt.Errorf("SignatureDoesNotMatch: request timestamp is too skewed")
			}
		}
	}

	// 1. Read request body if payload hash needs to be computed
	payloadHash := r.Header.Get("X-Amz-Content-Sha256")
	if payloadHash == "" {
		payloadHash = r.URL.Query().Get("X-Amz-Content-Sha256")
	}

	if payloadHash == "" {
		payloadHash = "UNSIGNED-PAYLOAD"
	}

	// 2. Canonical URI
	canonicalURI := s3EncodePath(r.URL.Path)

	// 3. Canonical Query String
	canonicalQuery := buildCanonicalQueryString(r.URL.Query(), authHeader == "")

	// 4. Canonical Headers & Signed Headers
	signedHeadersList := strings.Split(signedHeadersStr, ";")
	for i := range signedHeadersList {
		signedHeadersList[i] = strings.ToLower(strings.TrimSpace(signedHeadersList[i]))
	}
	sort.Strings(signedHeadersList)

	canonicalHeaders, signedHeaders := buildCanonicalHeaders(r, signedHeadersList)

	// 5. Build Canonical Request
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		r.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	canonicalRequestHash := sha256.Sum256([]byte(canonicalRequest))
	hexCanonicalRequestHash := hex.EncodeToString(canonicalRequestHash[:])

	// 6. Build String to Sign
	scope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, scope, hexCanonicalRequestHash)

	// 7. Derive Signing Key & Compute Expected Signature
	signingKey := deriveSigningKey(secretKey, dateStamp, region, service)
	expectedSignatureBytes := hmacSHA256(signingKey, []byte(stringToSign))
	expectedSignature := hex.EncodeToString(expectedSignatureBytes)

	// 8. Constant-time compare signatures
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expectedSignature))) != 1 {
		return nil, fmt.Errorf("SignatureDoesNotMatch")
	}

	return &SigV4AuthResult{
		AccessKeyID:   accessKeyID,
		Credential:    cred,
		SecretKey:     secretKey,
		SigningKey:    signingKey,
		SeedSignature: signature,
		AmzDate:       amzDate,
		Scope:         scope,
	}, nil
}

func hmacSHA256(key []byte, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func deriveSigningKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func s3Encode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func s3EncodePath(path string) string {
	if path == "" {
		return "/"
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = s3Encode(part)
	}
	return strings.Join(parts, "/")
}

func buildCanonicalQueryString(query url.Values, isPresigned bool) string {
	var keys []string
	for k := range query {
		if isPresigned && k == "X-Amz-Signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		vals := query[k]
		if len(vals) == 0 {
			pairs = append(pairs, fmt.Sprintf("%s=", s3Encode(k)))
			continue
		}
		sort.Strings(vals)
		for _, v := range vals {
			pairs = append(pairs, fmt.Sprintf("%s=%s", s3Encode(k), s3Encode(v)))
		}
	}
	return strings.Join(pairs, "&")
}

func buildCanonicalHeaders(r *http.Request, signedHeadersList []string) (canonicalHeaders string, signedHeaders string) {
	var buf bytes.Buffer
	for _, h := range signedHeadersList {
		var val string
		if h == "host" {
			val = r.Host
			if val == "" {
				val = r.Header.Get("Host")
			}
		} else {
			val = r.Header.Get(h)
		}

		// Trim whitespace and compress whitespace
		val = strings.Join(strings.Fields(val), " ")
		buf.WriteString(fmt.Sprintf("%s:%s\n", h, val))
	}
	return buf.String(), strings.Join(signedHeadersList, ";")
}
