import { useState, useCallback } from "react"
import { cn } from "#/lib/utils"
import { useNavStore } from "#/lib/stores/nav"
import { useUpdatePreferences } from "#/lib/queries/preferences"
import { SettingsRow } from "./settings-row"
import { Sun, Moon, Monitor, Check } from "lucide-react"

type ThemeMode = "light" | "dark" | "auto"

function getStoredTheme(): ThemeMode {
	if (typeof window === "undefined") return "auto"
	const stored = window.localStorage.getItem("theme")
	if (stored === "light" || stored === "dark" || stored === "auto") return stored
	return "auto"
}

function applyTheme(mode: ThemeMode) {
	const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches
	const resolved = mode === "auto" ? (prefersDark ? "dark" : "light") : mode

	document.documentElement.classList.remove("light", "dark")
	document.documentElement.classList.add(resolved)

	if (mode === "auto") {
		document.documentElement.removeAttribute("data-theme")
	} else {
		document.documentElement.setAttribute("data-theme", mode)
	}

	document.documentElement.style.colorScheme = resolved
	window.localStorage.setItem("theme", mode)
}

const themes: { value: ThemeMode; label: string; description: string; icon: typeof Sun }[] = [
	{ value: "light", label: "Light", description: "Clean and bright", icon: Sun },
	{ value: "dark", label: "Dark", description: "Easy on the eyes", icon: Moon },
	{ value: "auto", label: "System", description: "Match your OS", icon: Monitor },
]

export function AppearanceSection() {
	const [mode, setMode] = useState<ThemeMode>(getStoredTheme)
	const setStoreTheme = useNavStore((s) => s.setTheme)
	const updatePrefs = useUpdatePreferences()

	const select = useCallback((value: ThemeMode) => {
		setMode(value)
		applyTheme(value)
		setStoreTheme(value)
		updatePrefs.mutate({ theme: value })
	}, [setStoreTheme, updatePrefs])

	return (
		<div>
			<SettingsRow title="Theme" description="Choose how zEnv looks. Select a theme or let it follow your system preference.">
				<div className="grid grid-cols-3 gap-3">
					{themes.map((theme) => {
						const active = mode === theme.value
						return (
							<button
								key={theme.value}
								type="button"
								onClick={() => select(theme.value)}
								className={cn(
									"group relative flex flex-col items-center gap-2.5 rounded-lg border p-4 text-center transition-all",
									active
										? "border-primary bg-primary/5"
										: "border-border hover:border-muted-foreground/30 hover:bg-muted/50",
								)}
							>
								{active && (
									<div className="absolute top-2 right-2 flex size-4 items-center justify-center rounded-full bg-primary">
										<Check className="size-2.5 text-primary-foreground" />
									</div>
								)}
								<ThemePreview mode={theme.value} active={active} />
								<div>
									<p className={cn("text-xs font-medium", active ? "text-foreground" : "text-muted-foreground group-hover:text-foreground")}>
										{theme.label}
									</p>
									<p className="mt-0.5 text-[10px] text-muted-foreground">{theme.description}</p>
								</div>
							</button>
						)
					})}
				</div>
			</SettingsRow>
		</div>
	)
}

function ThemePreview({ mode, active }: { mode: ThemeMode; active: boolean }) {
	const isLight = mode === "light"
	const isDark = mode === "dark"
	const isSystem = mode === "auto"

	// Colors for the mini UI preview
	const bg = isDark ? "bg-zinc-900" : isLight ? "bg-white" : "bg-gradient-to-r from-white to-zinc-900"
	const sidebar = isDark ? "bg-zinc-800" : isLight ? "bg-zinc-100" : ""
	const line = isDark ? "bg-zinc-700" : isLight ? "bg-zinc-200" : "bg-zinc-300"
	const accent = isDark ? "bg-blue-500" : isLight ? "bg-blue-500" : "bg-blue-500"

	return (
		<div
			className={cn(
				"flex h-16 w-full overflow-hidden rounded-md border",
				active ? "border-primary/40" : "border-border",
				bg,
			)}
		>
			{isSystem ? (
				<>
					{/* Light half */}
					<div className="flex flex-1 bg-white">
						<div className="w-5 shrink-0 bg-zinc-100 p-1">
							<div className="mb-1 h-1 w-full rounded-full bg-zinc-300" />
							<div className="h-1 w-2/3 rounded-full bg-zinc-300" />
						</div>
						<div className="flex-1 p-1.5">
							<div className="mb-1 h-1.5 w-3/4 rounded bg-zinc-200" />
							<div className="h-1 w-1/2 rounded bg-blue-400/40" />
						</div>
					</div>
					{/* Dark half */}
					<div className="flex flex-1 bg-zinc-900">
						<div className="w-5 shrink-0 bg-zinc-800 p-1">
							<div className="mb-1 h-1 w-full rounded-full bg-zinc-700" />
							<div className="h-1 w-2/3 rounded-full bg-zinc-700" />
						</div>
						<div className="flex-1 p-1.5">
							<div className="mb-1 h-1.5 w-3/4 rounded bg-zinc-700" />
							<div className="h-1 w-1/2 rounded bg-blue-500/40" />
						</div>
					</div>
				</>
			) : (
				<>
					{/* Sidebar */}
					<div className={cn("w-6 shrink-0 p-1", sidebar)}>
						<div className={cn("mb-1 h-1 w-full rounded-full", line)} />
						<div className={cn("mb-1 h-1 w-2/3 rounded-full", line)} />
						<div className={cn("h-1 w-4/5 rounded-full", accent, "opacity-60")} />
					</div>
					{/* Content */}
					<div className="flex-1 p-1.5">
						<div className={cn("mb-1.5 h-1.5 w-3/4 rounded", line)} />
						<div className={cn("mb-1 h-1 w-full rounded", line, "opacity-50")} />
						<div className={cn("mb-1 h-1 w-5/6 rounded", line, "opacity-50")} />
						<div className={cn("mt-2 h-2 w-1/3 rounded", accent, "opacity-50")} />
					</div>
				</>
			)}
		</div>
	)
}
