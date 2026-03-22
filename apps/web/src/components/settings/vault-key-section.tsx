import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { SettingsRow, SettingsDivider } from "./settings-row"
import { NewVaultKeyForm } from "#/components/NewVaultKeyForm"
import { meQueryOptions } from "#/lib/queries/auth"
import { useAuthStore } from "#/lib/stores/auth"
import { api } from "#/lib/api-client"
import {
	deriveKeys,
	hashAuthKey,
	wrapKey,
	generateSalt,
	type KeyType,
} from "@zenv/amnesia"
import { AlertCircle, CheckCircle, KeyRound } from "lucide-react"

// ── Helpers ──

function toBase64(bytes: Uint8Array): string {
	let binary = ""
	for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i])
	return btoa(binary)
}

function pack(nonce: Uint8Array, ciphertext: Uint8Array): Uint8Array {
	const out = new Uint8Array(nonce.length + ciphertext.length)
	out.set(nonce, 0)
	out.set(ciphertext, nonce.length)
	return out
}

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

// ── Change Key ──

function ChangeKeyRow() {
	const qc = useQueryClient()
	const crypto = useAuthStore((s) => s.crypto)
	const [step, setStep] = useState<"idle" | "form">("idle")

	const changeKey = useMutation({
		mutationFn: async ({ vaultKey, keyType }: { vaultKey: string; keyType: KeyType }) => {
			if (!crypto) throw new Error("Vault is locked")

			// O(1) re-wrap: derive new KEK from new Vault Key, re-wrap same DEK
			const newSalt = generateSalt()
			const { kek: newKek, authKey: newAuthKey } = await deriveKeys(vaultKey, newSalt, keyType)
			const newAuthKeyHash = await hashAuthKey(newAuthKey)

			const { ciphertext, nonce } = await wrapKey(crypto.dek, newKek)
			const wrappedDek = pack(nonce, ciphertext)

			const { error } = await api().PUT("/auth/change-vault-key", {
				body: {
					vault_key_type: keyType,
					salt: toBase64(newSalt),
					auth_key_hash: toBase64(newAuthKeyHash),
					wrapped_dek: toBase64(wrappedDek),
				} as never,
			})
			if (error) throw new Error("Failed to change Vault Key")

			// Update local KEK
			useAuthStore.getState().setCrypto({ ...crypto, kek: newKek })
		},
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["auth", "me"] })
			setStep("idle")
		},
	})

	if (step === "idle") {
		return (
			<SettingsRow
				title="Change Vault Key"
				description="Re-wraps your DEK with a new key in one round trip. Zero item rows are touched — O(1) regardless of how many secrets you have."
			>
				{changeKey.isSuccess && (
					<Alert variant="success" className="mb-4">
						<CheckCircle />
						<AlertDescription>Vault Key changed successfully.</AlertDescription>
					</Alert>
				)}
				{changeKey.error && (
					<Alert variant="danger" className="mb-4">
						<AlertCircle />
						<AlertDescription>{changeKey.error.message}</AlertDescription>
					</Alert>
				)}

				<Button variant="outline" size="sm" onClick={() => setStep("form")}>
					Change Vault Key
				</Button>
			</SettingsRow>
		)
	}

	return (
		<SettingsRow
			title="New Vault Key"
			description="Choose a new PIN or passphrase. You can switch between formats freely — the DEK doesn't change."
		>
			<div className="space-y-4">
				<NewVaultKeyForm
					onSubmit={(vaultKey, keyType) => changeKey.mutate({ vaultKey, keyType })}
					isLoading={changeKey.isPending}
					submitLabel="Change Vault Key"
				/>
				<Button variant="ghost" size="xs" onClick={() => setStep("idle")}>
					Cancel
				</Button>
			</div>
		</SettingsRow>
	)
}
