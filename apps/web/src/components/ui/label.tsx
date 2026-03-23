import * as React from "react"

import { cn } from "#/lib/utils"

interface LabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {
	required?: boolean
}

const Label = React.forwardRef<HTMLLabelElement, LabelProps>(
	({ className, required, children, ...props }, ref) => {
		return (
			<label
				ref={ref}
				className={cn(
					"text-sm font-medium leading-none text-foreground",
					"peer-disabled:cursor-not-allowed peer-disabled:opacity-70 block",
					className,
				)}
				{...props}
			>
				{children}
				{required && <span className="ml-0.5 text-destructive" aria-hidden>*</span>}
			</label>
		)
	},
)
Label.displayName = "Label"

export { Label }
