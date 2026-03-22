import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "#/lib/utils"

const alertVariants = cva(
	[
		"relative flex w-full gap-3 rounded-lg border-0 p-4",
		"text-sm [&_svg]:size-4 [&_svg]:shrink-0 [&_svg]:translate-y-0.5",
	].join(" "),
	{
		variants: {
			variant: {
				info: "bg-primary-alpha-50 text-foreground shadow-[0px_0px_0px_1px_var(--border-alpha-100)] [&_svg]:text-primary",
				danger: "bg-[var(--danger-alpha-200)] text-foreground shadow-[0px_0px_0px_1px_var(--danger-alpha-300)] [&_svg]:text-destructive",
				warning: "bg-[var(--warning-alpha-200)] text-foreground shadow-[0px_0px_0px_1px_var(--warning-alpha-300)] [&_svg]:text-warning",
				success: "bg-[var(--success-alpha-200)] text-foreground shadow-[0px_0px_0px_1px_var(--success-alpha-300)] [&_svg]:text-success",
			},
		},
		defaultVariants: {
			variant: "info",
		},
	},
)

interface AlertProps extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof alertVariants> {}

const Alert = React.forwardRef<HTMLDivElement, AlertProps>(
	({ className, variant, ...props }, ref) => {
		return <div ref={ref} role="alert" className={cn(alertVariants({ variant, className }))} {...props} />
	},
)
Alert.displayName = "Alert"

const AlertTitle = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLHeadingElement>>(
	({ className, ...props }, ref) => {
		return <h5 ref={ref} className={cn("mb-1 font-medium leading-none", className)} {...props} />
	},
)
AlertTitle.displayName = "AlertTitle"

const AlertDescription = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(
	({ className, ...props }, ref) => {
		return <p ref={ref} className={cn("text-sm leading-relaxed opacity-80", className)} {...props} />
	},
)
AlertDescription.displayName = "AlertDescription"

export { Alert, AlertTitle, AlertDescription, alertVariants }
