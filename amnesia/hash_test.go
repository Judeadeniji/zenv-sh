package amnesia

import (
	"bytes"
	"testing"
)

func TestHashName_Deterministic(t *testing.T) {
	key := GenerateKey()
	name := "DATABASE_URL"

	h1 := HashName(name, key)
	h2 := HashName(name, key)

	if !bytes.Equal(h1, h2) {
		t.Fatal("HashName not deterministic")
	}
}

func TestHashName_DifferentNamesProduceDifferentHashes(t *testing.T) {
	key := GenerateKey()

	h1 := HashName("DATABASE_URL", key)
	h2 := HashName("STRIPE_KEY", key)

	if bytes.Equal(h1, h2) {
		t.Fatal("different names should produce different hashes")
	}
}

func TestHashName_DifferentKeysProduceDifferentHashes(t *testing.T) {
	key1 := GenerateKey()
	key2 := GenerateKey()
	name := "DATABASE_URL"

	h1 := HashName(name, key1)
	h2 := HashName(name, key2)

	if bytes.Equal(h1, h2) {
		t.Fatal("different HMAC keys should produce different hashes")
	}
}

func TestVerifyNameHash(t *testing.T) {
	key := GenerateKey()
	name := "API_KEY"

	hash := HashName(name, key)

	if !VerifyNameHash(name, key, hash) {
		t.Fatal("VerifyNameHash should return true for correct input")
	}

	if VerifyNameHash("WRONG_NAME", key, hash) {
		t.Fatal("VerifyNameHash should return false for wrong name")
	}
}

func TestHashAuthKey_Deterministic(t *testing.T) {
	authKey := GenerateKey()

	h1 := HashAuthKey(authKey)
	h2 := HashAuthKey(authKey)

	if !bytes.Equal(h1, h2) {
		t.Fatal("HashAuthKey not deterministic")
	}
}

func TestHashAuthKey_OutputSize(t *testing.T) {
	authKey := GenerateKey()
	hash := HashAuthKey(authKey)

	if len(hash) != KeySize {
		t.Fatalf("expected %d bytes, got %d", KeySize, len(hash))
	}
}

func TestHashAuthKey_DifferentInputsDiffer(t *testing.T) {
	h1 := HashAuthKey(GenerateKey())
	h2 := HashAuthKey(GenerateKey())

	if bytes.Equal(h1, h2) {
		t.Fatal("different auth keys should produce different hashes")
	}
}
