package crypto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Judeadeniji/zenv-sh/amnesia"
)

// SecretPayload is the JSON structure encrypted inside each vault item.
type SecretPayload struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// EncryptSecret encrypts a name+value pair with the DEK.
// Returns base64-encoded ciphertext, nonce, and name_hash.
func EncryptSecret(name, value string, dek, hmacKey []byte) (ciphertext, nonce, nameHash string, err error) {
	payload := SecretPayload{Name: name, Value: value}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal payload: %w", err)
	}

	ct, nc, err := amnesia.Encrypt(plaintext, dek)
	if err != nil {
		return "", "", "", fmt.Errorf("encrypt: %w", err)
	}

	nh := amnesia.HashName(name, hmacKey)

	return base64.StdEncoding.EncodeToString(ct),
		base64.StdEncoding.EncodeToString(nc),
		base64.StdEncoding.EncodeToString(nh),
		nil
}

// DecryptSecret decrypts a vault item back to name+value.
func DecryptSecret(ciphertextB64, nonceB64 string, dek []byte) (*SecretPayload, error) {
	ct, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	nc, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}

	plaintext, err := amnesia.Decrypt(ct, nc, dek)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var payload SecretPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return &payload, nil
}

// ComputeNameHash returns the base64-encoded HMAC-SHA256 of a secret name.
func ComputeNameHash(name string, hmacKey []byte) string {
	return base64.StdEncoding.EncodeToString(amnesia.HashName(name, hmacKey))
}

// NameHashURL returns the URL-safe base64 encoding for use in URL paths.
func NameHashURL(name string, hmacKey []byte) string {
	return base64.URLEncoding.EncodeToString(amnesia.HashName(name, hmacKey))
}
