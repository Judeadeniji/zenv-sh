// ── Query Keys ──

export const queryKeys = {
	auth: {
		me: ["auth", "me"] as const,
	},
	orgs: {
		all: ["orgs"] as const,
		detail: (orgId: string) => ["orgs", orgId] as const,
		members: (orgId: string) => ["orgs", orgId, "members"] as const,
	},
	projects: {
		list: (orgId: string) => ["projects", orgId] as const,
		detail: (projectId: string) => ["project", projectId] as const,
	},
	recovery: {
		status: ["recovery", "status"] as const,
		request: ["recovery", "request"] as const,
	},
} as const

// ── Mutation Keys ──

export const mutationKeys = {
	auth: {
		login: ["auth", "login"] as const,
		signup: ["auth", "signup"] as const,
		setupVault: ["auth", "setup-vault"] as const,
		unlockVault: ["auth", "unlock-vault"] as const,
	},
	orgs: {
		create: ["orgs", "create"] as const,
		addMember: ["orgs", "add-member"] as const,
		removeMember: ["orgs", "remove-member"] as const,
	},
	projects: {
		create: ["projects", "create"] as const,
	},
	recovery: {
		recoverWithKit: ["recovery", "recover-kit"] as const,
		setContact: ["recovery", "set-contact"] as const,
		removeContact: ["recovery", "remove-contact"] as const,
		initiate: ["recovery", "initiate"] as const,
		cancel: ["recovery", "cancel"] as const,
		approve: ["recovery", "approve"] as const,
		complete: ["recovery", "complete"] as const,
	},
} as const

// ── Storage Keys ──

export const storageKeys = {
	nav: "zenv-nav",
	inviteToken: "zenv-invite-token",
} as const
