import { useRef, useCallback } from "react"
import { Input } from "#/components/ui/input"
import { MNEMONIC_WORD_COUNT } from "#/lib/recovery"

interface MnemonicInputProps {
	words: string[]
	onChange: (words: string[]) => void
	disabled?: boolean
}

/**
 * Grid for entering BIP39 mnemonic recovery words.
 * Supports paste of all words at once (space or newline separated).
 */
export function MnemonicInput({ words, onChange, disabled }: MnemonicInputProps) {
	const count = MNEMONIC_WORD_COUNT
	const lastIdx = count - 1
	const inputRefs = useRef<(HTMLInputElement | null)[]>([])

	const setRef = useCallback((el: HTMLInputElement | null, i: number) => {
		inputRefs.current[i] = el
	}, [])

	const handleChange = (i: number, value: string) => {
		const cleaned = value.toLowerCase().replace(/[^a-z]/g, "")
		const next = [...words]
		next[i] = cleaned
		onChange(next)

		if (cleaned.length >= 3 && i < lastIdx) {
			inputRefs.current[i + 1]?.focus()
		}
	}

	const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>, i: number) => {
		const text = e.clipboardData.getData("text").trim()
		const pasted = text.split(/[\s,]+/).filter(Boolean)

		if (pasted.length > 1) {
			e.preventDefault()
			const next = [...words]
			for (let j = 0; j < pasted.length && i + j < count; j++) {
				next[i + j] = pasted[j].toLowerCase().replace(/[^a-z]/g, "")
			}
			onChange(next)

			const focusIdx = Math.min(i + pasted.length, lastIdx)
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
			if (i < lastIdx) inputRefs.current[i + 1]?.focus()
		}
	}

	// 12 words → 3 cols, 24 words → 4 cols
	const cols = count <= 12 ? "grid-cols-3" : "grid-cols-4"

	return (
		<div className={`grid ${cols} gap-2`}>
			{Array.from({ length: count }, (_, i) => (
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
