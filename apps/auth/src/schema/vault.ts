import { sql } from "drizzle-orm";
import {
	boolean,
	foreignKey,
	index,
	integer,
	jsonb,
	pgEnum,
	pgTable,
	text,
	timestamp,
	unique,
	uniqueIndex,
	uuid,
} from "drizzle-orm/pg-core";
import { bytea } from "../pg-custom-types.js";
import { users } from "./auth.js";
import { projectRotations, projects } from "./project.js";

// --- Enums ---
// Drizzle handles the 'CREATE TYPE' statements for these
export const vaultKeyTypeEnum = pgEnum("vault_key_type", ["pin", "passphrase"]);
export const environmentEnum = pgEnum("environment", ["development", "staging", "production"]);
export const permissionEnum = pgEnum("permission", ["read", "read_write"]);
export const recoveryStatusEnum = pgEnum("recovery_status", [
    "pending",
    "approved",
    "cancelled",
    "expired",
    "completed",
]);

// --- Identities Table ---
export const identities = pgTable("identities", {
    id: uuid().defaultRandom().primaryKey().notNull(),

    // Security & Keys
    authKeyHash: bytea("auth_key_hash").notNull(),
    salt: bytea("salt").notNull(),
    vaultKeyType: vaultKeyTypeEnum("vault_key_type").default('passphrase').notNull(),
    
    // Encryption Layers
    wrappedDek: bytea("wrapped_dek").notNull(),
    publicKey: bytea("public_key").notNull(),
    wrappedPrivateKey: bytea("wrapped_private_key").notNull(),
    
    // Recovery
    recoveryWrappedDek: bytea("recovery_wrapped_dek"),
    recoveryDisabled: boolean("recovery_disabled").default(false).notNull(),
    
    // Identity Link & Metadata
    identityId: uuid("identity_id").references(() => users.id),
    preferences: jsonb().default({}).notNull(),
    
    createdAt: timestamp("created_at", { withTimezone: true, mode: 'string' }).defaultNow().notNull(),
    updatedAt: timestamp("updated_at", { withTimezone: true, mode: 'string' }).defaultNow().notNull(),
}, (table) => [
    unique("identities_external_id_key").on(table.identityId),
]);

// --- Vault Items Table ---
export const vaultItems = pgTable(
    "vault_items",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        projectId: uuid("project_id").notNull(),
        environment: environmentEnum().notNull(),
        nameHash: bytea("name_hash").notNull(),
        ciphertext: bytea("ciphertext").notNull(),
        nonce: bytea("nonce").notNull(),
        version: integer().default(1).notNull(),
        /** Unencrypted per-item hints (MIME, labels, notes). Never put secrets here. */
        metadata: jsonb().default({}).notNull(),
        createdAt: timestamp("created_at", { withTimezone: true, mode: "string" }).defaultNow().notNull(),
        updatedAt: timestamp("updated_at", { withTimezone: true, mode: "string" }).defaultNow().notNull(),
        dekVersion: integer("dek_version").default(1).notNull(),
    },
    (table) => [
        uniqueIndex("idx_vault_items_lookup").on(
            table.projectId,
            table.environment,
            table.nameHash,
        ),
        index("idx_vault_items_project_env").on(
            table.projectId,
            table.environment,
        ),
        foreignKey({
            columns: [table.projectId],
            foreignColumns: [projects.id],
            name: "vault_items_project_id_fkey",
        }).onDelete("cascade"),
    ],
);

// --- Service Tokens Table ---
export const serviceTokens = pgTable(
    "service_tokens",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        projectId: uuid("project_id").notNull(),
        name: text().notNull(),
        tokenHash: bytea("token_hash").notNull(),
        environment: environmentEnum().notNull(),
        permission: permissionEnum().default("read").notNull(),
        createdBy: uuid("created_by"),
        expiresAt: timestamp("expires_at", { withTimezone: true, mode: "string" }),
        revokedAt: timestamp("revoked_at", { withTimezone: true, mode: "string" }),
        createdAt: timestamp("created_at", { withTimezone: true, mode: "string" }).defaultNow().notNull(),
    },
    (table) => [
        index("idx_service_tokens_hash")
            .on(table.tokenHash)
            .where(sql`(revoked_at IS NULL)`),
        index("idx_service_tokens_project").on(table.projectId),
        foreignKey({
            columns: [table.projectId],
            foreignColumns: [projects.id],
            name: "service_tokens_project_id_fkey",
        }).onDelete("cascade"),
        foreignKey({
            columns: [table.createdBy],
            foreignColumns: [users.id],
            name: "service_tokens_created_by_fkey",
        }),
        unique("service_tokens_token_hash_key").on(table.tokenHash),
    ],
);

// --- Vault Item Versions Table ---
export const vaultItemVersions = pgTable(
    "vault_item_versions",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        itemId: uuid("item_id").notNull(),
        version: integer().notNull(),
        ciphertext: bytea("ciphertext").notNull(),
        nonce: bytea("nonce").notNull(),
        createdAt: timestamp("created_at", { withTimezone: true, mode: "string" }).defaultNow().notNull(),
    },
    (table) => [
        index("idx_vault_item_versions_item").on(
            table.itemId,
            table.version,
        ),
        foreignKey({
            columns: [table.itemId],
            foreignColumns: [vaultItems.id],
            name: "vault_item_versions_item_id_fkey",
        }).onDelete("cascade"),
    ],
);

// --- Trusted Contacts Table ---
export const trustedContacts = pgTable(
    "trusted_contacts",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        userId: uuid("user_id").notNull(),
        contactUserId: uuid("contact_user_id").notNull(),
        trustedWrappedDek: bytea("trusted_wrapped_dek").notNull(),
        createdAt: timestamp("created_at", { withTimezone: true, mode: "string" }).defaultNow().notNull(),
    },
    (table) => [
        index("idx_trusted_contacts_contact").on(table.contactUserId),
        index("idx_trusted_contacts_user").on(table.userId),
        foreignKey({
            columns: [table.userId],
            foreignColumns: [users.id],
            name: "trusted_contacts_user_id_fkey",
        }).onDelete("cascade"),
        foreignKey({
            columns: [table.contactUserId],
            foreignColumns: [users.id],
            name: "trusted_contacts_contact_user_id_fkey",
        }).onDelete("cascade"),
        unique("trusted_contacts_user_id_contact_user_id_key").on(table.userId, table.contactUserId),
    ],
);

// --- Vault Item Rotations Table ---
export const vaultItemRotations = pgTable(
    "vault_item_rotations",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        projectId: uuid("project_id").notNull(),
        rotationId: uuid("rotation_id").notNull(),
        vaultItemId: uuid("vault_item_id").notNull(),
        newCiphertext: bytea("new_ciphertext").notNull(),
        newNonce: bytea("new_nonce").notNull(),
        stagedAt: timestamp("staged_at", { withTimezone: true, mode: "string" }).defaultNow().notNull(),
    },
    (table) => [
        index("idx_vault_item_rotations_rotation").on(table.rotationId),
        foreignKey({
            columns: [table.projectId],
            foreignColumns: [projects.id],
            name: "vault_item_rotations_project_id_fkey",
        }).onDelete("cascade"),
        foreignKey({
            columns: [table.rotationId],
            foreignColumns: [projectRotations.rotationId],
            name: "vault_item_rotations_rotation_id_fkey",
        }).onDelete("cascade"),
        foreignKey({
            columns: [table.vaultItemId],
            foreignColumns: [vaultItems.id],
            name: "vault_item_rotations_vault_item_id_fkey",
        }).onDelete("cascade"),
    ],
);

// --- Recovery Requests Table ---
export const recoveryRequests = pgTable(
    "recovery_requests",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        userId: uuid("user_id").notNull(),
        contactUserId: uuid("contact_user_id").notNull(),
        status: recoveryStatusEnum().default("pending").notNull(),
        recoveryPublicKey: bytea("recovery_public_key"),
        recoveryPayload: bytea("recovery_payload"),
        requestedAt: timestamp("requested_at", { withTimezone: true, mode: "string" }).defaultNow().notNull(),
        eligibleAt: timestamp("eligible_at", { withTimezone: true, mode: "string" }).notNull(),
        approvedAt: timestamp("approved_at", { withTimezone: true, mode: "string" }),
        completedAt: timestamp("completed_at", { withTimezone: true, mode: "string" }),
        cancelledAt: timestamp("cancelled_at", { withTimezone: true, mode: "string" }),
    },
    (table) => [
        uniqueIndex("idx_recovery_requests_active")
            .on(table.userId)
            .where(sql`(status = ANY (ARRAY['pending'::recovery_status, 'approved'::recovery_status]))`),
        index("idx_recovery_requests_contact")
            .on(table.contactUserId)
            .where(sql`(status = ANY (ARRAY['pending'::recovery_status, 'approved'::recovery_status]))`),
        foreignKey({
            columns: [table.userId],
            foreignColumns: [users.id],
            name: "recovery_requests_user_id_fkey",
        }).onDelete("cascade"),
        foreignKey({
            columns: [table.contactUserId],
            foreignColumns: [users.id],
            name: "recovery_requests_contact_user_id_fkey",
        }).onDelete("cascade"),
    ],
);