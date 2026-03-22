import { cn } from "#/lib/utils"

interface PageHeaderProps extends React.HTMLAttributes<HTMLDivElement> {
	title: string
	description?: string
	actions?: React.ReactNode
}

function PageHeader({ title, description, actions, className, ...props }: PageHeaderProps) {
	return (
		<div className={cn("flex items-start justify-between gap-4", className)} {...props}>
			<div className="min-w-0">
				<h1 className="truncate text-lg font-semibold tracking-tight">{title}</h1>
				{description && <p className="mt-0.5 text-sm text-muted-foreground">{description}</p>}
			</div>
			{actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
		</div>
	)
}

export { PageHeader }
