package amnesia

import (
	"bytes"
	"testing"
)

func TestGenerateKeypair(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	if len(pub) != Curve25519KeySize {
		t.Fatalf("public key: expected %d bytes, got %d", Curve25519KeySize, len(pub))
	}
	if len(priv) != Curve25519KeySize {
		t.Fatalf("private key: expected %d bytes, got %d", Curve25519KeySize, len(priv))
	}
}

func TestWrapUnwrapWithPublicKey_RoundTrip(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}

	payload := GenerateKey() // Simulate wrapping an Item Key

	ciphertext, err := WrapWithPublicKey(payload, pub)
	if err != nil {
		t.Fatalf("WrapWithPublicKey: %v", err)
	}

	decrypted, err := UnwrapWithPrivateKey(ciphertext, priv)
	if err != nil {
		t.Fatalf("UnwrapWithPrivateKey: %v", err)
	}

	if !bytes.Equal(payload, decrypted) {
		t.Fatal("decrypted payload does not match original")
	}
}

func TestUnwrapWithPrivateKey_WrongKey(t *testing.T) {
	pub1, _, _ := GenerateKeypair()
	_, priv2, _ := GenerateKeypair()

	payload := []byte("secret item key")

	ciphertext, err := WrapWithPublicKey(payload, pub1)
	if err != nil {
		t.Fatalf("WrapWithPublicKey: %v", err)
	}

	_, err = UnwrapWithPrivateKey(ciphertext, priv2)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong private key")
	}
}

func TestWrapWithPublicKey_DifferentCiphertextsEachTime(t *testing.T) {
	pub, _, _ := GenerateKeypair()
	payload := []byte("same payload")

	ct1, _ := WrapWithPublicKey(payload, pub)
	ct2, _ := WrapWithPublicKey(payload, pub)

	if bytes.Equal(ct1, ct2) {
		t.Fatal("two encryptions of same payload should produce different ciphertexts")
	}
}

func TestDerivePublicKey(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}

	derivedPub, err := DerivePublicKey(priv)
	if err != nil {
		t.Fatalf("DerivePublicKey: %v", err)
	}

	if !bytes.Equal(pub, derivedPub) {
		t.Fatal("derived public key does not match generated public key")
	}
}

func TestWrapUnwrap_LargePayload(t *testing.T) {
	pub, priv, _ := GenerateKeypair()

	// Simulate wrapping a larger payload (e.g., a wrapped Project Vault Key).
	payload := make([]byte, 256)
	copy(payload, "large-payload-for-team-sharing-test")

	ciphertext, err := WrapWithPublicKey(payload, pub)
	if err != nil {
		t.Fatalf("WrapWithPublicKey: %v", err)
	}

	decrypted, err := UnwrapWithPrivateKey(ciphertext, priv)
	if err != nil {
		t.Fatalf("UnwrapWithPrivateKey: %v", err)
	}

	if !bytes.Equal(payload, decrypted) {
		t.Fatal("decrypted large payload does not match original")
	}
}
