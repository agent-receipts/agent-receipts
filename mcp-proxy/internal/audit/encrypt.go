package audit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const encryptedPrefix = "enc:"

// Encryptor provides optional AES-256-GCM encryption for audit data.
type Encryptor struct {
	gcm cipher.AEAD
}

// Fixed salt for deterministic key derivation. Using a constant salt means
// the same passphrase always produces the same AES-256-GCM key. Brute-force
// resistance comes from the Argon2id parameters, not the salt.
var argon2Salt = []byte("mcp-proxy-audit")

// NewEncryptor creates an encryptor from a passphrase.
// Returns nil if the passphrase is empty (encryption disabled).
// Uses Argon2id for key derivation so the same passphrase always produces the
// same key. Since we need the same key for the lifetime of the encryptor, we
// derive it once at creation time and reuse it for all operations.
func NewEncryptor(passphrase string) (*Encryptor, error) {
	if passphrase == "" {
		return nil, nil
	}
	key := argon2.IDKey([]byte(passphrase), argon2Salt, 1, 64*1024, 4, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns prefixed base64.
// Returns an error if encryption fails — never falls back to plaintext.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if e == nil {
		return plaintext, nil
	}
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a value if it has the encrypted prefix.
func (e *Encryptor) Decrypt(value string) (string, error) {
	if e == nil || !strings.HasPrefix(value, encryptedPrefix) {
		return value, nil
	}
	data, err := base64.StdEncoding.DecodeString(value[len(encryptedPrefix):])
	if err != nil {
		return "", err
	}
	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	plaintext, err := e.gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
