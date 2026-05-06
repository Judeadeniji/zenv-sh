const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
	["year", 31_536_000],
	["month", 2_592_000],
	["week", 604_800],
	["day", 86_400],
	["hour", 3_600],
	["minute", 60],
	["second", 1],
]

const rtf = new Intl.RelativeTimeFormat("en", { numeric: "auto", style: "narrow" })

const dtfDate = new Intl.DateTimeFormat("en-US", {
	year: "numeric",
	month: "short",
	day: "2-digit",
})

const dtfDateTime = new Intl.DateTimeFormat("en-US", {
	year: "numeric",
	month: "short",
	day: "2-digit",
	hour: "numeric",
	minute: "2-digit",
})

/**
 * Format a date string as a relative time ("2h ago", "3d ago", "just now").
 */
export function formatRelativeTime(dateStr: string): string {
	const diffSec = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000)
	if (diffSec < 5) return "just now"
	for (const [unit, seconds] of UNITS) {
		if (diffSec >= seconds) {
			return rtf.format(-Math.floor(diffSec / seconds), unit)
		}
	}
	return "just now"
}

/** Format a date string as "May 06, 2026" (en-US). */
export function formatDate(dateStr: string): string {
	const d = new Date(dateStr)
	if (Number.isNaN(d.getTime())) return "—"
	return dtfDate.format(d)
}

/** Format a date string as "May 06, 2026, 9:59 AM" (en-US). */
export function formatDateTime(dateStr: string): string {
	const d = new Date(dateStr)
	if (Number.isNaN(d.getTime())) return "—"
	return dtfDateTime.format(d)
}
