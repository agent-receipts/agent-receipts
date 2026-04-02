package audit

import (
	"strings"
	"testing"
)

func TestRedactSensitiveKeys(t *testing.T) {
	input := `{"username":"alice","password":"s3cret","data":"safe"}`
	got := Redact(input)
	if strings.Contains(got, "s3cret") {
		t.Error("password not redacted")
	}
	if !strings.Contains(got, "alice") {
		t.Error("username should not be redacted")
	}
	if !strings.Contains(got, "safe") {
		t.Error("data should not be redacted")
	}
}

func TestRedactPatterns(t *testing.T) {
	input := `token: ghp_1234567890123456789012345678901234567`
	got := Redact(input)
	if strings.Contains(got, "ghp_") {
		t.Error("GitHub PAT not redacted")
	}
}

func TestRedactNestedJSON(t *testing.T) {
	input := `{"config":{"api_key":"abc123","host":"example.com"}}`
	got := Redact(input)
	if strings.Contains(got, "abc123") {
		t.Error("nested api_key not redacted")
	}
	if !strings.Contains(got, "example.com") {
		t.Error("host should not be redacted")
	}
}

func TestRedactNonJSON(t *testing.T) {
	input := "plain text with no json"
	got := Redact(input)
	if got != input {
		t.Errorf("expected unchanged string, got %q", got)
	}
}

func TestRedactDeeplyNested(t *testing.T) {
	input := `{"a":{"b":{"c":{"secret":"val"}}}}`
	got := Redact(input)
	if strings.Contains(got, "val") {
		t.Error("deeply nested secret not redacted")
	}
}

func TestRedactPEMBlock(t *testing.T) {
	pem := "-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAJBALR1234567890\nabcdefghijklmnop\n-----END RSA PRIVATE KEY-----"
	input := `some text ` + pem + ` more text`
	got := Redact(input)
	if strings.Contains(got, "MIIBog") {
		t.Error("PEM key body not redacted")
	}
	if strings.Contains(got, "BEGIN RSA PRIVATE KEY") {
		t.Error("PEM header not redacted")
	}
	if !strings.Contains(got, "some text") {
		t.Error("surrounding text should be preserved")
	}
}
