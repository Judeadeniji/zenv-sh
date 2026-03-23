import { KeyRound, FileKey, Users, Shield, LayoutDashboard, type LucideIcon } from "lucide-react"

export interface NavItem {
	label: string
	description: string
	icon: LucideIcon
	href: string
}

/** Project-scoped nav items — require both orgId and projectId */
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
		{
			label: "Audit Log",
			description: "A record of every action in your project.",
			icon: Shield,
			href: `/orgs/${orgId}/projects/${projectId}/audit`,
		},
	]
}

/** Org-scoped nav items — require orgId only */
export function getOrgItems(orgId: string): NavItem[] {
	return [
		{
			label: "Members",
			description: "Invite your team to collaborate. Everyone gets their own vault.",
			icon: Users,
			href: `/orgs/${orgId}/members`,
		},
	]
}
