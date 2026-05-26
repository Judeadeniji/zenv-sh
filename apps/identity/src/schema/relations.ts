import { relations } from "drizzle-orm/relations"
import { organizations, users } from "./auth.js"
import { projectKeyGrants, projectRotations, projectVaultKeys, projects } from "./project.js"
import {
	recoveryRequests,
	serviceTokens,
	trustedContacts,
	vaultItemRotations,
	vaultItemVersions,
	vaultItems,
} from "./vault.js"

export const usersVaultRelations = relations(users, ({ many }) => ({
	recoveryRequests_userId: many(recoveryRequests, {
		relationName: "recoveryRequests_userId_users_id",
	}),
	recoveryRequests_contactUserId: many(recoveryRequests, {
		relationName: "recoveryRequests_contactUserId_users_id",
	}),
	projectRotations: many(projectRotations),
	projectKeyGrants: many(projectKeyGrants),
	serviceTokens: many(serviceTokens),
	trustedContacts_userId: many(trustedContacts, {
		relationName: "trustedContacts_userId_users_id",
	}),
	trustedContacts_contactUserId: many(trustedContacts, {
		relationName: "trustedContacts_contactUserId_users_id",
	}),
}))

export const organizationsVaultRelations = relations(organizations, ({ many }) => ({
	projects: many(projects),
}))

export const projectsRelations = relations(projects, ({ one, many }) => ({
	organization: one(organizations, {
		fields: [projects.organizationId],
		references: [organizations.id],
	}),
	projectVaultKeys: many(projectVaultKeys),
	projectRotations: many(projectRotations),
	projectKeyGrants: many(projectKeyGrants),
	vaultItems: many(vaultItems),
	serviceTokens: many(serviceTokens),
	vaultItemRotations: many(vaultItemRotations),
}))

export const recoveryRequestsRelations = relations(recoveryRequests, ({ one }) => ({
	user_userId: one(users, {
		fields: [recoveryRequests.userId],
		references: [users.id],
		relationName: "recoveryRequests_userId_users_id",
	}),
	user_contactUserId: one(users, {
		fields: [recoveryRequests.contactUserId],
		references: [users.id],
		relationName: "recoveryRequests_contactUserId_users_id",
	}),
}))

export const projectVaultKeysRelations = relations(projectVaultKeys, ({ one }) => ({
	project: one(projects, {
		fields: [projectVaultKeys.projectId],
		references: [projects.id],
	}),
}))

export const projectRotationsRelations = relations(projectRotations, ({ one, many }) => ({
	project: one(projects, {
		fields: [projectRotations.projectId],
		references: [projects.id],
	}),
	user: one(users, {
		fields: [projectRotations.initiatedBy],
		references: [users.id],
	}),
	vaultItemRotations: many(vaultItemRotations),
}))

export const projectKeyGrantsRelations = relations(projectKeyGrants, ({ one }) => ({
	project: one(projects, {
		fields: [projectKeyGrants.projectId],
		references: [projects.id],
	}),
	user: one(users, {
		fields: [projectKeyGrants.userId],
		references: [users.id],
	}),
}))

export const vaultItemsRelations = relations(vaultItems, ({ one, many }) => ({
	project: one(projects, {
		fields: [vaultItems.projectId],
		references: [projects.id],
	}),
	vaultItemVersions: many(vaultItemVersions),
	vaultItemRotations: many(vaultItemRotations),
}))

export const serviceTokensRelations = relations(serviceTokens, ({ one }) => ({
	project: one(projects, {
		fields: [serviceTokens.projectId],
		references: [projects.id],
	}),
	user: one(users, {
		fields: [serviceTokens.createdBy],
		references: [users.id],
	}),
}))

export const vaultItemVersionsRelations = relations(vaultItemVersions, ({ one }) => ({
	vaultItem: one(vaultItems, {
		fields: [vaultItemVersions.itemId],
		references: [vaultItems.id],
	}),
}))

export const trustedContactsRelations = relations(trustedContacts, ({ one }) => ({
	user_userId: one(users, {
		fields: [trustedContacts.userId],
		references: [users.id],
		relationName: "trustedContacts_userId_users_id",
	}),
	user_contactUserId: one(users, {
		fields: [trustedContacts.contactUserId],
		references: [users.id],
		relationName: "trustedContacts_contactUserId_users_id",
	}),
}))

export const vaultItemRotationsRelations = relations(vaultItemRotations, ({ one }) => ({
	project: one(projects, {
		fields: [vaultItemRotations.projectId],
		references: [projects.id],
	}),
	projectRotation: one(projectRotations, {
		fields: [vaultItemRotations.rotationId],
		references: [projectRotations.rotationId],
	}),
	vaultItem: one(vaultItems, {
		fields: [vaultItemRotations.vaultItemId],
		references: [vaultItems.id],
	}),
}))
