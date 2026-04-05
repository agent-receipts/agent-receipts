package audit

import (
	"testing"
)

var testSalt = []byte("0123456789abcdef")

func TestEncryptDecryptRoundtrip(t *testing.T) {
	enc, err := NewEncryptor("test-passphrase", testSalt)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := "sensitive audit data"
	encrypted, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == plaintext {
		t.Error("encrypted should differ from plaintext")
	}
	if encrypted[:4] != "enc:" {
		t.Error("expected enc: prefix")
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != plaintext {
		t.Errorf("got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	enc1, err := NewEncryptor("passphrase-one", testSalt)
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := NewEncryptor("passphrase-two", testSalt)
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := enc1.Encrypt("secret data")
	if err != nil {
		t.Fatal(err)
	}

	_, err = enc2.Decrypt(encrypted)
	if err == nil {
		t.Error("expected error when decrypting with wrong passphrase")
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	enc, err := NewEncryptor("test-passphrase", testSalt)
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := enc.Encrypt("secret data")
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt a byte in the base64 payload (after "enc:" prefix).
	corrupted := []byte(encrypted)
	// Flip a byte somewhere in the middle of the payload.
	idx := len("enc:") + 10
	if idx < len(corrupted) {
		corrupted[idx] ^= 0xFF
	}

	_, err = enc.Decrypt(string(corrupted))
	if err == nil {
		t.Error("expected error when decrypting corrupted ciphertext")
	}
}

func TestDecryptTamperedPrefix(t *testing.T) {
	enc, err := NewEncryptor("test-passphrase", testSalt)
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := enc.Encrypt("secret data")
	if err != nil {
		t.Fatal(err)
	}

	// Remove the "enc:" prefix — Decrypt should return the raw string unchanged.
	withoutPrefix := encrypted[len("enc:"):]
	got, err := enc.Decrypt(withoutPrefix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != withoutPrefix {
		t.Errorf("expected raw string passthrough, got %q", got)
	}
}

func TestNilEncryptorPassthrough(t *testing.T) {
	enc, err := NewEncryptor("", testSalt)
	if err != nil {
		t.Fatal(err)
	}
	if enc != nil {
		t.Error("expected nil encryptor for empty passphrase")
	}

	got, err := enc.Encrypt("hello")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("expected passthrough, got %q", got)
	}

	got, err = enc.Decrypt("hello")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestNewEncryptorRejectsInvalidSalt(t *testing.T) {
	_, err := NewEncryptor("passphrase", nil)
	if err == nil {
		t.Error("expected error for nil salt")
	}
	_, err = NewEncryptor("passphrase", []byte("short"))
	if err == nil {
		t.Error("expected error for short salt")
	}
	_, err = NewEncryptor("passphrase", []byte("this-is-too-long-for-a-salt"))
	if err == nil {
		t.Error("expected error for long salt")
	}
}
