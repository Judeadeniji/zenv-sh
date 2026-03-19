package amnesia

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := GenerateKey()
	plaintext := []byte("DATABASE_URL=postgres://user:pass@localhost/db")

	ciphertext, nonce, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, nonce, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("decrypted does not match original plaintext")
	}
}

func TestEncrypt_UniqueNonces(t *testing.T) {
	key := GenerateKey()
	plaintext := []byte("same data")

	_, nonce1, _ := Encrypt(plaintext, key)
	_, nonce2, _ := Encrypt(plaintext, key)

	if bytes.Equal(nonce1, nonce2) {
		t.Fatal("two Encrypt calls produced identical nonces")
	}
}

func TestEncrypt_UniqueCiphertexts(t *testing.T) {
	key := GenerateKey()
	plaintext := []byte("same data")

	ct1, _, _ := Encrypt(plaintext, key)
	ct2, _, _ := Encrypt(plaintext, key)

	if bytes.Equal(ct1, ct2) {
		t.Fatal("two Encrypt calls with same plaintext should produce different ciphertexts")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := GenerateKey()
	key2 := GenerateKey()
	plaintext := []byte("secret value")

	ciphertext, nonce, _ := Encrypt(plaintext, key1)

	_, err := Decrypt(ciphertext, nonce, key2)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := GenerateKey()
	plaintext := []byte("important data")

	ciphertext, nonce, _ := Encrypt(plaintext, key)

	// Flip a byte.
	ciphertext[0] ^= 0xFF

	_, err := Decrypt(ciphertext, nonce, key)
	if err == nil {
		t.Fatal("expected decryption to fail with tampered ciphertext")
	}
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	_, _, err := Encrypt([]byte("data"), []byte("too-short"))
	if err == nil {
		t.Fatal("expected error for invalid key size")
	}
}

func TestWrapUnwrapKey_RoundTrip(t *testing.T) {
	kek := GenerateKey()
	dek := GenerateKey()

	wrapped, nonce, err := WrapKey(dek, kek)
	if err != nil {
		t.Fatalf("WrapKey: %v", err)
	}

	unwrapped, err := UnwrapKey(wrapped, nonce, kek)
	if err != nil {
		t.Fatalf("UnwrapKey: %v", err)
	}

	if !bytes.Equal(dek, unwrapped) {
		t.Fatal("unwrapped DEK does not match original")
	}
}

func TestUnwrapKey_WrongKEK(t *testing.T) {
	kek1 := GenerateKey()
	kek2 := GenerateKey()
	dek := GenerateKey()

	wrapped, nonce, _ := WrapKey(dek, kek1)

	_, err := UnwrapKey(wrapped, nonce, kek2)
	if err == nil {
		t.Fatal("expected unwrap to fail with wrong KEK")
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	key := GenerateKey()

	ciphertext, nonce, err := Encrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, nonce, key)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatalf("expected empty plaintext, got %d bytes", len(decrypted))
	}
}
