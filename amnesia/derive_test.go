package amnesia

import (
	"bytes"
	"testing"
)

func TestDeriveKeys_Deterministic(t *testing.T) {
	salt := bytes.Repeat([]byte{0xAA}, SaltSize)
	vaultKey := "correct-horse-battery-staple"

	kek1, auth1 := DeriveKeys(vaultKey, salt, KeyTypePassphrase)
	kek2, auth2 := DeriveKeys(vaultKey, salt, KeyTypePassphrase)

	if !bytes.Equal(kek1, kek2) {
		t.Fatal("DeriveKeys not deterministic: KEK differs")
	}
	if !bytes.Equal(auth1, auth2) {
		t.Fatal("DeriveKeys not deterministic: AuthKey differs")
	}
}

func TestDeriveKeys_OutputSizes(t *testing.T) {
	salt := GenerateSalt()

	kek, authKey := DeriveKeys("test-vault-key", salt, KeyTypePassphrase)

	if len(kek) != KeySize {
		t.Fatalf("KEK: expected %d bytes, got %d", KeySize, len(kek))
	}
	if len(authKey) != KeySize {
		t.Fatalf("AuthKey: expected %d bytes, got %d", KeySize, len(authKey))
	}
}

func TestDeriveKeys_KEKAndAuthKeyDiffer(t *testing.T) {
	salt := GenerateSalt()

	kek, authKey := DeriveKeys("my-vault-key", salt, KeyTypePassphrase)

	if bytes.Equal(kek, authKey) {
		t.Fatal("KEK and AuthKey should not be equal")
	}
}

func TestDeriveKeys_DifferentSaltProducesDifferentKeys(t *testing.T) {
	salt1 := bytes.Repeat([]byte{0x01}, SaltSize)
	salt2 := bytes.Repeat([]byte{0x02}, SaltSize)

	kek1, _ := DeriveKeys("same-key", salt1, KeyTypePassphrase)
	kek2, _ := DeriveKeys("same-key", salt2, KeyTypePassphrase)

	if bytes.Equal(kek1, kek2) {
		t.Fatal("different salts should produce different KEKs")
	}
}

func TestDeriveKeys_DifferentVaultKeyProducesDifferentKeys(t *testing.T) {
	salt := bytes.Repeat([]byte{0xBB}, SaltSize)

	kek1, _ := DeriveKeys("key-one", salt, KeyTypePassphrase)
	kek2, _ := DeriveKeys("key-two", salt, KeyTypePassphrase)

	if bytes.Equal(kek1, kek2) {
		t.Fatal("different vault keys should produce different KEKs")
	}
}

func TestDeriveKeys_PINAndPassphraseDiffer(t *testing.T) {
	salt := bytes.Repeat([]byte{0xCC}, SaltSize)
	vaultKey := "847291"

	kekPIN, _ := DeriveKeys(vaultKey, salt, KeyTypePIN)
	kekPass, _ := DeriveKeys(vaultKey, salt, KeyTypePassphrase)

	if bytes.Equal(kekPIN, kekPass) {
		t.Fatal("PIN and passphrase params should produce different outputs for same input")
	}
}

func TestParamsForKeyType(t *testing.T) {
	pin := ParamsForKeyType(KeyTypePIN)
	if pin.Memory != 256*1024 || pin.Iterations != 10 || pin.Parallelism != 4 {
		t.Fatalf("unexpected PIN params: %+v", pin)
	}

	pass := ParamsForKeyType(KeyTypePassphrase)
	if pass.Memory != 64*1024 || pass.Iterations != 3 || pass.Parallelism != 4 {
		t.Fatalf("unexpected passphrase params: %+v", pass)
	}
}
