import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import { Progress } from "#/components/ui/progress"
import { SettingsRow, SettingsDivider } from "./settings-row"
import { CreditCard, Receipt, ArrowUpRight } from "lucide-react"

// ── Pricing tiers from master plan (Section 6.3) ──

const TIERS = {
	free: { name: "Free", price: "$0/mo", projects: 1, secrets: 25, users: 1 },
	developer: { name: "Developer", price: "$15/mo", projects: "Unlimited", secrets: "1,000", users: 5 },
	team: { name: "Team", price: "$50/mo", projects: "Unlimited", secrets: "Unlimited", users: 15 },
	enterprise: { name: "Enterprise", price: "Custom", projects: "Unlimited", secrets: "Unlimited", users: "Unlimited" },
} as const

// ── Component ──

export function BillingSection() {
	// TODO: wire to real billing API (Stripe) when available
	const currentTier = "free" as keyof typeof TIERS
	const tier = TIERS[currentTier]

	return (
		<div>
			<PlanRow tier={tier} tierKey={currentTier} />
			<SettingsDivider />
			<UsageRow tier={tier} secretCount={0} projectCount={0} memberCount={1} />
			<SettingsDivider />
			<PaymentRow />
			<SettingsDivider />
			<InvoicesRow />
		</div>
	)
}

// ── Current Plan ──

function PlanRow({ tier, tierKey }: { tier: (typeof TIERS)[keyof typeof TIERS]; tierKey: string }) {
	return (
		<SettingsRow
			title="Current plan"
			description="Your subscription tier determines project, secret, and seat limits."
		>
			<div className="space-y-4">
				<div className="rounded-md border border-border px-4 py-3">
					<div className="flex items-center justify-between">
						<div>
							<p className="text-sm font-medium">{tier.name}</p>
							<p className="text-xs text-muted-foreground">{tier.price}</p>
						</div>
						<Badge variant={tierKey === "free" ? "neutral" : "primary"}>{tier.name}</Badge>
					</div>
					<div className="mt-3 grid grid-cols-3 gap-2 text-xs text-muted-foreground">
						<div>
							<span className="font-medium text-foreground">{tier.projects}</span> project{tier.projects === 1 ? "" : "s"}
						</div>
						<div>
							<span className="font-medium text-foreground">{tier.secrets}</span> secrets
						</div>
						<div>
							<span className="font-medium text-foreground">{tier.users}</span> user{tier.users === 1 ? "" : "s"}
						</div>
					</div>
				</div>

				{tierKey !== "enterprise" && (
					<Button variant="solid" size="sm">
						<ArrowUpRight />
						Upgrade plan
					</Button>
				)}
			</div>
		</SettingsRow>
	)
}

// ── Usage ──

function UsageRow({
	tier,
	secretCount,
	projectCount,
	memberCount,
}: {
	tier: (typeof TIERS)[keyof typeof TIERS]
	secretCount: number
	projectCount: number
	memberCount: number
}) {
	const secretLimit = typeof tier.secrets === "number" ? tier.secrets : null
	const projectLimit = typeof tier.projects === "number" ? tier.projects : null
	const memberLimit = typeof tier.users === "number" ? tier.users : null

	return (
		<SettingsRow title="Usage" description="Current usage against your plan limits.">
			<div className="space-y-3">
				<UsageBar label="Secrets" current={secretCount} max={secretLimit} />
				<UsageBar label="Projects" current={projectCount} max={projectLimit} />
				<UsageBar label="Members" current={memberCount} max={memberLimit} />
			</div>
		</SettingsRow>
	)
}

function UsageBar({ label, current, max }: { label: string; current: number; max: number | null }) {
	const pct = max ? Math.min((current / max) * 100, 100) : 0
	const isNearLimit = max ? pct >= 80 : false

	return (
		<div className="space-y-1">
			<div className="flex items-center justify-between text-xs">
				<span>{label}</span>
				<span className={isNearLimit ? "font-medium text-destructive" : "text-muted-foreground"}>
					{current} / {max ?? "∞"}
				</span>
			</div>
			{max && <Progress value={pct} className="h-1.5" />}
		</div>
	)
}

// ── Payment Method ──

function PaymentRow() {
	return (
		<SettingsRow title="Payment method" description="Manage your card on file for subscription billing.">
			<div className="flex items-center justify-between rounded-md border border-border px-3 py-2.5">
				<div className="flex items-center gap-2.5">
					<CreditCard className="size-4 text-muted-foreground" />
					<span className="text-sm text-muted-foreground">No payment method on file</span>
				</div>
				<Button variant="outline" size="xs">Add card</Button>
			</div>
		</SettingsRow>
	)
}

// ── Invoices ──

function InvoicesRow() {
	return (
		<SettingsRow title="Invoice history" description="Download past invoices and receipts.">
			<div className="flex items-center gap-2 rounded-md border border-border px-3 py-4 text-center">
				<Receipt className="mx-auto size-4 text-muted-foreground" />
				<p className="text-sm text-muted-foreground">No invoices yet</p>
			</div>
		</SettingsRow>
	)
}
