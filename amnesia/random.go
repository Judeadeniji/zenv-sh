package amnesia

import (
	"crypto/rand"
)

const (
	// SaltSize is the byte length of salts used for Argon2id key derivation.
	SaltSize = 32

	// NonceSize is the byte length of nonces used for AES-256-GCM.
	NonceSize = 12

	// KeySize is the byte length of symmetric keys (AES-256).
	KeySize = 32
)

// GenerateSalt returns 32 cryptographically random bytes for use as an Argon2id salt.
func GenerateSalt() []byte {
	return mustRandBytes(SaltSize)
}

// GenerateNonce returns 12 cryptographically random bytes for use as an AES-256-GCM nonce.
func GenerateNonce() []byte {
	return mustRandBytes(NonceSize)
}

// GenerateKey returns 32 cryptographically random bytes for use as an AES-256 key (e.g. DEK).
func GenerateKey() []byte {
	return mustRandBytes(KeySize)
}

func mustRandBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("amnesia: crypto/rand failed: " + err.Error())
	}
	return b
}
