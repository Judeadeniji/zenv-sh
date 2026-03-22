import { createFileRoute, Link } from "@tanstack/react-router"

export const Route = createFileRoute("/")({ component: Home })

function Home() {
	return (
		<div className="flex flex-col items-center justify-center py-24 text-center">
			<div className="mb-6 flex h-14 w-14 items-center justify-center rounded-xl bg-primary text-2xl font-bold text-primary-foreground shadow-lg">
				z
			</div>
			<h1 className="mb-3 text-3xl font-bold tracking-tight">zEnv Dashboard</h1>
			<p className="mb-8 max-w-md text-muted-foreground">
				Zero-knowledge secret management. All encryption happens client-side.
			</p>
			<Link
				to="/components"
				className="inline-flex items-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground no-underline shadow-(--shadow-button) transition-all hover:opacity-90"
			>
				Component Gallery
			</Link>
		</div>
	)
}
