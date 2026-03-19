package amnesia

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

var (
	ErrAsymmetricDecryptionFailed = errors.New("amnesia: asymmetric decryption failed")
)

const (
	// Curve25519KeySize is the byte length of X25519 public and private keys.
	Curve25519KeySize = 32
)

// GenerateKeypair generates an X25519 keypair for asymmetric encryption.
// Used for team sharing: encrypt Item Keys and Project Vault Keys for recipients.
func GenerateKeypair() (publicKey, privateKey []byte, err error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("amnesia: box.GenerateKey: %w", err)
	}
	return pub[:], priv[:], nil
}

// WrapWithPublicKey encrypts a payload (e.g., an Item Key or Project Vault Key) for a
// specific recipient using their X25519 public key. Generates an ephemeral keypair
// for each operation — the sender's identity is not embedded.
func WrapWithPublicKey(payload, recipientPublicKey []byte) ([]byte, error) {
	if len(recipientPublicKey) != Curve25519KeySize {
		return nil, fmt.Errorf("amnesia: public key must be %d bytes", Curve25519KeySize)
	}

	// Generate ephemeral keypair for this operation.
	ephPub, ephPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("amnesia: ephemeral key generation: %w", err)
	}

	var recipientPub [Curve25519KeySize]byte
	copy(recipientPub[:], recipientPublicKey)

	// Nonce: 24 bytes for NaCl box.
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("amnesia: nonce generation: %w", err)
	}

	// Seal: nonce + ephemeral public key + encrypted payload.
	sealed := box.Seal(nil, payload, &nonce, &recipientPub, ephPriv)

	// Output format: [24-byte nonce][32-byte ephemeral public key][sealed box]
	out := make([]byte, 0, 24+Curve25519KeySize+len(sealed))
	out = append(out, nonce[:]...)
	out = append(out, ephPub[:]...)
	out = append(out, sealed...)

	return out, nil
}

// UnwrapWithPrivateKey decrypts a payload that was encrypted with WrapWithPublicKey.
func UnwrapWithPrivateKey(ciphertext, recipientPrivateKey []byte) ([]byte, error) {
	if len(recipientPrivateKey) != Curve25519KeySize {
		return nil, fmt.Errorf("amnesia: private key must be %d bytes", Curve25519KeySize)
	}

	// Minimum size: 24 (nonce) + 32 (ephemeral pub) + box.Overhead (16)
	minSize := 24 + Curve25519KeySize + box.Overhead
	if len(ciphertext) < minSize {
		return nil, ErrAsymmetricDecryptionFailed
	}

	var nonce [24]byte
	copy(nonce[:], ciphertext[:24])

	var ephPub [Curve25519KeySize]byte
	copy(ephPub[:], ciphertext[24:24+Curve25519KeySize])

	sealed := ciphertext[24+Curve25519KeySize:]

	var recipientPriv [Curve25519KeySize]byte
	copy(recipientPriv[:], recipientPrivateKey)

	plaintext, ok := box.Open(nil, sealed, &nonce, &ephPub, &recipientPriv)
	if !ok {
		return nil, ErrAsymmetricDecryptionFailed
	}

	return plaintext, nil
}

// DerivePublicKey derives the X25519 public key from a private key.
func DerivePublicKey(privateKey []byte) ([]byte, error) {
	if len(privateKey) != Curve25519KeySize {
		return nil, fmt.Errorf("amnesia: private key must be %d bytes", Curve25519KeySize)
	}
	pub, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("amnesia: curve25519.X25519: %w", err)
	}
	return pub, nil
}
