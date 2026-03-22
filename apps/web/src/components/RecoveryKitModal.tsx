import { useState } from "react"
import { Button } from "#/components/ui/button"
import { Checkbox } from "#/components/ui/checkbox"
import { CardBox, Card, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { generateRecoveryKitPDF } from "#/lib/recovery-pdf"
import { Download, ShieldCheck, AlertTriangle } from "lucide-react"

interface RecoveryKitModalProps {
	email: string
	mnemonic: string
	onConfirm: () => void
	onDisableRecovery?: () => void
}

export function RecoveryKitModal({ email, mnemonic, onConfirm, onDisableRecovery }: RecoveryKitModalProps) {
	const [downloaded, setDownloaded] = useState(false)
	const [confirmed, setConfirmed] = useState(false)

	const words = mnemonic.split(" ")

	const handleDownload = () => {
		const date = new Date().toLocaleDateString("en-US", {
			year: "numeric",
			month: "long",
			day: "numeric",
		})
		const blob = generateRecoveryKitPDF(email, mnemonic, date)
		const url = URL.createObjectURL(blob)
		const a = document.createElement("a")
		a.href = url
		a.download = `zenv-recovery-kit-${Date.now()}.pdf`
		a.click()
		URL.revokeObjectURL(url)
		setDownloaded(true)
	}

	return (
		<div className="fixed inset-0 z-50 flex items-center justify-center bg-background/95 backdrop-blur-sm">
			<div className="w-full max-w-lg px-4">
				<div className="mb-6 text-center">
					<div className="mx-auto mb-3 flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
						<ShieldCheck className="size-4" />
					</div>
					<h1 className="text-lg font-semibold">Save your Recovery Kit</h1>
					<p className="mt-1 text-sm text-muted-foreground">
						This is the only way to recover your vault if you forget your key
					</p>
				</div>

				<CardBox>
					<Card>
						<CardContent className="pt-5">
							<Alert variant="warning" className="mb-4">
								<AlertTriangle />
								<AlertDescription>
									Write these words down or download the PDF. You won't see them again.
								</AlertDescription>
							</Alert>

							<div className="grid grid-cols-3 gap-x-4 gap-y-2 rounded-md bg-muted/50 p-3">
								{words.map((word, i) => (
									<div key={i} className="flex items-baseline gap-1.5">
										<span className="text-[10px] tabular-nums text-muted-foreground">
											{(i + 1).toString().padStart(2, "0")}
										</span>
										<span className="text-sm font-medium">{word}</span>
									</div>
								))}
							</div>

							<Button variant="outline" size="md" onClick={handleDownload} className="mt-4 w-full">
								<Download />
								Download Recovery Kit PDF
							</Button>

							<div className="mt-4 flex items-start gap-2">
								<Checkbox
									checked={confirmed}
									onCheckedChange={(v) => setConfirmed(v === true)}
									id="confirm-saved"
								/>
								<label htmlFor="confirm-saved" className="cursor-pointer text-xs leading-relaxed text-muted-foreground">
									I have saved my Recovery Kit in a secure location. I understand that without it, I cannot recover my
									vault if I forget my Vault Key.
								</label>
							</div>

							<Button
								variant="solid"
								size="md"
								onClick={onConfirm}
								disabled={!downloaded || !confirmed}
								className="mt-4 w-full"
							>
								Continue to dashboard
							</Button>

							{onDisableRecovery && (
								<button
									type="button"
									onClick={onDisableRecovery}
									className="mt-3 block w-full text-center text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
								>
									I don't want a Recovery Kit (not recommended)
								</button>
							)}
						</CardContent>
					</Card>
				</CardBox>
			</div>
		</div>
	)
}
