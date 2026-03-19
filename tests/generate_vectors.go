// generate_vectors produces a JSON file of known-answer test vectors
// from Go Amnesia. The TypeScript Amnesia validates against the same file.
// Any cross-language parity break fails CI.
//
// Run: go run ./tests/generate_vectors.go > tests/vectors.json
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Judeadeniji/zenv-sh/amnesia"
)

type Vectors struct {
	DeriveKeys    []DeriveKeysVector    `json:"deriveKeys"`
	Symmetric     []SymmetricVector     `json:"symmetric"`
	HashName      []HashNameVector      `json:"hashName"`
	HashAuthKey   []HashAuthKeyVector   `json:"hashAuthKey"`
}

type DeriveKeysVector struct {
	VaultKey string `json:"vaultKey"`
	Salt     string `json:"salt"`
	KeyType  string `json:"keyType"`
	KEK      string `json:"kek"`
	AuthKey  string `json:"authKey"`
}

type SymmetricVector struct {
	Plaintext  string `json:"plaintext"`
	Key        string `json:"key"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type HashNameVector struct {
	Name    string `json:"name"`
	HmacKey string `json:"hmacKey"`
	Hash    string `json:"hash"`
}

type HashAuthKeyVector struct {
	AuthKey string `json:"authKey"`
	Hash    string `json:"hash"`
}

func h(b []byte) string { return hex.EncodeToString(b) }
func unhex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func main() {
	v := Vectors{}

	// --- DeriveKeys vectors ---
	// Use passphrase params (faster) for test vectors.
	salt := unhex("0102030405060708091011121314151617181920212223242526272829303132")
	kek, authKey := amnesia.DeriveKeys("test-passphrase", salt, amnesia.KeyTypePassphrase)
	v.DeriveKeys = append(v.DeriveKeys, DeriveKeysVector{
		VaultKey: "test-passphrase",
		Salt:     h(salt),
		KeyType:  "passphrase",
		KEK:      h(kek),
		AuthKey:  h(authKey),
	})

	salt2 := unhex("aabbccddee112233445566778899001122334455667788990011223344556677")
	kek2, authKey2 := amnesia.DeriveKeys("another-key", salt2, amnesia.KeyTypePassphrase)
	v.DeriveKeys = append(v.DeriveKeys, DeriveKeysVector{
		VaultKey: "another-key",
		Salt:     h(salt2),
		KeyType:  "passphrase",
		KEK:      h(kek2),
		AuthKey:  h(authKey2),
	})

	// --- Symmetric vectors ---
	// Fixed key and nonce for deterministic encryption.
	symKey := unhex("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	plaintext := []byte("hello zenv zero-knowledge")
	nonce := unhex("000102030405060708090a0b")

	ciphertext, err := amnesia.EncryptWithNonce(plaintext, symKey, nonce)
	if err != nil {
		fmt.Fprintf(os.Stderr, "EncryptWithNonce: %v\n", err)
		os.Exit(1)
	}
	v.Symmetric = append(v.Symmetric, SymmetricVector{
		Plaintext:  h(plaintext),
		Key:        h(symKey),
		Nonce:      h(nonce),
		Ciphertext: h(ciphertext),
	})

	// Second vector with different data.
	plaintext2 := []byte(`{"name":"DB_URL","value":"postgres://localhost/mydb"}`)
	nonce2 := unhex("aabbccddeeff001122334455")
	ciphertext2, err := amnesia.EncryptWithNonce(plaintext2, symKey, nonce2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "EncryptWithNonce: %v\n", err)
		os.Exit(1)
	}
	v.Symmetric = append(v.Symmetric, SymmetricVector{
		Plaintext:  h(plaintext2),
		Key:        h(symKey),
		Nonce:      h(nonce2),
		Ciphertext: h(ciphertext2),
	})

	// --- HashName vectors ---
	hmacKey := unhex("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	for _, name := range []string{"DATABASE_URL", "API_KEY", "JWT_SECRET"} {
		hash := amnesia.HashName(name, hmacKey)
		v.HashName = append(v.HashName, HashNameVector{
			Name:    name,
			HmacKey: h(hmacKey),
			Hash:    h(hash),
		})
	}

	// --- HashAuthKey vectors ---
	authKeyBytes := unhex("deadbeefcafebabedeadbeefcafebabedeadbeefcafebabedeadbeefcafebabe")
	authHash := amnesia.HashAuthKey(authKeyBytes)
	v.HashAuthKey = append(v.HashAuthKey, HashAuthKeyVector{
		AuthKey: h(authKeyBytes),
		Hash:    h(authHash),
	})

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "json encode: %v\n", err)
		os.Exit(1)
	}
}
