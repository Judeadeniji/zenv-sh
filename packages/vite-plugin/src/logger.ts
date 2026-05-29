/**
 * @zenv-sh/vite-plugin — terminal logger.
 *
 * Centralised logging with ANSI colour codes so output is visually distinct
 * from Vite's own log lines. Contributors: add new log levels here, not inline.
 */

const PREFIX = "\x1b[36m[zenv]\x1b[0m" // cyan
const WARN_PREFIX = "\x1b[33m[zenv]\x1b[0m" // yellow

/** Log an informational message to stdout. */
export const log = (msg: string): void => {
	console.log(`${PREFIX} ${msg}`)
}

/** Log a warning message to stderr. */
export const warn = (msg: string): void => {
	console.warn(`${WARN_PREFIX} ${msg}`)
}
