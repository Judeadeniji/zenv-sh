import * as React from "react"
import { cn } from "#/lib/utils"
import { Button } from "#/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert"
import { Check, Copy, Eye, EyeOff, TriangleAlert } from "lucide-react"

interface OneTimeDisplayProps extends React.HTMLAttributes<HTMLDivElement> {
	value: string
	label: string
	warning?: string
	masked?: boolean
}

function OneTimeDisplay({ value, label, warning, masked = true, className, ...props }: OneTimeDisplayProps) {
	const [copied, setCopied] = React.useState(false)
	const [revealed, setRevealed] = React.useState(!masked)

	const handleCopy = React.useCallback(() => {
		if (navigator.clipboard?.writeText) {
			navigator.clipboard.writeText(value).then(() => {
				setCopied(true)
				setTimeout(() => setCopied(false), 2000)
			})
		} else {
			const textarea = document.createElement("textarea")
			textarea.value = value
			textarea.style.position = "fixed"
			textarea.style.opacity = "0"
			document.body.appendChild(textarea)
			textarea.select()
			document.execCommand("copy")
			document.body.removeChild(textarea)
			setCopied(true)
			setTimeout(() => setCopied(false), 2000)
		}
	}, [value])

	return (
		<div className={cn("space-y-3", className)} {...props}>
			<div className="space-y-1.5">
				<label className="text-xs font-medium text-muted-foreground">{label}</label>
				<div className="flex items-center gap-2">
					<div className="flex-1 overflow-hidden rounded-md bg-muted px-2.5 py-1.5 font-mono text-xs shadow-[0px_0px_0px_1px_var(--border-alpha-100)]">
						{revealed ? (
							<span className="select-all break-all">{value}</span>
						) : (
							<span className="text-muted-foreground">{"•".repeat(Math.min(value.length, 40))}</span>
						)}
					</div>
					{masked && (
						<Button
							variant="outline"
							size="icon-sm"
							onClick={() => setRevealed((r) => !r)}
							aria-label={revealed ? "Hide value" : "Reveal value"}
						>
							{revealed ? <EyeOff /> : <Eye />}
						</Button>
					)}
					<Button variant="outline" size="icon-sm" onClick={handleCopy} aria-label="Copy to clipboard">
						{copied ? <Check className="text-success" /> : <Copy />}
					</Button>
				</div>
			</div>
			{warning && (
				<Alert variant="warning">
					<TriangleAlert />
					<div>
						<AlertTitle>One-time display</AlertTitle>
						<AlertDescription>{warning}</AlertDescription>
					</div>
				</Alert>
			)}
		</div>
	)
}

export { OneTimeDisplay }
