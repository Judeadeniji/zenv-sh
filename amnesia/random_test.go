package amnesia

import (
	"testing"
)

func TestGenerateSalt(t *testing.T) {
	salt := GenerateSalt()
	if len(salt) != SaltSize {
		t.Fatalf("expected %d bytes, got %d", SaltSize, len(salt))
	}

	// Two salts must never be equal.
	salt2 := GenerateSalt()
	if string(salt) == string(salt2) {
		t.Fatal("two GenerateSalt calls returned identical output")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce := GenerateNonce()
	if len(nonce) != NonceSize {
		t.Fatalf("expected %d bytes, got %d", NonceSize, len(nonce))
	}
}

func TestGenerateKey(t *testing.T) {
	key := GenerateKey()
	if len(key) != KeySize {
		t.Fatalf("expected %d bytes, got %d", KeySize, len(key))
	}

	key2 := GenerateKey()
	if string(key) == string(key2) {
		t.Fatal("two GenerateKey calls returned identical output")
	}
}
