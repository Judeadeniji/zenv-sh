import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { z } from "zod"
import { cn } from "#/lib/utils"
import { Tabs, TabsList, TabsTrigger } from "#/components/ui/tabs"
import { AccountSection } from "#/components/settings/account-section"
import { RecoverySection } from "#/components/settings/recovery-section"
import { OrgSection } from "#/components/settings/org-section"
import { VaultKeySection } from "#/components/settings/vault-key-section"
import { BillingSection } from "#/components/settings/billing-section"
import { User, ShieldCheck, Building2, KeyRound, CreditCard } from "lucide-react"

const tabs = ["account", "vault-key", "recovery", "organization", "billing"] as const
type Tab = (typeof tabs)[number]

const searchSchema = z.object({
	tab: z.enum(tabs),
	action: z.string()
})

export const Route = createFileRoute("/_authed/_unlocked/settings")({
	validateSearch: searchSchema.partial(),
	component: SettingsPage,
})

const navItems = [
	{ value: "account" as const, label: "Account", icon: User },
	{ value: "vault-key" as const, label: "Vault Key", icon: KeyRound },
	{ value: "recovery" as const, label: "Recovery", icon: ShieldCheck },
	{ value: "organization" as const, label: "Organization", icon: Building2 },
	{ value: "billing" as const, label: "Billing", icon: CreditCard },
]

function SettingsPage() {
	const { tab, action } = Route.useSearch()
	const navigate = useNavigate()
	const activeTab: Tab = tab ?? "account"

	return (
		<div>
			<div className="mb-2">
				<h1 className="text-xl font-semibold tracking-tight">Settings</h1>
				<p className="mt-1 text-sm text-muted-foreground">
					Manage your account settings and preferences.
				</p>
			</div>

			{/* Desktop: horizontal tabs */}
			<div className="hidden md:block">
				<Tabs
					value={activeTab}
					onValueChange={(v) =>
						navigate({ to: "/settings", search: { tab: v as Tab } })
					}
				>
					<TabsList variant="line" className="mb-8">
						{navItems.map((item) => (
							<TabsTrigger key={item.value} value={item.value}>
								{item.label}
							</TabsTrigger>
						))}
					</TabsList>
				</Tabs>
			</div>

			{/* Mobile: vertical nav */}
			<nav className="mb-6 flex gap-1 overflow-x-auto md:hidden">
				{navItems.map((item) => (
					<Link
						key={item.value}
						to="/settings"
						search={{ tab: item.value }}
						className={cn(
							"flex shrink-0 items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs no-underline transition-colors",
							activeTab === item.value
								? "bg-muted font-medium text-foreground"
								: "text-muted-foreground hover:bg-muted/50",
						)}
					>
						<item.icon className="size-3" />
						{item.label}
					</Link>
				))}
			</nav>

			{/* Content */}
			{activeTab === "account" && <AccountSection />}
			{activeTab === "vault-key" && <VaultKeySection />}
			{activeTab === "recovery" && <RecoverySection action={action} />}
			{activeTab === "organization" && <OrgSection />}
			{activeTab === "billing" && <BillingSection />}
		</div>
	)
}
