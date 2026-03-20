package handler_test

import (
	"encoding/base64"
	"testing"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/api/internal/testutil"
)

func TestMe_NoVaultSetup(t *testing.T) {
	identity := testutil.CreateIdentityUser(t, ts.DB)

	resp := doReq(t, "GET", ts.URL+"/v1/auth/me", nil, identity.SessionToken)
	assertStatus(t, resp, 200)

	var body struct {
		Email              string `json:"email"`
		VaultSetupComplete bool   `json:"vault_setup_complete"`
		VaultUnlocked      bool   `json:"vault_unlocked"`
	}
	decodeJSON(t, resp, &body)

	if body.Email != identity.Email {
		t.Errorf("email = %q, want %q", body.Email, identity.Email)
	}
	if body.VaultSetupComplete {
		t.Error("vault_setup_complete should be false for identity-only user")
	}
	if body.VaultUnlocked {
		t.Error("vault_unlocked should be false")
	}
}

func TestMe_WithVaultSetup(t *testing.T) {
	identity := testutil.CreateIdentityUser(t, ts.DB)
	testutil.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)

	resp := doReq(t, "GET", ts.URL+"/v1/auth/me", nil, identity.SessionToken)
	assertStatus(t, resp, 200)

	var body struct {
		Email              string `json:"email"`
		VaultSetupComplete bool   `json:"vault_setup_complete"`
		VaultKeyType       string `json:"vault_key_type"`
		Salt               string `json:"salt"`
	}
	decodeJSON(t, resp, &body)

	if !body.VaultSetupComplete {
		t.Error("vault_setup_complete should be true after CreateZenvUser")
	}
	if body.VaultKeyType == "" {
		t.Error("vault_key_type should not be empty")
	}
	if body.Salt == "" {
		t.Error("salt should not be empty")
	}
}

func TestMe_NoSession(t *testing.T) {
	resp := doReq(t, "GET", ts.URL+"/v1/auth/me", nil, "")
	assertStatus(t, resp, 401)
}

func TestSetupVault_Success(t *testing.T) {
	identity := testutil.CreateIdentityUser(t, ts.DB)

	// Generate real crypto material.
	vaultKey := "test-setup-vault-key"
	salt := amnesia.GenerateSalt()
	kek, authKey := amnesia.DeriveKeys(vaultKey, salt, amnesia.KeyTypePassphrase)

	dek := amnesia.GenerateKey()
	wrappedDEK, dekNonce, err := amnesia.WrapKey(dek, kek)
	if err != nil {
		t.Fatalf("wrap DEK: %v", err)
	}
	wrappedDEKFull := append(dekNonce, wrappedDEK...)

	pubKey, privKey, err := amnesia.GenerateKeypair()
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}
	wrappedPrivKey, privNonce, err := amnesia.Encrypt(privKey, dek)
	if err != nil {
		t.Fatalf("encrypt private key: %v", err)
	}
	wrappedPrivKeyFull := append(privNonce, wrappedPrivKey...)

	reqBody := jsonBody{
		"vault_key_type":      "passphrase",
		"salt":                base64.StdEncoding.EncodeToString(salt),
		"auth_key_hash":       base64.StdEncoding.EncodeToString(authKey),
		"wrapped_dek":         base64.StdEncoding.EncodeToString(wrappedDEKFull),
		"public_key":          base64.StdEncoding.EncodeToString(pubKey),
		"wrapped_private_key": base64.StdEncoding.EncodeToString(wrappedPrivKeyFull),
	}

	resp := doReq(t, "POST", ts.URL+"/v1/auth/setup-vault", reqBody, identity.SessionToken)
	assertStatus(t, resp, 201)

	var body struct {
		UserID             string `json:"user_id"`
		VaultSetupComplete bool   `json:"vault_setup_complete"`
	}
	decodeJSON(t, resp, &body)

	if body.UserID == "" {
		t.Error("user_id should not be empty")
	}
	if !body.VaultSetupComplete {
		t.Error("vault_setup_complete should be true")
	}
}

func TestSetupVault_Duplicate(t *testing.T) {
	identity := testutil.CreateIdentityUser(t, ts.DB)
	testutil.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)

	// Try to set up vault again.
	salt := amnesia.GenerateSalt()
	kek, authKey := amnesia.DeriveKeys("dup-key", salt, amnesia.KeyTypePassphrase)
	dek := amnesia.GenerateKey()
	wrappedDEK, dekNonce, _ := amnesia.WrapKey(dek, kek)
	wrappedDEKFull := append(dekNonce, wrappedDEK...)
	pubKey, privKey, _ := amnesia.GenerateKeypair()
	wrappedPrivKey, privNonce, _ := amnesia.Encrypt(privKey, dek)
	wrappedPrivKeyFull := append(privNonce, wrappedPrivKey...)

	reqBody := jsonBody{
		"vault_key_type":      "passphrase",
		"salt":                base64.StdEncoding.EncodeToString(salt),
		"auth_key_hash":       base64.StdEncoding.EncodeToString(authKey),
		"wrapped_dek":         base64.StdEncoding.EncodeToString(wrappedDEKFull),
		"public_key":          base64.StdEncoding.EncodeToString(pubKey),
		"wrapped_private_key": base64.StdEncoding.EncodeToString(wrappedPrivKeyFull),
	}

	resp := doReq(t, "POST", ts.URL+"/v1/auth/setup-vault", reqBody, identity.SessionToken)
	assertStatus(t, resp, 409)
}

func TestSetupVault_MissingFields(t *testing.T) {
	identity := testutil.CreateIdentityUser(t, ts.DB)

	// Omit salt.
	reqBody := jsonBody{
		"vault_key_type":      "passphrase",
		"auth_key_hash":       base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"wrapped_dek":         base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"public_key":          base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"wrapped_private_key": base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
	}

	resp := doReq(t, "POST", ts.URL+"/v1/auth/setup-vault", reqBody, identity.SessionToken)
	assertStatus(t, resp, 400)
}

func TestSetupVault_InvalidKeyType(t *testing.T) {
	identity := testutil.CreateIdentityUser(t, ts.DB)

	reqBody := jsonBody{
		"vault_key_type":      "biometric",
		"salt":                base64.StdEncoding.EncodeToString(amnesia.GenerateSalt()),
		"auth_key_hash":       base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"wrapped_dek":         base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"public_key":          base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
		"wrapped_private_key": base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
	}

	resp := doReq(t, "POST", ts.URL+"/v1/auth/setup-vault", reqBody, identity.SessionToken)
	assertStatus(t, resp, 400)
}

func TestUnlock_CorrectKey(t *testing.T) {
	identity := testutil.CreateIdentityUser(t, ts.DB)
	zenvUser := testutil.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)

	// The unlock endpoint expects the raw auth key (base64-encoded), NOT the hash.
	// The server hashes it with amnesia.HashAuthKey() and compares.
	reqBody := jsonBody{
		"auth_key_hash": base64.StdEncoding.EncodeToString(zenvUser.AuthKey),
	}

	// Use both Bearer header AND cookie so the server can store vault state in Redis.
	resp := doReqWithCookie(t, "POST", ts.URL+"/v1/auth/unlock", reqBody, identity.SessionToken)
	assertStatus(t, resp, 200)

	var body struct {
		WrappedDEK        string `json:"wrapped_dek"`
		WrappedPrivateKey string `json:"wrapped_private_key"`
		PublicKey         string `json:"public_key"`
	}
	decodeJSON(t, resp, &body)

	if body.WrappedDEK == "" {
		t.Error("wrapped_dek should not be empty")
	}
	if body.WrappedPrivateKey == "" {
		t.Error("wrapped_private_key should not be empty")
	}
	if body.PublicKey == "" {
		t.Error("public_key should not be empty")
	}
}

func TestUnlock_WrongKey(t *testing.T) {
	identity := testutil.CreateIdentityUser(t, ts.DB)
	testutil.CreateZenvUser(t, ts.DB, identity.IdentityID, identity.Email)

	// Submit a random auth key that does not match.
	reqBody := jsonBody{
		"auth_key_hash": base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
	}

	resp := doReqWithCookie(t, "POST", ts.URL+"/v1/auth/unlock", reqBody, identity.SessionToken)
	assertStatus(t, resp, 403)
}

func TestUnlock_NoUser(t *testing.T) {
	identity := testutil.CreateIdentityUser(t, ts.DB)
	// No zEnv user created — only identity exists.

	reqBody := jsonBody{
		"auth_key_hash": base64.StdEncoding.EncodeToString(amnesia.GenerateKey()),
	}

	resp := doReqWithCookie(t, "POST", ts.URL+"/v1/auth/unlock", reqBody, identity.SessionToken)
	assertStatus(t, resp, 404)
}
