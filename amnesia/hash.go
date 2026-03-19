package amnesia

import (
	"crypto/hmac"
	"crypto/sha256"

	"golang.org/x/crypto/argon2"
)

// HashName computes HMAC-SHA256(hmacKey, name) for server-side indexed lookups.
// The server can find vault items by name hash without ever knowing the plaintext name.
func HashName(name string, hmacKey []byte) []byte {
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write([]byte(name))
	return mac.Sum(nil)
}

// VerifyNameHash returns true if the given hash matches HMAC-SHA256(hmacKey, name).
func VerifyNameHash(name string, hmacKey, expected []byte) bool {
	computed := HashName(name, hmacKey)
	return hmac.Equal(computed, expected)
}

// HashAuthKey hashes the Auth Key for server-side storage using Argon2id.
// The server stores this hash — not the Auth Key itself — so even a database
// breach cannot recover the Auth Key without brute-forcing Argon2id.
func HashAuthKey(authKey []byte) []byte {
	// Use a fixed salt derived from the auth key itself for deterministic hashing.
	// This is acceptable because the auth key already has high entropy (32 random-looking bytes
	// from Argon2id). The purpose is verification, not key derivation from a weak input.
	salt := sha256Sum(authKey)[:SaltSize]

	return argon2.IDKey(
		authKey,
		salt,
		3,        // iterations
		64*1024,  // 64 MB memory
		4,        // parallelism
		KeySize,  // 32-byte output
	)
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
