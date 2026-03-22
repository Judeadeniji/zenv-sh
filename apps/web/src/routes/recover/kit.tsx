import { useState } from "react"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { useMutation, useQuery } from "@tanstack/react-query"
import { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Button } from "#/components/ui/button"
import { Spinner } from "#/components/ui/spinner"
import { MnemonicInput } from "#/components/MnemonicInput"
import { NewVaultKeyForm } from "#/components/NewVaultKeyForm"
import { RecoveryKitModal } from "#/components/RecoveryKitModal"
import { mnemonicToRecoveryKey, unwrapDekFromRecovery, generateRecoveryKey, recoveryKeyToMnemonic, wrapDekForRecovery } from "#/lib/recovery"
import { meQueryOptions } from "#/lib/queries/auth"
import { api } from "#/lib/api-client"
import { deriveKeys, hashAuthKey, wrapKey, generateSalt } from "@zenv/amnesia"
import type { KeyType } from "@zenv/amnesia"
import { mutationKeys } from "#/lib/keys"
import { AlertCircle, KeyRound, CheckCircle2 } from "lucide-react"

export const Route = createFileRoute("/recover/kit")({
	component: RecoverKitPage,
})

type Step = "enter-words" | "verifying" | "new-key" | "setting-up" | "recovery-gate"

function RecoverKitPage() {
	const navigate = useNavigate()
	const { data: me } = useQuery(meQueryOptions)

	const [step, setStep] = useState<Step>("enter-words")
	const [words, setWords] = useState<string[]>(Array(24).fill(""))
	const [dek, setDek] = useState<Uint8Array | null>(null)
	const [newMnemonic, setNewMnemonic] = useState("")
	const [error, setError] = useState("")

	const allFilled = words.every((w) => w.length >= 3)

	const verifyWords = useMutation({
		mutationKey: mutationKeys.recovery.recoverWithKit,
		mutationFn: async () => {
			const mnemonic = words.join(" ")
			const recoveryKey = mnemonicToRecoveryKey(mnemonic)

			const { data, error: fetchErr } = await api().GET("/auth/recovery/kit")
			if (fetchErr || !data) throw new Error("Failed to fetch recovery material")

			const res = data;
			const blob = Uint8Array.from(atob(res.recovery_wrapped_dek!), (c) => c.charCodeAt(0))
			const recoveredDek = await unwrapDekFromRecovery(blob, recoveryKey)

			return recoveredDek
		},
		onSuccess: (recoveredDek) => {
			setDek(recoveredDek)
			setError("")
			setStep("new-key")
		},
		onError: () => {
			setError("Invalid recovery words. Please check and try again.")
		},
	})

	const setupNewKey = useMutation({
		mutationKey: [...mutationKeys.recovery.recoverWithKit, "new-key"],
		mutationFn: async ({ vaultKey, keyType }: { vaultKey: string; keyType: KeyType }) => {
			if (!dek) throw new Error("No DEK available")

			const salt = generateSalt()
			const { kek, authKey } = await deriveKeys(vaultKey, salt, keyType)
			const authKeyHash = await hashAuthKey(authKey)

			const { ciphertext: wdCt, nonce: wdNonce } = await wrapKey(dek, kek)
			const wrappedDEK = new Uint8Array(wdNonce.length + wdCt.length)
			wrappedDEK.set(wdNonce, 0)
			wrappedDEK.set(wdCt, wdNonce.length)

			const newRecoveryKey = generateRecoveryKey()
			const newMnemonicWords = recoveryKeyToMnemonic(newRecoveryKey)
			setNewMnemonic(newMnemonicWords)

			const newRecoveryWrappedDEK = await wrapDekForRecovery(dek, newRecoveryKey)

			const toBase64 = (bytes: Uint8Array) => {
				let binary = ""
				for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i])
				return btoa(binary)
			}

			const { error } = await api().POST("/auth/recovery/kit/recover", {
				body: {
					new_vault_key_type: keyType,
					new_salt: toBase64(salt),
					new_auth_key_hash: toBase64(authKeyHash),
					new_wrapped_dek: toBase64(wrappedDEK),
					new_wrapped_private_key: "",
					new_recovery_wrapped_dek: toBase64(newRecoveryWrappedDEK),
				},
			})
			if (error) throw new Error("Failed to save new vault key")

			return { kek, dek, publicKey: new Uint8Array(), privateKey: new Uint8Array() }
		},
		onSuccess: () => {
			setStep("recovery-gate")
		},
	})

	const handleVerify = () => {
		setError("")
		verifyWords.mutate()
	}

	const handleNewKey = (vaultKey: string, keyType: KeyType) => {
		setStep("setting-up")
		setupNewKey.mutate({ vaultKey, keyType })
	}

	const handleRecoveryConfirm = () => {
		setNewMnemonic("")
		navigate({ to: "/login" })
	}

	if (step === "verifying") {
		return (
			<div className="flex min-h-screen items-center justify-center px-4">
				<div className="flex flex-col items-center gap-4">
					<Spinner size="lg" />
					<p className="text-sm text-muted-foreground">Verifying recovery words...</p>
				</div>
			</div>
		)
	}

	if (step === "setting-up") {
		return (
			<div className="flex min-h-screen items-center justify-center px-4">
				<div className="flex flex-col items-center gap-4">
					<Spinner size="lg" />
					<p className="text-sm text-muted-foreground">Deriving new keys...</p>
				</div>
			</div>
		)
	}

	if (step === "recovery-gate") {
		return (
			<RecoveryKitModal
				email={me?.email ?? ""}
				mnemonic={newMnemonic}
				onConfirm={handleRecoveryConfirm}
			/>
		)
	}

	if (step === "new-key") {
		return (
			<div className="flex min-h-screen flex-col bg-background">
				<div className="flex flex-1 items-center justify-center px-4 py-8">
					<div className="w-full max-w-100">
						<CardBox>
							<Card className="p-0">
								<CardHeader className="px-6 pt-6 text-center">
									<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-success/10 text-success">
										<CheckCircle2 className="size-4" />
									</div>
									<CardTitle>Recovery words verified</CardTitle>
									<CardDescription className="text-xs">
										Now set a new Vault Key for your account.
									</CardDescription>
								</CardHeader>

								<CardContent className="px-6 pt-4 pb-6">
									{setupNewKey.error && (
										<Alert variant="danger" className="mb-4">
											<AlertCircle />
											<AlertDescription>{setupNewKey.error.message}</AlertDescription>
										</Alert>
									)}
									<NewVaultKeyForm
										onSubmit={handleNewKey}
										isLoading={setupNewKey.isPending}
										loadingText="Setting new key..."
										submitLabel="Set new Vault Key"
									/>
								</CardContent>
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

	// Step: enter-words
	return (
		<div className="flex min-h-screen flex-col bg-background">
			<div className="flex flex-1 items-center justify-center px-4 py-8">
				<div className="w-full max-w-lg">
					<CardBox>
						<Card className="p-0">
							<CardHeader className="px-6 pt-6 text-center">
								<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
									<KeyRound className="size-4" />
								</div>
								<CardTitle>Enter your recovery words</CardTitle>
								<CardDescription className="text-xs">
									Type or paste the 24 words from your Recovery Kit PDF.
								</CardDescription>
							</CardHeader>

							<CardContent className="px-6 pt-4 pb-6">
								{error && (
									<Alert variant="danger" className="mb-4">
										<AlertCircle />
										<AlertDescription>{error}</AlertDescription>
									</Alert>
								)}

								<MnemonicInput
									words={words}
									onChange={setWords}
									disabled={verifyWords.isPending}
								/>

								<Button
									variant="solid"
									size="sm"
									onClick={handleVerify}
									disabled={!allFilled}
									isLoading={verifyWords.isPending}
									loadingText="Verifying..."
									className="mt-4 w-full"
								>
									Verify recovery words
								</Button>
							</CardContent>

							<div className="border-t border-border bg-muted/30 px-6 py-3 text-center text-xs text-muted-foreground">
								<Link
									to="/recover"
									className="font-medium text-primary hover:underline"
								>
									Back to recovery options
								</Link>
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
