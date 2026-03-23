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
