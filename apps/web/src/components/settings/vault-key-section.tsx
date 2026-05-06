import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { mutationKeys, queryKeys } from "#/lib/keys"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { SettingsRow, SettingsDivider } from "./settings-row"
import { MnemonicInput } from "#/components/MnemonicInput"
import { NewVaultKeyForm } from "#/components/NewVaultKeyForm"
import { meQueryOptions } from "#/lib/queries/auth"
import { useAuthStore } from "#/lib/stores/auth"
import { api } from "#/lib/api-client"
import {
	wordsToEntropy,
	unwrapDekFromRecovery,
	MNEMONIC_WORD_COUNT,
} from "#/lib/recovery"
import {
	hashAuthKey,
	wrapKey,
	generateSalt,
	type KeyType,
} from "@zenv/amnesia"
import { deriveKeysAsync } from "#/lib/derive-keys"
import { toBase64, fromBase64, pack } from "#/lib/encoding"
import { AlertCircle, CheckCircle, KeyRound } from "lucide-react"

// ── Component ──

export function VaultKeySection() {
	const { data: me } = useQuery(meQueryOptions)
	const currentType = me?.vault_key_type ?? "pin"

	return (
		<div>
			<CurrentKeyRow keyType={currentType} />
			<SettingsDivider />
			<ChangeKeyRow />
		</div>
	)
}

// ── Current Key Info ──

function CurrentKeyRow({ keyType }: { keyType: string }) {
	return (
		<SettingsRow
			title="Current Vault Key"
			description="Your Vault Key derives the encryption key that protects your secrets. zEnv never sees or stores it."
		>
			<div className="flex items-center gap-3 rounded-md border border-border px-3 py-2.5">
				<KeyRound className="size-4 text-muted-foreground" />
				<div>
					<p className="text-sm font-medium">
						{keyType === "pin" ? "PIN" : "Passphrase"}
					</p>
					<p className="text-xs text-muted-foreground">
						{keyType === "pin"
							? "6+ digit numeric PIN with aggressive Argon2id parameters."
							: "12+ character passphrase with standard Argon2id parameters."}
					</p>
				</div>
				<Badge variant="neutral" className="ml-auto">{keyType}</Badge>
			</div>
		</SettingsRow>
	)
}

// ── Change Key (requires mnemonic verification first) ──

type ChangeStep = "idle" | "verify-mnemonic" | "new-key"

function ChangeKeyRow() {
	const qc = useQueryClient()
	const crypto = useAuthStore((s) => s.crypto)
	const [step, setStep] = useState<ChangeStep>("idle")
	const [words, setWords] = useState<string[]>(Array(MNEMONIC_WORD_COUNT).fill(""))
	const [verifiedDek, setVerifiedDek] = useState<Uint8Array | null>(null)

	const allFilled = words.every((w) => w.length >= 3)

	const verifyMnemonic = useMutation({
		mutationKey: mutationKeys.auth.verifyMnemonic,
		mutationFn: async () => {
			const mnemonic = words.join(" ")
			const entropy = wordsToEntropy(mnemonic)

			// @ts-ignore types will be regenerated
			const { data, error: fetchErr } = await api().GET("/auth/recovery/kit")
			if (fetchErr || !data) throw new Error("Failed to fetch recovery material")

			const blob = fromBase64((data as { recovery_wrapped_dek: string }).recovery_wrapped_dek)
			return await unwrapDekFromRecovery(blob, entropy)
		},
		onSuccess: (dek) => {
			setVerifiedDek(dek)
			setStep("new-key")
		},
	})

	const changeKey = useMutation({
		mutationKey: mutationKeys.auth.changeVaultKey,
		mutationFn: async ({ vaultKey, keyType }: { vaultKey: string; keyType: KeyType }) => {
			const dek = verifiedDek ?? crypto?.dek
			if (!dek) throw new Error("No DEK available")

			const newSalt = generateSalt()
			const { kek: newKek, authKey: newAuthKey } = await deriveKeysAsync(vaultKey, newSalt, keyType)
			const newAuthKeyHash = await hashAuthKey(newAuthKey)

			const { ciphertext, nonce } = await wrapKey(dek, newKek)
			const wrappedDek = pack(nonce, ciphertext)

			const { error } = await api().PUT("/auth/change-vault-key", {
				body: {
					vault_key_type: keyType,
					salt: toBase64(newSalt),
					auth_key_hash: toBase64(newAuthKeyHash),
					wrapped_dek: toBase64(wrappedDek),
				},
			})
			if (error) throw new Error("Failed to change Vault Key")

			if (crypto) {
				useAuthStore.getState().setCrypto({ ...crypto, kek: newKek })
			}
		},
		onSuccess: async () => {
			await qc.invalidateQueries({ queryKey: queryKeys.auth.me })
			setStep("idle")
			setWords(Array(MNEMONIC_WORD_COUNT).fill(""))
			setVerifiedDek(null)
		},
	})

	const handleCancel = () => {
		setStep("idle")
		setWords(Array(MNEMONIC_WORD_COUNT).fill(""))
		setVerifiedDek(null)
	}

	if (step === "idle") {
		return (
			<SettingsRow
				title="Change Vault Key"
				description="To change your Vault Key, you must first verify your Recovery Kit. This ensures you always have a working recovery path."
			>
				{changeKey.isSuccess && (
					<Alert variant="success" className="mb-4">
						<CheckCircle />
						<AlertDescription>Vault Key changed successfully.</AlertDescription>
					</Alert>
				)}

				<Button variant="outline" size="sm" onClick={() => setStep("verify-mnemonic")}>
					Change Vault Key
				</Button>
			</SettingsRow>
		)
	}

	if (step === "verify-mnemonic") {
		return (
			<SettingsRow
				title="Verify your Recovery Kit"
				description="Enter your 12 recovery words to prove you have a working recovery path before changing your Vault Key."
			>
				<div className="space-y-4">
					{verifyMnemonic.error && (
						<Alert variant="danger">
							<AlertCircle />
							<AlertDescription>Invalid recovery words. Check and try again.</AlertDescription>
						</Alert>
					)}

					<MnemonicInput words={words} onChange={setWords} disabled={verifyMnemonic.isPending} />

					<div className="flex gap-2">
						<Button
							variant="solid"
							size="sm"
							onClick={() => verifyMnemonic.mutate()}
							disabled={!allFilled}
							isLoading={verifyMnemonic.isPending}
							loadingText="Verifying..."
						>
							Verify
						</Button>
						<Button variant="ghost" size="sm" onClick={handleCancel}>
							Cancel
						</Button>
					</div>
				</div>
			</SettingsRow>
		)
	}

	// step === "new-key"
	return (
		<SettingsRow
			title="Set new Vault Key"
			description="Recovery Kit verified. Choose a new PIN or passphrase — the DEK stays the same, only the wrapper changes."
		>
			<div className="space-y-4">
				{changeKey.error && (
					<Alert variant="danger">
						<AlertCircle />
						<AlertDescription>{changeKey.error.message}</AlertDescription>
					</Alert>
				)}

				<Alert variant="success" className="text-xs">
					<CheckCircle />
					<AlertDescription>Recovery words verified.</AlertDescription>
				</Alert>

				<NewVaultKeyForm
					onSubmit={(vaultKey, keyType) => changeKey.mutate({ vaultKey, keyType })}
					isLoading={changeKey.isPending}
					submitLabel="Change Vault Key"
				/>

				<Button variant="ghost" size="xs" onClick={handleCancel}>
					Cancel
				</Button>
			</div>
		</SettingsRow>
	)
}
