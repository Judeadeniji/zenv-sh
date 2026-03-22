import { KeyRound, FileKey, Users, Shield, type LucideIcon } from "lucide-react"

export interface NavItem {
	label: string
	description: string
	icon: LucideIcon
	href: string
}

/** Primary manage nav items — single source of truth for sidebar + dashboard. */
export const manageItems: NavItem[] = [
	{
		label: "Secrets",
		description: "Store and manage encrypted environment variables for your apps.",
		icon: KeyRound,
		href: "/secrets",
	},
	{
		label: "Service Tokens",
		description: "Generate tokens so your CI/CD pipelines can access secrets.",
		icon: FileKey,
		href: "/tokens",
	},
	{
		label: "Members",
		description: "Invite your team to collaborate. Everyone gets their own vault.",
		icon: Users,
		href: "/members",
	},
	{
		label: "Audit Log",
		description: "A record of every action in your organization.",
		icon: Shield,
		href: "/audit",
	},
]
