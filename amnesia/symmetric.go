package amnesia

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

var (
	ErrInvalidKeySize    = errors.New("amnesia: key must be 32 bytes")
	ErrCiphertextTooShort = errors.New("amnesia: ciphertext too short")
	ErrDecryptionFailed  = errors.New("amnesia: decryption failed — wrong key or corrupted data")
)

// Encrypt encrypts plaintext using AES-256-GCM with the given key.
// Returns the ciphertext and a randomly generated 96-bit nonce.
// Each call produces a unique nonce — never reuse nonces with the same key.
func Encrypt(plaintext, key []byte) (ciphertext, nonce []byte, err error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, nil, err
	}

	nonce = GenerateNonce()
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// EncryptWithNonce encrypts using a caller-supplied nonce.
// Only for deterministic test vector generation — production code must use Encrypt.
func EncryptWithNonce(plaintext, key, nonce []byte) (ciphertext []byte, err error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("amnesia: nonce must be %d bytes, got %d", NonceSize, len(nonce))
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the given key and nonce.
func Decrypt(ciphertext, nonce, key []byte) (plaintext []byte, err error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("amnesia: nonce must be %d bytes, got %d", NonceSize, len(nonce))
	}

	plaintext, err = gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	return plaintext, nil
}

// WrapKey wraps a DEK with a KEK using AES-256-GCM. Semantically identical to
// Encrypt but named separately to make the intent clear in calling code.
func WrapKey(dek, kek []byte) (ciphertext, nonce []byte, err error) {
	return Encrypt(dek, kek)
}

// UnwrapKey unwraps a DEK using a KEK. Semantically identical to Decrypt.
func UnwrapKey(ciphertext, nonce, kek []byte) (dek []byte, err error) {
	return Decrypt(ciphertext, nonce, kek)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("amnesia: aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("amnesia: cipher.NewGCM: %w", err)
	}
	return gcm, nil
}
