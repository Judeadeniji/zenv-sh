package crypto

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/Judeadeniji/zenv-sh/amnesia"
)

func TestEncryptSecret_RoundTrip(t *testing.T) {
	dek := amnesia.GenerateKey()
	hmacKey := amnesia.GenerateKey()

	name := "API_KEY"
	value := "super-secret-value"

	ct, nonce, hash, err := EncryptSecret(name, value, dek, hmacKey)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}
	if ct == "" {
		t.Error("Expected non-empty ciphertext")
	}
	if nonce == "" {
		t.Error("Expected non-empty nonce")
	}
	if hash == "" {
		t.Error("Expected non-empty name hash")
	}

	// Decrypt and verify round-trip
	payload, err := DecryptSecret(ct, nonce, dek)
	if err != nil {
		t.Fatalf("DecryptSecret failed: %v", err)
	}
	if payload.Name != name {
		t.Errorf("Expected name=%s, got %s", name, payload.Name)
	}
	if payload.Value != value {
		t.Errorf("Expected value=%s, got %s", value, payload.Value)
	}
}

func TestEncryptSecret_UniqueNonces(t *testing.T) {
	dek := amnesia.GenerateKey()
	hmacKey := amnesia.GenerateKey()

	_, nonce1, _, err := EncryptSecret("KEY", "val", dek, hmacKey)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}
	_, nonce2, _, err := EncryptSecret("KEY", "val", dek, hmacKey)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}
	// Each encryption call should produce a unique nonce
	if nonce1 == nonce2 {
		t.Error("Expected unique nonces for each encryption, got identical nonces")
	}
}

func TestEncryptSecret_SameKeyDifferentHashes(t *testing.T) {
	dek := amnesia.GenerateKey()
	hmacKey := amnesia.GenerateKey()

	_, _, hash1, _ := EncryptSecret("SECRET_A", "val1", dek, hmacKey)
	_, _, hash2, _ := EncryptSecret("SECRET_B", "val2", dek, hmacKey)

	if hash1 == hash2 {
		t.Error("Expected different name hashes for different secret names")
	}
}

func TestDecryptSecret_WrongKey(t *testing.T) {
	dek := amnesia.GenerateKey()
	wrongDek := amnesia.GenerateKey()
	hmacKey := amnesia.GenerateKey()

	ct, nonce, _, err := EncryptSecret("MY_KEY", "my_value", dek, hmacKey)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	_, err = DecryptSecret(ct, nonce, wrongDek)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key")
	}
}

func TestDecryptSecret_InvalidBase64Ciphertext(t *testing.T) {
	dek := amnesia.GenerateKey()

	_, err := DecryptSecret("not-valid-base64!!!", "dGVzdA==", dek)
	if err == nil {
		t.Error("Expected error for invalid base64 ciphertext")
	}
	if !strings.Contains(err.Error(), "decode ciphertext") {
		t.Errorf("Expected 'decode ciphertext' in error, got: %v", err)
	}
}

func TestDecryptSecret_InvalidBase64Nonce(t *testing.T) {
	dek := amnesia.GenerateKey()
	hmacKey := amnesia.GenerateKey()

	ct, _, _, err := EncryptSecret("KEY", "val", dek, hmacKey)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	_, err = DecryptSecret(ct, "not-valid-base64!!!", dek)
	if err == nil {
		t.Error("Expected error for invalid base64 nonce")
	}
	if !strings.Contains(err.Error(), "decode nonce") {
		t.Errorf("Expected 'decode nonce' in error, got: %v", err)
	}
}

func TestDecryptSecret_TamperedCiphertext(t *testing.T) {
	dek := amnesia.GenerateKey()
	hmacKey := amnesia.GenerateKey()

	ct, nonce, _, err := EncryptSecret("KEY", "val", dek, hmacKey)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	// Tamper with ciphertext
	raw, _ := base64.StdEncoding.DecodeString(ct)
	if len(raw) > 0 {
		raw[0] ^= 0xFF
	}
	tamperedCt := base64.StdEncoding.EncodeToString(raw)

	_, err = DecryptSecret(tamperedCt, nonce, dek)
	if err == nil {
		t.Error("Expected error for tampered ciphertext")
	}
}

func TestComputeNameHash_Deterministic(t *testing.T) {
	hmacKey := amnesia.GenerateKey()
	name := "DATABASE_URL"

	hash1 := ComputeNameHash(name, hmacKey)
	hash2 := ComputeNameHash(name, hmacKey)

	if hash1 != hash2 {
		t.Errorf("ComputeNameHash should be deterministic, got different results: %s vs %s", hash1, hash2)
	}
}

func TestComputeNameHash_DifferentKeys(t *testing.T) {
	key1 := amnesia.GenerateKey()
	key2 := amnesia.GenerateKey()
	name := "API_KEY"

	hash1 := ComputeNameHash(name, key1)
	hash2 := ComputeNameHash(name, key2)

	if hash1 == hash2 {
		t.Error("Expected different hashes for different HMAC keys")
	}
}

func TestComputeNameHash_DifferentNames(t *testing.T) {
	hmacKey := amnesia.GenerateKey()

	hash1 := ComputeNameHash("SECRET_A", hmacKey)
	hash2 := ComputeNameHash("SECRET_B", hmacKey)

	if hash1 == hash2 {
		t.Error("Expected different hashes for different secret names")
	}
}

func TestComputeNameHash_IsBase64(t *testing.T) {
	hmacKey := amnesia.GenerateKey()
	hash := ComputeNameHash("SOME_KEY", hmacKey)

	_, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		t.Errorf("ComputeNameHash should return valid base64, got: %s, error: %v", hash, err)
	}
}

func TestNameHashURL_IsURLSafeBase64(t *testing.T) {
	hmacKey := amnesia.GenerateKey()
	hash := NameHashURL("MY_SECRET", hmacKey)

	_, err := base64.URLEncoding.DecodeString(hash)
	if err != nil {
		t.Errorf("NameHashURL should return valid URL-safe base64, got: %s, error: %v", hash, err)
	}
}

func TestNameHashURL_Deterministic(t *testing.T) {
	hmacKey := amnesia.GenerateKey()
	name := "MY_TOKEN"

	hash1 := NameHashURL(name, hmacKey)
	hash2 := NameHashURL(name, hmacKey)

	if hash1 != hash2 {
		t.Errorf("NameHashURL should be deterministic, got %s vs %s", hash1, hash2)
	}
}

func TestComputeNameHash_MatchesEncryptHash(t *testing.T) {
	dek := amnesia.GenerateKey()
	name := "STRIPE_KEY"

	// EncryptSecret uses hmacKey for name hash
	_, _, encHash, err := EncryptSecret(name, "value", dek, dek)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	// ComputeNameHash with the same key should produce the same hash
	computedHash := ComputeNameHash(name, dek)

	if encHash != computedHash {
		t.Errorf("EncryptSecret name hash %s doesn't match ComputeNameHash %s", encHash, computedHash)
	}
}

func TestEncryptSecret_EmptyValue(t *testing.T) {
	dek := amnesia.GenerateKey()
	hmacKey := amnesia.GenerateKey()

	ct, nonce, _, err := EncryptSecret("EMPTY_KEY", "", dek, hmacKey)
	if err != nil {
		t.Fatalf("EncryptSecret should handle empty value, got: %v", err)
	}

	payload, err := DecryptSecret(ct, nonce, dek)
	if err != nil {
		t.Fatalf("DecryptSecret failed: %v", err)
	}
	if payload.Value != "" {
		t.Errorf("Expected empty value, got %s", payload.Value)
	}
}