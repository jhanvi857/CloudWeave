package s3

import (
	"bufio"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"strconv"
	"strings"
)

var ErrChunkSignatureMismatch = errors.New("SignatureDoesNotMatch: chunk signature mismatch")

// ChunkSigValidator holds SigV4 state for rolling chunk signature validation.
type ChunkSigValidator struct {
	SigningKey    []byte
	SeedSignature string
	AmzDate       string
	Scope         string
	priorSig      string
}

// AWSChunkedReader wraps an io.Reader reading AWS aws-chunked encoded body streams.
// It parses hex chunk sizes and signature headers (e.g. "1000;chunk-signature=...\r\n"),
// strips trailing \r\n separators, validates per-chunk SigV4 signatures (when configured),
// and yields clean payload bytes.
type AWSChunkedReader struct {
	r               *bufio.Reader
	currentChunkRem int64
	eof             bool
	validator       *ChunkSigValidator
	currentSig      string
	currentHasher   hash.Hash
}

// NewAWSChunkedReader returns an AWSChunkedReader reading from r.
func NewAWSChunkedReader(r io.Reader) *AWSChunkedReader {
	return &AWSChunkedReader{
		r: bufio.NewReader(r),
	}
}

// SetSignatureValidator configures the reader to validate rolling SigV4 chunk signatures.
func (a *AWSChunkedReader) SetSignatureValidator(signingKey []byte, seedSig, amzDate, scope string) {
	if len(signingKey) > 0 && seedSig != "" {
		a.validator = &ChunkSigValidator{
			SigningKey:    signingKey,
			SeedSignature: seedSig,
			AmzDate:       amzDate,
			Scope:         scope,
			priorSig:      seedSig,
		}
	}
}

func (a *AWSChunkedReader) Read(p []byte) (int, error) {
	if a.eof {
		return 0, io.EOF
	}

	if a.currentChunkRem == 0 {
		// Read chunk header line: <hex-size>;chunk-signature=<sig>\r\n
		line, err := a.r.ReadString('\n')
		if err != nil && (err != io.EOF || len(line) == 0) {
			return 0, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			// Skip empty lines between chunks
			return a.Read(p)
		}

		// Split header line at ';' or '\r'
		headerParts := strings.SplitN(line, ";", 2)
		hexSizeStr := strings.TrimSpace(headerParts[0])

		chunkSize, parseErr := strconv.ParseInt(hexSizeStr, 16, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("invalid aws-chunked header %q: %w", line, parseErr)
		}

		// Reject absurdly large chunk sizes to prevent resource exhaustion (finding #10)
		const maxAWSChunkedSize = 64 * 1024 * 1024 // 64 MiB
		if chunkSize > maxAWSChunkedSize {
			return 0, fmt.Errorf("aws-chunked: chunk size %d exceeds maximum allowed %d", chunkSize, maxAWSChunkedSize)
		}

		a.currentSig = extractChunkSignature(line)
		if a.validator != nil {
			a.currentHasher = sha256.New()
		}

		if chunkSize == 0 {
			if err := a.validateChunkSignature("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"); err != nil {
				return 0, err
			}
			a.eof = true
			a.consumeTrailers()
			return 0, io.EOF
		}

		a.currentChunkRem = chunkSize
	}

	toRead := int64(len(p))
	if toRead > a.currentChunkRem {
		toRead = a.currentChunkRem
	}

	n, err := a.r.Read(p[:toRead])
	if n > 0 {
		if a.validator != nil && a.currentHasher != nil {
			a.currentHasher.Write(p[:n])
		}
		a.currentChunkRem -= int64(n)
		if a.currentChunkRem == 0 {
			if a.validator != nil {
				chunkHashHex := hex.EncodeToString(a.currentHasher.Sum(nil))
				if valErr := a.validateChunkSignature(chunkHashHex); valErr != nil {
					return n, valErr
				}
			}
			a.consumeCRLF()
		}
	}
	if err == io.EOF && a.currentChunkRem > 0 {
		return n, io.ErrUnexpectedEOF
	}
	return n, err
}

func extractChunkSignature(line string) string {
	parts := strings.Split(line, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "chunk-signature=") {
			return strings.TrimPrefix(p, "chunk-signature=")
		}
	}
	return ""
}

func (a *AWSChunkedReader) validateChunkSignature(chunkHashHex string) error {
	if a.validator == nil || a.currentSig == "" {
		return nil
	}

	// Build SigV4 chunk StringToSign:
	// AWS4-HMAC-SHA256-PAYLOAD\n<date>\n<scope>\n<priorSig>\n<emptyHeaderHash>\n<chunkHash>
	emptyHeaderHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256-PAYLOAD\n%s\n%s\n%s\n%s\n%s",
		a.validator.AmzDate,
		a.validator.Scope,
		a.validator.priorSig,
		emptyHeaderHash,
		chunkHashHex,
	)

	expectedSigBytes := hmacSHA256(a.validator.SigningKey, []byte(stringToSign))
	expectedSig := hex.EncodeToString(expectedSigBytes)

	if subtle.ConstantTimeCompare([]byte(strings.ToLower(a.currentSig)), []byte(strings.ToLower(expectedSig))) != 1 {
		return ErrChunkSignatureMismatch
	}

	// Update rolling seed for next chunk
	a.validator.priorSig = a.currentSig
	return nil
}

func (a *AWSChunkedReader) consumeCRLF() {
	b, err := a.r.Peek(2)
	if err == nil && string(b) == "\r\n" {
		a.r.Discard(2)
	} else {
		b1, err1 := a.r.Peek(1)
		if err1 == nil && string(b1) == "\n" {
			a.r.Discard(1)
		}
	}
}

func (a *AWSChunkedReader) consumeTrailers() {
	for {
		line, err := a.r.ReadString('\n')
		if err != nil || strings.TrimSpace(line) == "" {
			break
		}
	}
}
