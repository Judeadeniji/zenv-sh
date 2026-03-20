package testutil

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/amnesia"
)

// IdentityUser represents a test identity user with session.
type IdentityUser struct {
	IdentityID   string
	Email        string
	SessionToken string
}

// CreateIdentityUser inserts a user + session into the identity tables.
func CreateIdentityUser(t *testing.T, db *sql.DB) IdentityUser {
	t.Helper()

	id := "id-" + uuid.New().String()[:8]
	email := fmt.Sprintf("test-%s@test.zenv.sh", uuid.New().String()[:8])
	token := "tok-" + uuid.New().String()

	_, err := db.Exec(
		`INSERT INTO "user" (id, name, email) VALUES ($1, $2, $3)`,
		id, "Test User", email,
	)
	if err != nil {
		t.Fatalf("insert identity user: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO "session" (id, token, user_id, expires_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())`,
		"sess-"+uuid.New().String()[:8], token, id,
		time.Now().Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("insert identity session: %v", err)
	}

	return IdentityUser{
		IdentityID:   id,
		Email:        email,
		SessionToken: token,
	}
}

// CreateExpiredIdentityUser creates an identity user with an expired session.
func CreateExpiredIdentityUser(t *testing.T, db *sql.DB) IdentityUser {
	t.Helper()

	id := "id-" + uuid.New().String()[:8]
	email := fmt.Sprintf("test-%s@test.zenv.sh", uuid.New().String()[:8])
	token := "tok-" + uuid.New().String()

	_, err := db.Exec(
		`INSERT INTO "user" (id, name, email) VALUES ($1, $2, $3)`,
		id, "Expired User", email,
	)
	if err != nil {
		t.Fatalf("insert identity user: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO "session" (id, token, user_id, expires_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())`,
		"sess-"+uuid.New().String()[:8], token, id,
		time.Now().Add(-1*time.Hour),
	)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	return IdentityUser{
		IdentityID:   id,
		Email:        email,
		SessionToken: token,
	}
}

// ZenvUser represents a test zEnv user with crypto material.
type ZenvUser struct {
	UserID   uuid.UUID
	VaultKey string // plaintext vault key for unlock tests
	AuthKey  []byte // raw auth key (before hashing) for unlock tests
}

// CreateZenvUser creates a zEnv user linked to an identity ID,
// using real Amnesia crypto.
func CreateZenvUser(t *testing.T, db *sql.DB, identityID, email string) ZenvUser {
	t.Helper()

	vaultKey := "test-vault-key-" + uuid.New().String()[:8]
	salt := amnesia.GenerateSalt()
	kek, authKey := amnesia.DeriveKeys(vaultKey, salt, amnesia.KeyTypePassphrase)

	authKeyHash := amnesia.HashAuthKey(authKey)

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

	userID := uuid.New()
	now := time.Now().UTC()

	// Schema: users(id, email, auth_key_hash, vault_key_type, salt,
	//   wrapped_dek, public_key, wrapped_private_key, created_at, updated_at, identity_id)
	_, err = db.Exec(
		`INSERT INTO users (id, email, auth_key_hash, vault_key_type, salt,
		 wrapped_dek, public_key, wrapped_private_key, identity_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		userID, email, authKeyHash, "passphrase", salt,
		wrappedDEKFull, pubKey, wrappedPrivKeyFull, identityID, now, now,
	)
	if err != nil {
		t.Fatalf("insert zenv user: %v", err)
	}

	return ZenvUser{
		UserID:   userID,
		VaultKey: vaultKey,
		AuthKey:  authKey,
	}
}

// CreateProject creates an org + project + vault key for testing.
func CreateProject(t *testing.T, db *sql.DB, ownerID uuid.UUID) (orgID, projectID uuid.UUID) {
	t.Helper()

	// Schema: organizations(id, name, owner_id, created_at)
	orgID = uuid.New()
	_, err := db.Exec(
		`INSERT INTO organizations (id, name, owner_id) VALUES ($1, $2, $3)`,
		orgID, "TestOrg-"+uuid.New().String()[:8], ownerID,
	)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Schema: organization_members(id, organization_id, user_id, role, joined_at)
	_, err = db.Exec(
		`INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1, $2, 'admin')`,
		orgID, ownerID,
	)
	if err != nil {
		t.Fatalf("insert org member: %v", err)
	}

	// Schema: projects(id, organization_id, name, created_at)
	projectID = uuid.New()
	_, err = db.Exec(
		`INSERT INTO projects (id, organization_id, name) VALUES ($1, $2, $3)`,
		projectID, orgID, "TestProj-"+uuid.New().String()[:8],
	)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}

	// Schema: project_vault_keys(id, project_id, project_salt, wrapped_project_dek, created_at)
	projectSalt := amnesia.GenerateSalt()
	projectDEK := amnesia.GenerateKey()
	projectKEK, _ := amnesia.DeriveKeys("project-vault-key", projectSalt, amnesia.KeyTypePassphrase)
	wrappedPDEK, pdNonce, err := amnesia.WrapKey(projectDEK, projectKEK)
	if err != nil {
		t.Fatalf("wrap project DEK: %v", err)
	}
	wrappedPDEKFull := append(pdNonce, wrappedPDEK...)

	_, err = db.Exec(
		`INSERT INTO project_vault_keys (project_id, project_salt, wrapped_project_dek) VALUES ($1, $2, $3)`,
		projectID, projectSalt, wrappedPDEKFull,
	)
	if err != nil {
		t.Fatalf("insert project vault key: %v", err)
	}

	return orgID, projectID
}

// CreateServiceToken creates a service token and returns the plaintext.
// Schema: service_tokens(id, project_id, name, token_hash, environment, permission, created_by, expires_at, revoked_at, created_at)
func CreateServiceToken(t *testing.T, db *sql.DB, projectID uuid.UUID, env, permission string) string {
	t.Helper()

	tokenPlaintext := fmt.Sprintf("ze_%s_%s", env, hex.EncodeToString(amnesia.GenerateKey()))
	hash := sha256.Sum256([]byte(tokenPlaintext))

	_, err := db.Exec(
		`INSERT INTO service_tokens (project_id, name, token_hash, environment, permission) VALUES ($1, $2, $3, $4, $5)`,
		projectID, "test-token-"+uuid.New().String()[:8], hash[:], env, permission,
	)
	if err != nil {
		t.Fatalf("insert service token: %v", err)
	}

	return tokenPlaintext
}
