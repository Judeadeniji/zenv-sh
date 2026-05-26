// ── Query Keys ──

export const queryKeys = {
	auth: {
		me: ["identity", "me"] as const,
	},
	orgs: {
		all: ["orgs"] as const,
		list: (params?: Record<string, unknown>) => [...queryKeys.orgs.all, "list", params] as const,
		detail: (orgId: string) => ["orgs", orgId] as const,
		members: (orgId: string, params?: Record<string, unknown>) =>
			["orgs", orgId, "members", params] as const,
		invitations: (orgId: string, params?: Record<string, unknown>) =>
			["orgs", orgId, "invitations", params] as const,
	},
	projects: {
		list: (orgId: string, params?: Record<string, unknown>) => ["projects", orgId, params] as const,
		detail: (projectId: string) => ["project", projectId] as const,
		stats: (projectId: string) => ["project", projectId, "stats"] as const,
	},
	secrets: {
		list: (projectId: string, params?: Record<string, unknown>) =>
			["secrets", projectId, params] as const,
		detail: (projectId: string, nameHash: string) => ["secrets", projectId, nameHash] as const,
		/** Decrypted payload for one secret (lazy fetch; share prefix with list for invalidation). */
		payload: (projectId: string, environment: string, nameHash: string) =>
			[...queryKeys.secrets.list(projectId), environment, "payload", nameHash] as const,
		versions: (projectId: string, nameHash: string) =>
			["secrets", projectId, nameHash, "versions"] as const,
	},
	tokens: {
		list: (projectId: string, params?: Record<string, unknown>) =>
			["tokens", projectId, params] as const,
	},
	members: {
		list: (orgId: string, params?: Record<string, unknown>) => ["members", orgId, params] as const,
	},
	audit: {
		list: (projectId: string, params?: Record<string, unknown>) =>
			["audit", projectId, params] as const,
	},
	recovery: {
		status: ["recovery", "status"] as const,
		request: ["recovery", "request"] as const,
		incomingRequests: ["recovery", "incoming-requests"] as const,
	},
	preferences: ["preferences"] as const,
} as const

// ── Mutation Keys ──

export const mutationKeys = {
	auth: {
		login: ["identity", "login"] as const,
		signup: ["identity", "signup"] as const,
		setupVault: ["identity", "setup-vault"] as const,
		unlockVault: ["identity", "unlock-vault"] as const,
		updateProfile: ["identity", "update-profile"] as const,
		changePassword: ["identity", "change-password"] as const,
		linkSocial: ["identity", "link-social"] as const,
		toggleTwoFactor: ["identity", "toggle-2fa"] as const,
		changeVaultKey: ["identity", "change-vault-key"] as const,
		verifyMnemonic: ["identity", "verify-mnemonic"] as const,
	},
	orgs: {
		create: ["orgs", "create"] as const,
		rename: ["orgs", "rename"] as const,
		delete: ["orgs", "delete"] as const,
		addMember: ["orgs", "add-member"] as const,
		removeMember: ["orgs", "remove-member"] as const,
	},
	projects: {
		create: ["projects", "create"] as const,
		delete: ["projects", "delete"] as const,
	},
	rotation: {
		start: ["rotation", "start"] as const,
		stage: ["rotation", "stage"] as const,
		commit: ["rotation", "commit"] as const,
		cancel: ["rotation", "cancel"] as const,
	},
	secrets: {
		create: ["secrets", "create"] as const,
		update: ["secrets", "update"] as const,
		delete: ["secrets", "delete"] as const,
		rollback: ["secrets", "rollback"] as const,
		patchMetadata: ["secrets", "patch-metadata"] as const,
	},
	tokens: {
		create: ["tokens", "create"] as const,
		revoke: ["tokens", "revoke"] as const,
		destroy: ["tokens", "destroy"] as const,
	},
	recovery: {
		recoverWithKit: ["recovery", "recover-kit"] as const,
		regenerateKit: ["recovery", "regenerate-kit"] as const,
		verifyVaultKey: ["recovery", "verify-vault-key"] as const,
		toggleDisable: ["recovery", "toggle-disable"] as const,
		setContact: ["recovery", "set-contact"] as const,
		removeContact: ["recovery", "remove-contact"] as const,
		initiate: ["recovery", "initiate"] as const,
		cancel: ["recovery", "cancel"] as const,
		approve: ["recovery", "approve"] as const,
		complete: ["recovery", "complete"] as const,
	},
	invite: {
		accept: ["invite", "accept"] as const,
	},
	preferences: {
		update: ["preferences", "update"] as const,
	},
} as const

// ── Storage Keys ──

export const storageKeys = {
	nav: "zenv-nav",
	inviteToken: "zenv-invite-token",
} as const
