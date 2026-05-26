import {
    foreignKey,
    index,
    integer,
    pgEnum,
    pgTable,
    text,
    timestamp,
    unique,
    uuid,
} from "drizzle-orm/pg-core";
import { bytea } from "../pg-custom-types";
import { organizations, users } from "./auth";

// --- Enums ---
export const rotationStatusEnum = pgEnum("rotation_status", [
    "staging",
    "committing",
    "complete",
    "failed",
]);

// --- Projects Table ---
export const projects = pgTable(
    "projects",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        organizationId: uuid("organization_id").notNull(),
        name: text().notNull(),
        createdAt: timestamp("created_at", { withTimezone: true, mode: "string" })
            .defaultNow()
            .notNull(),
    },
    (table) => [
        foreignKey({
            columns: [table.organizationId],
            foreignColumns: [organizations.id],
            name: "projects_organization_id_fkey",
        }).onDelete("cascade"),
        unique("projects_organization_id_name_key").on(table.organizationId, table.name),
    ],
);

// --- Project Vault Keys Table ---
export const projectVaultKeys = pgTable(
    "project_vault_keys",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        projectId: uuid("project_id").notNull(),
        projectSalt: bytea("project_salt").notNull(),
        wrappedProjectDek: bytea("wrapped_project_dek").notNull(),
        createdAt: timestamp("created_at", { withTimezone: true, mode: "string" })
            .defaultNow()
            .notNull(),
        dekVersion: integer("dek_version").default(1).notNull(),
    },
    (table) => [
        foreignKey({
            columns: [table.projectId],
            foreignColumns: [projects.id],
            name: "project_vault_keys_project_id_fkey",
        }).onDelete("cascade"),
        unique("project_vault_keys_project_id_key").on(table.projectId),
    ],
);

// --- Project Rotations Table ---
export const projectRotations = pgTable(
    "project_rotations",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        projectId: uuid("project_id").notNull(),
        rotationId: uuid("rotation_id").notNull(),
        status: rotationStatusEnum().default("staging").notNull(), // Swapped to Enum
        totalItems: integer("total_items").notNull(),
        stagedItems: integer("staged_items").default(0).notNull(),
        initiatedBy: uuid("initiated_by").notNull(),
        startedAt: timestamp("started_at", { withTimezone: true, mode: "string" })
            .defaultNow()
            .notNull(),
        completedAt: timestamp("completed_at", { withTimezone: true, mode: "string" }),
    },
    (table) => [
        // Removed .using("btree", ...) and .op("uuid_ops")
        index("idx_project_rotations_project").on(table.projectId),
        foreignKey({
            columns: [table.projectId],
            foreignColumns: [projects.id],
            name: "project_rotations_project_id_fkey",
        }).onDelete("cascade"),
        foreignKey({
            columns: [table.initiatedBy],
            foreignColumns: [users.id],
            name: "project_rotations_initiated_by_fkey",
        }),
        unique("project_rotations_rotation_id_key").on(table.rotationId),
    ],
);

// --- Project Key Grants Table ---
export const projectKeyGrants = pgTable(
    "project_key_grants",
    {
        id: uuid().defaultRandom().primaryKey().notNull(),
        projectId: uuid("project_id").notNull(),
        userId: uuid("user_id").notNull(),
        wrappedProjectVaultKey: bytea("wrapped_project_vault_key").notNull(),
        grantedAt: timestamp("granted_at", { withTimezone: true, mode: "string" })
            .defaultNow()
            .notNull(),
    },
    (table) => [
        foreignKey({
            columns: [table.projectId],
            foreignColumns: [projects.id],
            name: "project_key_grants_project_id_fkey",
        }).onDelete("cascade"),
        foreignKey({
            columns: [table.userId],
            foreignColumns: [users.id],
            name: "project_key_grants_user_id_fkey",
        }).onDelete("cascade"),
        unique("project_key_grants_project_id_user_id_key").on(table.projectId, table.userId),
    ],
);
