import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs))
}

export function getInitials(name?: string, email?: string): string {
	const source = name || email || "?"
	return source
		.split(/[\s@]/)
		.slice(0, 2)
		.map((s) => s[0]?.toUpperCase() ?? "")
		.join("")
}
