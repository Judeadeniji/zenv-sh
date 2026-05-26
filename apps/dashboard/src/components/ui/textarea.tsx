import * as React from "react"

import { cn } from "#/lib/utils"
import type { Feedback } from "./input"

interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
	feedback?: Feedback
}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
	({ className, feedback, ...props }, ref) => {
		return (
			<textarea
				ref={ref}
				data-feedback={feedback}
				className={cn(
					"flex min-h-[72px] w-full rounded-md bg-input/30 px-2.5 py-1.5",
					"text-sm text-foreground placeholder:text-muted-foreground",
					"outline-none border-0 transition-shadow duration-200 resize-y",
					"disabled:cursor-not-allowed disabled:opacity-50",
					"shadow-[0px_0px_0px_1px_var(--border-alpha-150),var(--shadow-input)]",
					"hover:shadow-[0px_0px_0px_1px_var(--border-alpha-300),var(--shadow-input)]",
					"focus:shadow-[0px_0px_0px_1px_var(--border-alpha-300),var(--shadow-input),var(--shadow-focus-ring)]",
					"data-[feedback=error]:shadow-[0px_0px_0px_1px_var(--danger-alpha-400),0px_0px_1px_0px_var(--danger-alpha-200)]",
					"data-[feedback=error]:focus:shadow-[0px_0px_0px_1px_var(--danger-alpha-500),0px_0px_1px_0px_var(--danger-alpha-200),0px_0px_0px_3px_var(--danger-alpha-200)]",
					className,
				)}
				{...props}
			/>
		)
	},
)
Textarea.displayName = "Textarea"

export { Textarea }
