import { Link } from "@tanstack/react-router"
import ThemeToggle from "./ThemeToggle"

export default function Header() {
	return (
		<header className="sticky top-0 z-50 border-b border-border bg-background/80 px-4 backdrop-blur-lg">
			<nav className="mx-auto flex max-w-7xl items-center gap-4 py-3">
				<Link
					to="/"
					className="flex items-center gap-2 text-sm font-medium text-foreground no-underline"
				>
					<span className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-xs font-bold text-primary-foreground">
						z
					</span>
					zEnv
				</Link>

				<div className="ml-auto flex items-center gap-2">
					<ThemeToggle />
				</div>
			</nav>
		</header>
	)
}
