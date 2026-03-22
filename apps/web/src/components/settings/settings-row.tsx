import { cn } from "#/lib/utils"

interface SettingsRowProps {
	title: string
	description?: string
	children: React.ReactNode
	className?: string
}

/**
 * Two-column settings row: description on the left, controls on the right.
 * Stacks vertically on mobile.
 */
export function SettingsRow({ title, description, children, className }: SettingsRowProps) {
	return (
		<div className={cn("grid gap-4 py-6 md:grid-cols-[280px_1fr]", className)}>
			<div>
				<h3 className="text-sm font-medium">{title}</h3>
				{description && <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{description}</p>}
			</div>
			<div className="max-w-lg">{children}</div>
		</div>
	)
}

/**
 * Horizontal divider between settings rows.
 */
export function SettingsDivider() {
	return <div className="border-t border-border" />
}
