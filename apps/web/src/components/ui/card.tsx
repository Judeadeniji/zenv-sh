import * as React from "react"

import { cn } from "#/lib/utils"

/*
 * Card — Clerk-style compound component.
 *
 * CardBox: outer wrapper with card shadow + 1px shadow border
 * Card: inner content with content shadow + 1px shadow border, bg
 * CardHeader / CardContent / CardFooter: layout slots
 */

const CardBox = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
	({ className, ...props }, ref) => (
		<div
			ref={ref}
			className={cn(
				"isolate w-full overflow-hidden rounded-xl border-0 text-foreground",
				"shadow-[var(--shadow-card),0px_0px_0px_1px_var(--border-alpha-100)]",
				className,
			)}
			{...props}
		/>
	),
)
CardBox.displayName = "CardBox"

const Card = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
	({ className, ...props }, ref) => (
		<div
			ref={ref}
			className={cn(
				"relative rounded-lg border-0 bg-card p-8 text-card-foreground",
				"shadow-[var(--shadow-card-content),0px_0px_0px_1px_var(--border-alpha-50)]",
				className,
			)}
			{...props}
		/>
	),
)
Card.displayName = "Card"

const CardHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
	({ className, ...props }, ref) => (
		<div ref={ref} className={cn("flex flex-col gap-1.5", className)} {...props} />
	),
)
CardHeader.displayName = "CardHeader"

const CardTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
	({ className, ...props }, ref) => (
		<h3 ref={ref} className={cn("text-base font-medium leading-none tracking-tight", className)} {...props} />
	),
)
CardTitle.displayName = "CardTitle"

const CardDescription = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(
	({ className, ...props }, ref) => (
		<p ref={ref} className={cn("text-sm text-muted-foreground", className)} {...props} />
	),
)
CardDescription.displayName = "CardDescription"

const CardContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
	({ className, ...props }, ref) => (
		<div ref={ref} className={cn("pt-6", className)} {...props} />
	),
)
CardContent.displayName = "CardContent"

const CardFooter = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
	({ className, ...props }, ref) => (
		<div
			ref={ref}
			className={cn("flex items-center gap-3 pt-6", className)}
			{...props}
		/>
	),
)
CardFooter.displayName = "CardFooter"

const ActionCard = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
	({ className, ...props }, ref) => (
		<div
			ref={ref}
			className={cn(
				"rounded-lg border-0 bg-card p-4 text-card-foreground",
				"shadow-[var(--shadow-action-card),0px_0px_0px_1px_var(--border-alpha-100)]",
				className,
			)}
			{...props}
		/>
	),
)
ActionCard.displayName = "ActionCard"

export { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter, ActionCard }
