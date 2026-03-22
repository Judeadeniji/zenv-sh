import * as React from "react"
import { mergeProps } from "@base-ui/react/merge-props"
import { useRender } from "@base-ui/react/use-render"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "#/lib/utils"

const buttonVariants = cva(
	[
		"relative isolate inline-flex shrink-0 items-center justify-center",
		"rounded-md font-medium whitespace-nowrap select-none",
		"outline-none transition-all duration-200",
		"disabled:pointer-events-none disabled:opacity-50",
		"[&_svg]:pointer-events-none [&_svg]:shrink-0",
	].join(" "),
	{
		variants: {
			variant: {
				solid: [
					"border-0 bg-primary text-primary-foreground",
					"shadow-[0px_0px_0px_1px_var(--primary),0px_1px_1px_0px_rgba(255,255,255,0.07)_inset,0px_2px_3px_0px_rgba(34,42,53,0.20),0px_1px_1px_0px_rgba(0,0,0,0.24)]",
					"after:pointer-events-none after:absolute after:inset-0 after:rounded-[inherit] after:-z-1",
					"after:bg-[linear-gradient(180deg,rgba(255,255,255,0.15)_0%,transparent_100%)]",
					"after:opacity-100 after:transition-opacity after:duration-200",
					"hover:after:opacity-0 active:after:opacity-100",
					"focus-visible:shadow-[0px_0px_0px_1px_var(--primary),0px_1px_1px_0px_rgba(255,255,255,0.07)_inset,0px_2px_3px_0px_rgba(34,42,53,0.20),0px_1px_1px_0px_rgba(0,0,0,0.24),var(--shadow-focus-ring)]",
				].join(" "),
				outline: [
					"border-0 bg-background text-foreground",
					"shadow-[0px_0px_0px_1px_var(--border-alpha-150),0px_2px_3px_-1px_rgba(0,0,0,0.08),0px_1px_0px_0px_rgba(0,0,0,0.02)]",
					"hover:bg-muted",
					"focus-visible:shadow-[0px_0px_0px_1px_var(--border-alpha-150),0px_2px_3px_-1px_rgba(0,0,0,0.08),0px_1px_0px_0px_rgba(0,0,0,0.02),var(--shadow-focus-ring)]",
				].join(" "),
				ghost: [
					"border-0 text-muted-foreground",
					"hover:bg-muted hover:text-foreground",
					"focus-visible:shadow-(--shadow-focus-ring)",
				].join(" "),
				danger: [
					"border-0 bg-destructive text-white",
					"shadow-[0px_0px_0px_1px_var(--destructive),0px_1px_1px_0px_rgba(255,255,255,0.07)_inset,0px_2px_3px_0px_rgba(34,42,53,0.20),0px_1px_1px_0px_rgba(0,0,0,0.24)]",
					"after:pointer-events-none after:absolute after:inset-0 after:rounded-[inherit] after:-z-1",
					"after:bg-[linear-gradient(180deg,rgba(255,255,255,0.15)_0%,transparent_100%)]",
					"after:opacity-100 after:transition-opacity after:duration-200",
					"hover:after:opacity-0 active:after:opacity-100",
					"focus-visible:shadow-[0px_0px_0px_1px_var(--destructive),0px_1px_1px_0px_rgba(255,255,255,0.07)_inset,0px_2px_3px_0px_rgba(34,42,53,0.20),0px_1px_1px_0px_rgba(0,0,0,0.24),0px_0px_0px_3px_var(--danger-alpha-200)]",
				].join(" "),
				link: [
					"border-0 text-primary underline-offset-4",
					"hover:underline",
					"focus-visible:shadow-(--shadow-focus-ring)",
				].join(" "),
			},
			size: {
				xs: "h-6 gap-1 px-2 text-xs [&_svg:not([class*='size-'])]:size-3",
				sm: "h-7 gap-1 px-2.5 text-xs [&_svg:not([class*='size-'])]:size-3.5",
				md: "h-8 gap-1.5 px-3 text-sm [&_svg:not([class*='size-'])]:size-4",
				icon: "size-7 [&_svg:not([class*='size-'])]:size-4",
				"icon-sm": "size-6 [&_svg:not([class*='size-'])]:size-3.5",
			},
		},
		compoundVariants: [
			{ variant: "link", className: "h-auto p-0" },
		],
		defaultVariants: {
			variant: "solid",
			size: "sm",
		},
	},
)

interface ButtonProps
	extends useRender.ComponentProps<"button">,
		VariantProps<typeof buttonVariants> {
	isLoading?: boolean
	loadingText?: string
}

function Button({ className, variant, size, isLoading, loadingText, children, disabled, render, ...props }: ButtonProps) {
	const content = isLoading ? (
		<span className="relative inline-flex items-center gap-1.5">
			<span
				className={cn(
					"inline-block size-3.5 animate-spin rounded-full",
					"border-[1.5px] border-current border-b-transparent border-l-transparent",
					loadingText ? "" : "absolute",
				)}
				aria-busy
				aria-live="polite"
			/>
			{loadingText || <span className="invisible">{children}</span>}
		</span>
	) : (
		children
	)

	return useRender({
		defaultTagName: "button",
		render,
		props: mergeProps<"button">(
			{
				"data-slot": "button",
				className: cn(buttonVariants({ variant, size, className })),
				disabled: disabled || isLoading,
			},
			props,
			{ children: content },
		),
		state: {},
	})
}

export { Button, buttonVariants }
