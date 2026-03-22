import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Spinner } from "#/components/ui/spinner"
import { NewVaultKeyForm } from "#/components/NewVaultKeyForm"
import { RecoveryKitModal } from "#/components/RecoveryKitModal"
import { useQuery } from "@tanstack/react-query"
import { meQueryOptions, useSetupVault } from "#/lib/queries/auth"
import { generateRecoveryEntropy, entropyToWords } from "#/lib/recovery"
import { storageKeys } from "#/lib/keys"
import { AlertCircle, Lock } from "lucide-react"
import type { KeyType } from "@zenv/amnesia"

export const Route = createFileRoute("/_authed/vault-setup")({
	component: VaultSetupPage,
})

type Step = "key-input" | "deriving" | "recovery-gate"

function VaultSetupPage() {
	const navigate = useNavigate()
	const { data: me } = useQuery(meQueryOptions)
	const setupVault = useSetupVault()

	const [step, setStep] = useState<Step>("key-input")
	const [mnemonic, setMnemonic] = useState("")

	const handleKeySubmit = async (vaultKey: string, keyType: KeyType) => {
		setStep("deriving")

		const entropy = generateRecoveryEntropy()
		const mnemonicWords = entropyToWords(entropy)
		setMnemonic(mnemonicWords)

		try {
			await setupVault.mutateAsync({ vaultKey, keyType, recoveryKey: entropy })
			setStep("recovery-gate")
		} catch {
			setStep("key-input")
		}
	}

	const handleRecoveryConfirm = async () => {
		setMnemonic("")

		// Check for pending invite token from signup flow
		const inviteToken = sessionStorage.getItem(storageKeys.inviteToken)
		if (inviteToken) {
			sessionStorage.removeItem(storageKeys.inviteToken)
			navigate({ to: "/join/$token", params: { token: inviteToken } })
			return
		}

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
		<div className="flex min-h-screen flex-col bg-background">
			<div className="flex flex-1 items-center justify-center px-4 py-8">
				<div className="w-full max-w-100">
					<CardBox>
						<Card className="p-0">
							<CardHeader className="px-6 pt-6 text-center">
								<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
									<Lock className="size-4" />
								</div>
								<CardTitle>Create your Vault Key</CardTitle>
								<CardDescription className="text-xs">
									This key encrypts your secrets. It never leaves your device.
								</CardDescription>
							</CardHeader>

							<CardContent className="px-6 pt-4 pb-6">
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

							<div className="border-t border-border bg-muted/30 px-6 py-3 text-center text-xs text-muted-foreground">
								zEnv never sees your Vault Key. If you forget it, you'll need your Recovery Kit.
							</div>
						</Card>
					</CardBox>
				</div>
			</div>

			<footer className="flex items-center justify-between px-6 py-4 text-xs text-muted-foreground">
				<span>&copy; {new Date().getFullYear()} zEnv</span>
				<div className="flex items-center gap-1">
					<a href="/support" className="hover:text-foreground">Support</a>
					<span>&middot;</span>
					<a href="/privacy" className="hover:text-foreground">Privacy</a>
					<span>&middot;</span>
					<a href="/terms" className="hover:text-foreground">Terms</a>
				</div>
			</footer>
		</div>
	)
}
