package amnesia

import (
	"golang.org/x/crypto/argon2"
)

// KeyType determines Argon2id parameters. PIN uses aggressive parameters to
// compensate for low entropy; passphrase uses lighter parameters.
type KeyType string

const (
	KeyTypePIN        KeyType = "pin"
	KeyTypePassphrase KeyType = "passphrase"

	// DerivedKeySize is the total Argon2id output: 32 bytes KEK + 32 bytes Auth Key.
	DerivedKeySize = 64
)

// Argon2idParams holds the memory, iterations, and parallelism for Argon2id.
type Argon2idParams struct {
	Memory      uint32 // in KiB
	Iterations  uint32
	Parallelism uint8
}

// ParamsForKeyType returns the Argon2id parameters appropriate for the given key type.
//
//   - PIN (~20 bits entropy): m=256MB, t=10, p=4 — Argon2id does the heavy lifting.
//   - Passphrase (~50-70 bits): m=64MB, t=3, p=4 — entropy does the heavy lifting.
func ParamsForKeyType(kt KeyType) Argon2idParams {
	switch kt {
	case KeyTypePIN:
		return Argon2idParams{
			Memory:      256 * 1024, // 256 MB
			Iterations:  10,
			Parallelism: 4,
		}
	case KeyTypePassphrase:
		return Argon2idParams{
			Memory:      64 * 1024, // 64 MB
			Iterations:  3,
			Parallelism: 4,
		}
	default:
		// Default to passphrase params for unknown types.
		return ParamsForKeyType(KeyTypePassphrase)
	}
}

// DeriveKeys runs Argon2id exactly once, producing a 64-byte output that is split
// by convention into two 32-byte keys:
//
//   - bytes 0-31:  KEK (Key Encryption Key) — used to wrap/unwrap the DEK.
//   - bytes 32-63: Auth Key — hashed and sent to the server for Vault Key verification.
//
// The Vault Key and salt are the only inputs. The Vault Key never leaves the client.
// Argon2id is never run twice per unlock — one run, one output, split by fixed convention.
func DeriveKeys(vaultKey string, salt []byte, keyType KeyType) (kek, authKey []byte) {
	params := ParamsForKeyType(keyType)

	derived := argon2.IDKey(
		[]byte(vaultKey),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		DerivedKeySize,
	)

	kek = make([]byte, KeySize)
	authKey = make([]byte, KeySize)
	copy(kek, derived[:KeySize])
	copy(authKey, derived[KeySize:])

	return kek, authKey
}
