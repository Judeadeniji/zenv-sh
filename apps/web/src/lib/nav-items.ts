import { KeyRound, FileKey, Users, Shield, Settings, LayoutDashboard, type LucideIcon } from "lucide-react"

export interface NavItem {
	label: string
	description: string
	icon: LucideIcon
	href: string
}

/** Primary project actions — the daily workflow */
export function getProjectItems(orgId: string, projectId: string): NavItem[] {
	return [
		{
			label: "Overview",
			description: "Project dashboard, key, and quick start.",
			icon: LayoutDashboard,
			href: `/orgs/${orgId}/projects/${projectId}`,
		},
		{
			label: "Secrets",
			description: "Store and manage encrypted secrets for your apps.",
			icon: KeyRound,
			href: `/orgs/${orgId}/projects/${projectId}/secrets`,
		},
		{
			label: "Service Tokens",
			description: "Generate tokens so your CI/CD pipelines can access secrets.",
			icon: FileKey,
			href: `/orgs/${orgId}/projects/${projectId}/tokens`,
		},
	]
}

/** Org-wide items — team and monitoring */
export function getOrgItems(orgId: string, projectId?: string): NavItem[] {
	const items: NavItem[] = [
		{
			label: "Members",
			description: "Invite your team to collaborate. Everyone gets their own vault.",
			icon: Users,
			href: `/orgs/${orgId}/members`,
		},
	]

	if (projectId) {
		items.push({
			label: "Audit Log",
			description: "A record of every action in your project.",
			icon: Shield,
			href: `/orgs/${orgId}/projects/${projectId}/audit`,
		})
	}

	return items
}

/** Settings — always last, lowest frequency */
export function getSettingsItems(orgId: string, projectId?: string): NavItem[] {
	const items: NavItem[] = []

	if (projectId) {
		items.push({
			label: "Project Settings",
			description: "Project key, configuration, and danger zone.",
			icon: Settings,
			href: `/orgs/${orgId}/projects/${projectId}/settings`,
		})
	}

	return items
}
