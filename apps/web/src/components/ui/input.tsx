import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "#/lib/utils"

const inputVariants = cva(
	[
		"flex w-full rounded-md bg-input/30 px-2.5",
		"text-foreground placeholder:text-muted-foreground",
		"outline-none border-0 transition-shadow duration-200",
		"disabled:cursor-not-allowed disabled:opacity-50",
		"shadow-[0px_0px_0px_1px_var(--border-alpha-150),var(--shadow-input)]",
		"hover:shadow-[0px_0px_0px_1px_var(--border-alpha-300),var(--shadow-input)]",
		"focus:shadow-[0px_0px_0px_1px_var(--border-alpha-300),var(--shadow-input),var(--shadow-focus-ring)]",
		"focus-within:shadow-[0px_0px_0px_1px_var(--border-alpha-300),var(--shadow-input),var(--shadow-focus-ring)]",
		"data-[feedback=error]:shadow-[0px_0px_0px_1px_var(--danger-alpha-400),0px_0px_1px_0px_var(--danger-alpha-200)]",
		"data-[feedback=error]:hover:shadow-[0px_0px_0px_1px_var(--danger-alpha-500),0px_0px_1px_0px_var(--danger-alpha-200)]",
		"data-[feedback=error]:focus:shadow-[0px_0px_0px_1px_var(--danger-alpha-500),0px_0px_1px_0px_var(--danger-alpha-200),0px_0px_0px_3px_var(--danger-alpha-200)]",
		"data-[feedback=warning]:shadow-[0px_0px_0px_1px_var(--warning-alpha-400),0px_0px_1px_0px_var(--warning-alpha-200)]",
		"data-[feedback=warning]:hover:shadow-[0px_0px_0px_1px_var(--warning-alpha-500),0px_0px_1px_0px_var(--warning-alpha-200)]",
		"data-[feedback=warning]:focus:shadow-[0px_0px_0px_1px_var(--warning-alpha-500),0px_0px_1px_0px_var(--warning-alpha-200),0px_0px_0px_3px_var(--warning-alpha-200)]",
		"data-[feedback=success]:shadow-[0px_0px_0px_1px_var(--success-alpha-400),0px_0px_1px_0px_var(--success-alpha-200)]",
		"data-[feedback=success]:hover:shadow-[0px_0px_0px_1px_var(--success-alpha-500),0px_0px_1px_0px_var(--success-alpha-200)]",
		"data-[feedback=success]:focus:shadow-[0px_0px_0px_1px_var(--success-alpha-500),0px_0px_1px_0px_var(--success-alpha-200),0px_0px_0px_3px_var(--success-alpha-200)]",
	].join(" "),
	{
		variants: {
			inputSize: {
				sm: "h-7 text-xs",
				md: "h-8 text-sm",
			},
		},
		defaultVariants: {
			inputSize: "md",
		},
	},
)

type Feedback = "error" | "warning" | "success"

interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size">, VariantProps<typeof inputVariants> {
	feedback?: Feedback
}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
	({ className, inputSize, feedback, type, ...props }, ref) => {
		return (
			<input
				ref={ref}
				type={type}
				data-feedback={feedback}
				className={cn(inputVariants({ inputSize, className }))}
				{...props}
			/>
		)
	},
)
Input.displayName = "Input"

export { Input, inputVariants, type Feedback }
