import { useRef, useCallback } from "react"
import { Input } from "#/components/ui/input"

interface MnemonicInputProps {
	words: string[]
	onChange: (words: string[]) => void
	disabled?: boolean
}

/**
 * 24-field grid for entering BIP39 mnemonic recovery words.
 * Supports paste of all 24 words at once (space or newline separated).
 */
export function MnemonicInput({ words, onChange, disabled }: MnemonicInputProps) {
	const inputRefs = useRef<(HTMLInputElement | null)[]>([])

	const setRef = useCallback((el: HTMLInputElement | null, i: number) => {
		inputRefs.current[i] = el
	}, [])

	const handleChange = (i: number, value: string) => {
		const cleaned = value.toLowerCase().replace(/[^a-z]/g, "")
		const next = [...words]
		next[i] = cleaned
		onChange(next)

		// Auto-advance on complete word (most BIP39 words are 3-8 chars)
		if (cleaned.length >= 3 && i < 23) {
			inputRefs.current[i + 1]?.focus()
		}
	}

	const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>, i: number) => {
		const text = e.clipboardData.getData("text").trim()
		const pasted = text.split(/[\s,]+/).filter(Boolean)

		if (pasted.length > 1) {
			e.preventDefault()
			const next = [...words]
			for (let j = 0; j < pasted.length && i + j < 24; j++) {
				next[i + j] = pasted[j].toLowerCase().replace(/[^a-z]/g, "")
			}
			onChange(next)

			// Focus the field after the last pasted word
			const focusIdx = Math.min(i + pasted.length, 23)
			inputRefs.current[focusIdx]?.focus()
		}
	}

	const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>, i: number) => {
		if (e.key === "Backspace" && words[i] === "" && i > 0) {
			e.preventDefault()
			inputRefs.current[i - 1]?.focus()
		}
		if (e.key === " " || e.key === "Tab") {
			if (e.key === " ") e.preventDefault()
			if (i < 23) inputRefs.current[i + 1]?.focus()
		}
	}

	return (
		<div className="grid grid-cols-3 gap-2">
			{Array.from({ length: 24 }, (_, i) => (
				<div key={i} className="flex items-center gap-1.5">
					<span className="w-5 text-right text-[10px] tabular-nums text-muted-foreground">
						{(i + 1).toString().padStart(2, "0")}
					</span>
					<Input
						ref={(el) => setRef(el, i)}
						value={words[i] || ""}
						onChange={(e) => handleChange(i, e.target.value)}
						onPaste={(e) => handlePaste(e, i)}
						onKeyDown={(e) => handleKeyDown(e, i)}
						inputSize="sm"
						className="font-mono"
						autoComplete="off"
						spellCheck={false}
						disabled={disabled}
					/>
				</div>
			))}
		</div>
	)
}
