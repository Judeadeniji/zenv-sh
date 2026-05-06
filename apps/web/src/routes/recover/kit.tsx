import { useState } from "react"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { useMutation, useQuery } from "@tanstack/react-query"
import { z } from "zod"
import { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Button } from "#/components/ui/button"
import { Spinner } from "#/components/ui/spinner"
import { MnemonicInput } from "#/components/MnemonicInput"
import { NewVaultKeyForm } from "#/components/NewVaultKeyForm"
import { RecoveryKitModal } from "#/components/RecoveryKitModal"
import {
	wordsToEntropy,
	unwrapDekFromRecovery,
	generateRecoveryEntropy,
	entropyToWords,
	wrapDekForRecovery,
	MNEMONIC_WORD_COUNT,
} from "#/lib/recovery"
import { meQueryOptions } from "#/lib/queries/auth"
import { api } from "#/lib/api-client"
import { hashAuthKey, wrapKey, unwrapKey, generateSalt } from "@zenv/amnesia"
import { deriveKeysAsync } from "#/lib/derive-keys"
import type { KeyType } from "@zenv/amnesia"
import { mutationKeys } from "#/lib/keys"
import { toBase64, fromBase64 } from "#/lib/encoding"
import { AlertCircle, KeyRound, CheckCircle2, RefreshCw } from "lucide-react"

const searchSchema = z.object({
	regenerate: z.coerce.boolean().optional(),
})

export const Route = createFileRoute("/recover/kit")({
	validateSearch: searchSchema,
	component: RecoverKitPage,
})

function RecoverKitPage() {
	const { regenerate } = Route.useSearch()

	if (regenerate) {
		return <RegenerateFlow />
	}
	return <RecoverFlow />
}

// ════════════════════════════════════════
// Regenerate Flow — user is unlocked, wants new 12 words
// ════════════════════════════════════════

function RegenerateFlow() {
	const navigate = useNavigate()
	const { data: me } = useQuery(meQueryOptions)

	const [step, setStep] = useState<"verify-key" | "generating" | "gate">("verify-key")
	const [mnemonic, setMnemonic] = useState("")
	const [verifyError, setVerifyError] = useState("")

	// Step 1: Verify vault key → derive DEK
	const verifyKey = useMutation({
		mutationKey: mutationKeys.recovery.verifyVaultKey,
		mutationFn: async ({ vaultKey, keyType }: { vaultKey: string; keyType: KeyType }) => {
			if (!me?.salt) throw new Error("Missing vault data")

			const salt = fromBase64(me.salt)
			const { kek, authKey } = await deriveKeysAsync(vaultKey, salt, keyType as KeyType)
			const authKeyHash = await hashAuthKey(authKey)

			// Verify against server
			const { data, error } = await api().POST("/auth/unlock", {
				body: { auth_key_hash: toBase64(authKeyHash) },
			})	
			if (error || !data) throw new Error("Wrong Vault Key")

			const res = data
			const wd = fromBase64(res.wrapped_dek!)
			const wdNonce = wd.slice(0, 12)
			const wdCt = wd.slice(12)

			return await unwrapKey(wdCt, wdNonce, kek)
		},
	})

	// Step 2: Generate new recovery kit using verified DEK
	const regenerate = useMutation({
		mutationKey: mutationKeys.recovery.regenerateKit,
		mutationFn: async (dek: Uint8Array) => {
			const entropy = generateRecoveryEntropy()
			const words = entropyToWords(entropy)
			const blob = await wrapDekForRecovery(dek, entropy)

			const { error } = await api().PUT("/auth/recovery/kit", {
				body: { recovery_wrapped_dek: toBase64(blob) },
			})
			if (error) throw new Error("Failed to save new recovery kit")

			return words
		},
		onSuccess: (words) => {
			setMnemonic(words)
			setStep("gate")
		},
	})

	const handleKeyVerified = (vaultKey: string, keyType: KeyType) => {
		setVerifyError("")
		verifyKey.mutate(
			{ vaultKey, keyType },
			{
				onSuccess: (dek) => {
					setStep("generating")
					regenerate.mutate(dek)
				},
				onError: () => {
					setVerifyError("Wrong Vault Key. Please try again.")
				},
			},
		)
	}

	if (step === "generating") {
		return (
			<div className="flex min-h-screen items-center justify-center px-4">
				<div className="flex flex-col items-center gap-4">
					<Spinner size="lg" />
					<p className="text-sm text-muted-foreground">Generating new recovery kit...</p>
				</div>
			</div>
		)
	}

	if (step === "gate") {
		return (
			<RecoveryKitModal
				email={me?.email ?? ""}
				mnemonic={mnemonic}
				onConfirm={() => {
					setMnemonic("")
					navigate({ to: "/settings", search: { tab: "recovery" } })
				}}
			/>
		)
	}

	// Step: verify-key
	return (
		<div className="flex min-h-screen flex-col bg-background">
			<div className="flex flex-1 items-center justify-center px-4 py-8">
				<div className="w-full max-w-100">
					<CardBox>
						<Card className="p-0">
							<CardHeader className="px-6 pt-6 text-center">
								<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
									<RefreshCw className="size-4" />
								</div>
								<CardTitle>Regenerate Recovery Kit</CardTitle>
								<CardDescription className="text-xs">
									Enter your Vault Key to verify your identity before generating new recovery words.
								</CardDescription>
							</CardHeader>

							<CardContent className="px-6 pt-4 pb-6">
								{(verifyError || regenerate.error) && (
									<Alert variant="danger" className="mb-4">
										<AlertCircle />
										<AlertDescription>
											{verifyError || regenerate.error?.message}
										</AlertDescription>
									</Alert>
								)}

								<Alert variant="warning" className="mb-4 text-xs">
									<AlertCircle />
									<AlertDescription>
										Your old recovery words will stop working immediately.
									</AlertDescription>
								</Alert>

								<NewVaultKeyForm
									onSubmit={handleKeyVerified}
									isLoading={verifyKey.isPending}
									submitLabel="Verify & regenerate"
									loadingText="Verifying..."
									confirmMode={false}
								/>

								<div className="mt-3 text-center">
									<Button
										variant="ghost"
										size="xs"
										render={<Link to="/settings" search={{ tab: "recovery" }} />}
									>
										Cancel
									</Button>
								</div>
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

// ════════════════════════════════════════
// Recover Flow — user forgot vault key, enters old words
// ════════════════════════════════════════

type RecoverStep = "enter-words" | "verifying" | "new-key" | "setting-up" | "recovery-gate"

function RecoverFlow() {
	const navigate = useNavigate()
	const { data: me } = useQuery(meQueryOptions)

	const [step, setStep] = useState<RecoverStep>("enter-words")
	const [words, setWords] = useState<string[]>(Array(MNEMONIC_WORD_COUNT).fill(""))
	const [dek, setDek] = useState<Uint8Array | null>(null)
	const [newMnemonic, setNewMnemonic] = useState("")
	const [error, setError] = useState("")

	const allFilled = words.every((w) => w.length >= 3)

	const verifyWords = useMutation({
		mutationKey: mutationKeys.recovery.recoverWithKit,
		mutationFn: async () => {
			const mnemonic = words.join(" ")
			const recoveryKey = wordsToEntropy(mnemonic)

			const { data, error: fetchErr } = await api().GET("/auth/recovery/kit")
			if (fetchErr || !data) throw new Error("Failed to fetch recovery material")

			const res = data
			const blob = fromBase64(res.recovery_wrapped_dek!)
			return await unwrapDekFromRecovery(blob, recoveryKey)
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
			const { kek, authKey } = await deriveKeysAsync(vaultKey, salt, keyType)
			const authKeyHash = await hashAuthKey(authKey)

			const { ciphertext: wdCt, nonce: wdNonce } = await wrapKey(dek, kek)
			const wrappedDEK = new Uint8Array(wdNonce.length + wdCt.length)
			wrappedDEK.set(wdNonce, 0)
			wrappedDEK.set(wdCt, wdNonce.length)

			const newEntropy = generateRecoveryEntropy()
			const newMnemonicWords = entropyToWords(newEntropy)
			setNewMnemonic(newMnemonicWords)

			const newRecoveryWrappedDEK = await wrapDekForRecovery(dek, newEntropy)

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
		},
		onSuccess: () => {
			setStep("recovery-gate")
		},
	})

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
				onConfirm={() => {
					setNewMnemonic("")
					navigate({ to: "/login" })
				}}
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
										onSubmit={(vaultKey, keyType) => {
											setStep("setting-up")
											setupNewKey.mutate({ vaultKey, keyType })
										}}
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
									Type or paste the 12 words from your Recovery Kit PDF.
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
									onClick={() => {
										setError("")
										verifyWords.mutate()
									}}
									disabled={!allFilled}
									isLoading={verifyWords.isPending}
									loadingText="Verifying..."
									className="mt-4 w-full"
								>
									Verify recovery words
								</Button>
							</CardContent>

							<div className="border-t border-border bg-muted/30 px-6 py-3 text-center text-xs text-muted-foreground">
								<Link to="/recover" className="font-medium text-primary hover:underline">
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
