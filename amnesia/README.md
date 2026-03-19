# Amnesia (Go)

Pure cryptographic primitive library for zEnv. No network. No storage. No concept of users, projects, or secrets. It takes bytes in and gives bytes out.

The only external dependency is `golang.org/x/crypto`.

## API

```go
// Key derivation — one Argon2id run, 64-byte output split by convention
// bytes 0-31 → KEK, bytes 32-63 → Auth Key
DeriveKeys(vaultKey string, salt []byte, keyType KeyType) (kek, authKey []byte)

// Symmetric encryption (AES-256-GCM, 96-bit random nonce per call)
Encrypt(plaintext, key []byte) (ciphertext, nonce []byte, err error)
Decrypt(ciphertext, nonce, key []byte) (plaintext []byte, err error)

// Key wrapping (semantically identical to Encrypt/Decrypt)
WrapKey(dek, kek []byte) (ciphertext, nonce []byte, err error)
UnwrapKey(ciphertext, nonce, kek []byte) (dek []byte, err error)

// Name hashing for server-side indexed lookup without revealing names
HashName(name string, hmacKey []byte) []byte

// Auth Key hashing (Argon2id) before server storage
HashAuthKey(authKey []byte) []byte

// Asymmetric encryption (X25519 + NaCl box)
GenerateKeypair() (publicKey, privateKey []byte, err error)
WrapWithPublicKey(payload, publicKey []byte) ([]byte, error)
UnwrapWithPrivateKey(ciphertext, privateKey []byte) ([]byte, error)

// Secure random generation
GenerateSalt() []byte   // 32 bytes
GenerateNonce() []byte  // 12 bytes
GenerateKey() []byte    // 32 bytes
```

## Usage

```go
import "github.com/Judeadeniji/zenv-sh/amnesia"

// Derive keys from a vault key
salt := amnesia.GenerateSalt()
kek, authKey := amnesia.DeriveKeys("my-passphrase", salt, amnesia.KeyTypePassphrase)

// Generate and wrap a DEK
dek := amnesia.GenerateKey()
wrappedDEK, nonce, _ := amnesia.WrapKey(dek, kek)

// Encrypt a secret
plaintext := []byte(`{"name":"DB_URL","value":"postgres://..."}`)
ciphertext, nonce, _ := amnesia.Encrypt(plaintext, dek)

// Decrypt
decrypted, _ := amnesia.Decrypt(ciphertext, nonce, dek)
```

## Testing

```bash
make test-amnesia
# or
go test -v -count=1 ./amnesia/...
```

31 tests covering round-trip encryption, known-answer vectors, key derivation determinism, and error cases.

## Purity Rules

Amnesia must **never**:

- Make HTTP requests
- Read from or write to the filesystem
- Import anything from the API, CLI, or SDK packages
- Know what a "user", "project", or "environment" is
- Have any configuration beyond what's passed to each function

This purity means Amnesia is independently auditable. A security firm reviewing zEnv's zero-knowledge claims reads this package and can verify the entire cryptographic story without touching the rest of the codebase.
