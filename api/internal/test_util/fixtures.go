package test_util

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/amnesia"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

// IdentityUser represents a test identity user with session.
type IdentityUser struct {
	IdentityID   string
	Email        string
	SessionToken string
}

// CreateIdentityUser registers a fake Better Auth session on the test auth mock.
func CreateIdentityUser(t *testing.T, ts *TestServer) IdentityUser {
	t.Helper()
	if ts.Auth == nil {
		t.Fatal("CreateIdentityUser: ts.Auth is nil")
	}

	id := uuid.New().String()
	email := fmt.Sprintf("test-%s@test.zenv.sh", uuid.New().String()[:8])
	token := "tok-" + uuid.New().String()

	ts.Auth.RegisterSession(token, id, email, "Test User", time.Now().Add(24*time.Hour))

	authUID, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("parse identity id: %v", err)
	}
	now := time.Now().UTC()
	if _, err := table.Users.INSERT(
		table.Users.ID,
		table.Users.Name,
		table.Users.Email,
		table.Users.EmailVerified,
		table.Users.CreatedAt,
		table.Users.UpdatedAt,
	).VALUES(
		authUID,
		"Test User",
		email,
		Bool(false),
		now,
		now,
	).Exec(ts.DB); err != nil {
		t.Fatalf("insert users row: %v", err)
	}

	return IdentityUser{
		IdentityID:   id,
		Email:        email,
		SessionToken: token,
	}
}

// CreateExpiredIdentityUser creates an identity user with an expired session.
func CreateExpiredIdentityUser(t *testing.T, ts *TestServer) IdentityUser {
	t.Helper()
	u := CreateIdentityUser(t, ts)
	ts.Auth.RegisterSession(u.SessionToken, u.IdentityID, u.Email, "Test User", time.Now().Add(-1*time.Hour))
	return u
}

// ZenvUser represents a test zEnv user with crypto material.
type ZenvUser struct {
	UserID   uuid.UUID
	VaultKey string
	AuthKey  []byte
}

// CreateZenvUser creates a vault identity linked to a Better Auth user.
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

	identityUUID := uuid.New()
	now := time.Now().UTC()

	// Using Identities table from your refactored schema
	_, err = table.Identities.INSERT(
		table.Identities.ID,
		table.Identities.AuthKeyHash,
		table.Identities.VaultKeyType,
		table.Identities.Salt,
		table.Identities.WrappedDek,
		table.Identities.PublicKey,
		table.Identities.WrappedPrivateKey,
		table.Identities.IdentityID,
		table.Identities.CreatedAt,
		table.Identities.UpdatedAt,
	).VALUES(
		identityUUID,
		authKeyHash,
		"passphrase", // Use the generated Enum type
		salt,
		wrappedDEKFull,
		pubKey,
		wrappedPrivKeyFull,
		identityID,
		now,
		now,
	).Exec(db)
	if err != nil {
		t.Fatalf("insert identities row: %v", err)
	}

	return ZenvUser{
		UserID:   identityUUID,
		VaultKey: vaultKey,
		AuthKey:  authKey,
	}
}

// CreateProject creates an org + project + vault key.
// memberUserID must be the identity-provider user id (Better Auth users.id), not the vault identities.id row.
func CreateProject(t *testing.T, db *sql.DB, memberUserID uuid.UUID) (orgID, projectID uuid.UUID) {
	t.Helper()

	orgID = uuid.New()
	now := time.Now().UTC()
	_, err := table.Organizations.INSERT(
		table.Organizations.ID,
		table.Organizations.Name,
		table.Organizations.Slug,
		table.Organizations.CreatedAt,
		table.Organizations.OwnerID,
	).VALUES(
		orgID.String(),
		"TestOrg-"+uuid.New().String()[:8],
		"test-org-"+uuid.New().String()[:8],
		now,
		memberUserID.String(),
	).Exec(db)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Insert membership using the 'Member' table from Drizzle schema
	_, err = table.Members.INSERT(
		table.Members.ID,
		table.Members.OrganizationID,
		table.Members.UserID,
		table.Members.Role,
		table.Members.CreatedAt,
	).VALUES(
		uuid.New().String(),
		orgID.String(),
		memberUserID.String(),
		"admin",
		now,
	).Exec(db)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}

	projectID = uuid.New()
	_, err = table.Projects.INSERT(
		table.Projects.ID,
		table.Projects.OrganizationID,
		table.Projects.Name,
		table.Projects.CreatedAt,
	).VALUES(
		projectID,
		orgID.String(),
		"TestProj-"+uuid.New().String()[:8],
		now,
	).Exec(db)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}

	projectSalt := amnesia.GenerateSalt()
	projectDEK := amnesia.GenerateKey()
	projectKEK, _ := amnesia.DeriveKeys("project-vault-key", projectSalt, amnesia.KeyTypePassphrase)
	wrappedPDEK, pdNonce, err := amnesia.WrapKey(projectDEK, projectKEK)
	if err != nil {
		t.Fatalf("wrap project DEK: %v", err)
	}
	wrappedPDEKFull := append(pdNonce, wrappedPDEK...)

	_, err = table.ProjectVaultKeys.INSERT(
		table.ProjectVaultKeys.ID,
		table.ProjectVaultKeys.ProjectID,
		table.ProjectVaultKeys.ProjectSalt,
		table.ProjectVaultKeys.WrappedProjectDek,
		table.ProjectVaultKeys.CreatedAt,
	).VALUES(
		uuid.New(),
		projectID,
		projectSalt,
		wrappedPDEKFull,
		now,
	).Exec(db)
	if err != nil {
		t.Fatalf("insert project vault key: %v", err)
	}

	return orgID, projectID
}

// CreateServiceToken creates a service token and returns the plaintext.
func CreateServiceToken(t *testing.T, db *sql.DB, projectID uuid.UUID, env, permission string) string {
	t.Helper()

	tokenPlaintext := fmt.Sprintf("ze_%s_%s", env, hex.EncodeToString(amnesia.GenerateKey()))
	hash := sha256.Sum256([]byte(tokenPlaintext))

	_, err := table.ServiceTokens.INSERT(
		table.ServiceTokens.ID,
		table.ServiceTokens.ProjectID,
		table.ServiceTokens.Name,
		table.ServiceTokens.TokenHash,
		table.ServiceTokens.Environment,
		table.ServiceTokens.Permission,
	).VALUES(
		uuid.New(),
		projectID,
		"test-token-"+uuid.New().String()[:8],
		hash[:],
		env,
		permission,
	).Exec(db)
	if err != nil {
		t.Fatalf("insert service token: %v", err)
	}

	return tokenPlaintext
}
