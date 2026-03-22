import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "#/lib/utils"

const badgeVariants = cva(
	[
		"inline-flex items-center rounded-md border-0 px-2 py-0.5",
		"text-xs font-medium transition-colors",
	].join(" "),
	{
		variants: {
			variant: {
				primary: [
					"bg-[var(--primary-alpha-50)] text-primary",
					"shadow-[0px_0px_0px_1px_var(--border-alpha-100),var(--shadow-badge)]",
				].join(" "),
				danger: [
					"bg-[var(--danger-alpha-200)] text-destructive",
					"shadow-[0px_0px_0px_1px_var(--danger-alpha-300),var(--shadow-badge)]",
				].join(" "),
				success: [
					"bg-[var(--success-alpha-200)] text-success",
					"shadow-[0px_0px_0px_1px_var(--success-alpha-300),var(--shadow-badge)]",
				].join(" "),
				warning: [
					"bg-[var(--warning-alpha-200)] text-warning",
					"shadow-[0px_0px_0px_1px_var(--warning-alpha-300),var(--shadow-badge)]",
				].join(" "),
				neutral: [
					"bg-muted text-muted-foreground",
					"shadow-[0px_0px_0px_1px_var(--border-alpha-100),var(--shadow-badge)]",
				].join(" "),
			},
		},
		defaultVariants: {
			variant: "neutral",
		},
	},
)

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
	return <span className={cn(badgeVariants({ variant, className }))} {...props} />
}

export { Badge, badgeVariants }
