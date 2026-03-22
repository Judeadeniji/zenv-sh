import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { CardBox, Card, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Spinner } from "#/components/ui/spinner"
import { NewVaultKeyForm } from "#/components/NewVaultKeyForm"
import { RecoveryKitModal } from "#/components/RecoveryKitModal"
import { useSetupVault } from "#/lib/queries/auth"
import { useAuthStore } from "#/lib/stores/auth"
import { generateRecoveryKey, recoveryKeyToMnemonic } from "#/lib/recovery"
import { AlertCircle, Lock } from "lucide-react"
import type { KeyType } from "@zenv/amnesia"

export const Route = createFileRoute("/_authed/vault-setup")({
	component: VaultSetupPage,
})

type Step = "key-input" | "deriving" | "recovery-gate"

function VaultSetupPage() {
	const navigate = useNavigate()
	const me = useAuthStore((s) => s.me)
	const setupVault = useSetupVault()

	const [step, setStep] = useState<Step>("key-input")
	const [mnemonic, setMnemonic] = useState("")

	const handleKeySubmit = async (vaultKey: string, keyType: KeyType) => {
		setStep("deriving")

		const recoveryKey = generateRecoveryKey()
		const mnemonicWords = recoveryKeyToMnemonic(recoveryKey)
		setMnemonic(mnemonicWords)

		try {
			await setupVault.mutateAsync({ vaultKey, keyType, recoveryKey })
			setStep("recovery-gate")
		} catch {
			setStep("key-input")
		}
	}

	const handleRecoveryConfirm = () => {
		// Zero the mnemonic from state
		setMnemonic("")
		navigate({ to: "/" })
	}

	if (step === "deriving") {
		return (
			<div className="flex min-h-screen items-center justify-center px-4">
				<div className="flex flex-col items-center gap-4 text-center">
					<Spinner size="lg" />
					<div>
						<p className="text-sm font-medium">Setting up your vault</p>
						<p className="mt-1 text-xs text-muted-foreground">
							Running Argon2id key derivation — this may take a few seconds
						</p>
					</div>
				</div>
			</div>
		)
	}

	if (step === "recovery-gate") {
		return (
			<RecoveryKitModal
				email={me?.email ?? ""}
				mnemonic={mnemonic}
				onConfirm={handleRecoveryConfirm}
			/>
		)
	}

	return (
		<div className="flex min-h-screen items-center justify-center px-4">
			<div className="w-full max-w-sm">
				<div className="mb-6 text-center">
					<div className="mx-auto mb-3 flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
						<Lock className="size-4" />
					</div>
					<h1 className="text-lg font-semibold">Create your Vault Key</h1>
					<p className="mt-1 text-sm text-muted-foreground">
						This key encrypts your secrets. It never leaves your device.
					</p>
				</div>

				<CardBox>
					<Card>
						<CardContent className="pt-5">
							{setupVault.error && (
								<Alert variant="danger" className="mb-4">
									<AlertCircle />
									<AlertDescription>{setupVault.error.message}</AlertDescription>
								</Alert>
							)}

							<NewVaultKeyForm
								onSubmit={handleKeySubmit}
								isLoading={setupVault.isPending}
								loadingText="Deriving keys..."
								submitLabel="Create Vault"
							/>
						</CardContent>
					</Card>
				</CardBox>

				<p className="mt-4 text-center text-xs text-muted-foreground">
					zEnv never sees your Vault Key. If you forget it, you'll need your Recovery Kit.
				</p>
			</div>
		</div>
	)
}
