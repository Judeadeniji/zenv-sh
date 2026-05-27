package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/Judeadeniji/zenv-sh/amnesia"
)

func TestEncryptDecryptSecret(t *testing.T) {
	dek := amnesia.GenerateKey()
	hmacKey := amnesia.GenerateKey()

	name := "DB_URL"
	value := "postgres://user:pass@localhost/db"

	ct, nc, nh, err := EncryptSecret(name, value, dek, hmacKey)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	if ct == "" || nc == "" || nh == "" {
		t.Errorf("EncryptSecret returned empty string(s)")
	}

	// Verify name hash is correct
	expectedHash := base64.StdEncoding.EncodeToString(amnesia.HashName(name, hmacKey))
	if nh != expectedHash {
		t.Errorf("Expected name hash %s, got %s", expectedHash, nh)
	}

	payload, err := DecryptSecret(ct, nc, dek)
	if err != nil {
		t.Fatalf("DecryptSecret failed: %v", err)
	}

	if payload.Name != name {
		t.Errorf("Expected name %s, got %s", name, payload.Name)
	}
	if payload.Value != value {
		t.Errorf("Expected value %s, got %s", value, payload.Value)
	}
}

func TestDecryptSecret_Errors(t *testing.T) {
	dek := amnesia.GenerateKey()
	
	// Invalid base64 for ciphertext
	_, err := DecryptSecret("invalid-base64!!", "valid", dek)
	if err == nil {
		t.Error("Expected error for invalid ciphertext base64")
	}

	// Invalid base64 for nonce
	_, err = DecryptSecret("validct", "invalid-base64!!", dek)
	if err == nil {
		t.Error("Expected error for invalid nonce base64")
	}

	// Tampered ciphertext
	ctB64 := base64.StdEncoding.EncodeToString([]byte("fake-ciphertext-too-short"))
	ncB64 := base64.StdEncoding.EncodeToString(amnesia.GenerateNonce())
	_, err = DecryptSecret(ctB64, ncB64, dek)
	if err == nil {
		t.Error("Expected decryption error for tampered ciphertext")
	}
}

func TestComputeNameHash(t *testing.T) {
	hmacKey := amnesia.GenerateKey()
	name := "API_KEY"

	hash1 := ComputeNameHash(name, hmacKey)
	hash2 := ComputeNameHash(name, hmacKey)

	if hash1 != hash2 {
		t.Errorf("ComputeNameHash is not deterministic")
	}
}

func TestNameHashURL(t *testing.T) {
	hmacKey := amnesia.GenerateKey()
	name := "API_KEY"

	urlHash := NameHashURL(name, hmacKey)
	
	// Ensure it's valid URL base64
	_, err := base64.URLEncoding.DecodeString(urlHash)
	if err != nil {
		t.Errorf("NameHashURL did not return valid URLEncoding base64: %v", err)
	}
}
