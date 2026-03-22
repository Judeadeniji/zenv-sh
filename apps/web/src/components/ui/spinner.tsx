import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "#/lib/utils"

const spinnerVariants = cva(
	"inline-block animate-spin rounded-full border-current border-b-transparent border-l-transparent",
	{
		variants: {
			size: {
				xs: "size-3 border",
				sm: "size-4 border-2",
				md: "size-5 border-2",
				lg: "size-6 border-[2.5px]",
				xl: "size-8 border-[3px]",
			},
		},
		defaultVariants: {
			size: "sm",
		},
	},
)

interface SpinnerProps extends React.HTMLAttributes<HTMLSpanElement>, VariantProps<typeof spinnerVariants> {
	label?: string
}

function Spinner({ className, size, label = "Loading", ...props }: SpinnerProps) {
	return (
		<span
			role="status"
			className={cn(spinnerVariants({ size, className }))}
			aria-busy
			aria-live="polite"
			aria-label={label}
			{...props}
		/>
	)
}

export { Spinner, spinnerVariants }
