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

// NewEncryptor creates an encryptor from a passphrase and a per-installation
// salt. Returns nil if the passphrase is empty (encryption disabled).
// Uses Argon2id for key derivation so the same passphrase + salt always
// produces the same key. The salt should be randomly generated once per
// installation and persisted (see Store.EncryptionSalt).
func NewEncryptor(passphrase string, salt []byte) (*Encryptor, error) {
	if passphrase == "" {
		return nil, nil
	}
	if len(salt) != 16 {
		return nil, fmt.Errorf("invalid salt length: got %d, want 16", len(salt))
	}
	key := argon2.IDKey([]byte(passphrase), salt, 1, 64*1024, 4, 32)

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
