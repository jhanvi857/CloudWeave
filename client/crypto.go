package client

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	keyLen   = 32 // 256 bits for AES-256
	saltLen  = 16 // 128 bits
	nonceLen = 12 // 96 bits for AES-GCM
)

// deriveKey uses Argon2id (3 iterations, 32MB memory, 2 threads) for high-security key derivation.
func deriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, 3, 32*1024, 2, keyLen)
}

// EncryptPayload encrypts data using AES-256-GCM and Argon2id key derivation.
// If convergent is true, deterministic salt and nonce are derived from plaintext HMAC to allow deduplication of identical encrypted chunks.
func EncryptPayload(passphrase string, plaintext []byte, convergent bool) ([]byte, string, string, error) {
	if passphrase == "" {
		return nil, "", "", fmt.Errorf("passphrase cannot be empty")
	}

	var salt, nonce []byte

	if convergent {
		// Convergent Encryption: derive deterministic salt and nonce from plaintext HMAC + passphrase
		// Allows identical plaintexts to yield identical ciphertexts for storage deduplication.
		mac := hmac.New(sha256.New, []byte("cloudweave-convergent-salt:"+passphrase))
		mac.Write(plaintext)
		h := mac.Sum(nil)
		salt = h[:saltLen]

		macNonce := hmac.New(sha256.New, []byte("cloudweave-convergent-nonce:"+passphrase))
		macNonce.Write(plaintext)
		hNonce := macNonce.Sum(nil)
		nonce = hNonce[:nonceLen]
	} else {
		// Standard Randomized Encryption
		salt = make([]byte, saltLen)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, "", "", fmt.Errorf("generating salt: %w", err)
		}

		nonce = make([]byte, nonceLen)
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return nil, "", "", fmt.Errorf("generating nonce: %w", err)
		}
	}

	key := deriveKey(passphrase, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, "", "", fmt.Errorf("creating cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, "", "", fmt.Errorf("creating GCM mode: %w", err)
	}

	ciphertext := aesGCM.Seal(nil, nonce, plaintext, nil)
	return ciphertext, hex.EncodeToString(salt), hex.EncodeToString(nonce), nil
}

// DecryptPayload decrypts AES-256-GCM ciphertext using Argon2id passphrase key derivation.
func DecryptPayload(passphrase, saltHex, nonceHex string, ciphertext []byte) ([]byte, error) {
	if passphrase == "" {
		return nil, fmt.Errorf("decryption passphrase required")
	}

	salt, err := hex.DecodeString(saltHex)
	if err != nil || len(salt) == 0 {
		return nil, fmt.Errorf("invalid encryption salt metadata: %w", err)
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil || len(nonce) == 0 {
		return nil, fmt.Errorf("invalid encryption nonce metadata: %w", err)
	}

	key := deriveKey(passphrase, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM mode: %w", err)
	}

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (invalid passphrase or corrupted payload): %w", err)
	}

	return plaintext, nil
}
