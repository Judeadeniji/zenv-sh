import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "#/components/ui/tabs"

const pinSchema = z
	.object({
		pin: z.string().min(6, "PIN must be at least 6 digits").regex(/^\d+$/, "PIN must contain only digits"),
		confirmPin: z.string(),
	})
	.refine((d) => d.pin === d.confirmPin, { message: "PINs don't match", path: ["confirmPin"] })

const passphraseSchema = z
	.object({
		passphrase: z.string().min(12, "Passphrase must be at least 12 characters"),
		confirmPassphrase: z.string(),
	})
	.refine((d) => d.passphrase === d.confirmPassphrase, {
		message: "Passphrases don't match",
		path: ["confirmPassphrase"],
	})

type PinInput = z.infer<typeof pinSchema>
type PassphraseInput = z.infer<typeof passphraseSchema>

interface NewVaultKeyFormProps {
	onSubmit: (vaultKey: string, keyType: "pin" | "passphrase") => void
	isLoading?: boolean
	loadingText?: string
	submitLabel?: string
}

export function NewVaultKeyForm({
	onSubmit,
	isLoading,
	loadingText = "Setting up vault...",
	submitLabel = "Continue",
}: NewVaultKeyFormProps) {
	const [keyType, setKeyType] = useState<"pin" | "passphrase">("passphrase")

	const pinForm = useForm<PinInput>({
		resolver: zodResolver(pinSchema),
		defaultValues: { pin: "", confirmPin: "" },
	})

	const passphraseForm = useForm<PassphraseInput>({
		resolver: zodResolver(passphraseSchema),
		defaultValues: { passphrase: "", confirmPassphrase: "" },
	})

	const handlePinSubmit = (data: PinInput) => onSubmit(data.pin, "pin")
	const handlePassphraseSubmit = (data: PassphraseInput) => onSubmit(data.passphrase, "passphrase")

	return (
		<Tabs value={keyType} onValueChange={(v) => setKeyType(v as "pin" | "passphrase")}>
			<TabsList variant="line" className="mb-4 w-full">
				<TabsTrigger value="passphrase">Passphrase</TabsTrigger>
				<TabsTrigger value="pin">PIN</TabsTrigger>
			</TabsList>

			<TabsContent value="passphrase">
				<form onSubmit={passphraseForm.handleSubmit(handlePassphraseSubmit)} className="grid gap-3">
					<div className="space-y-1.5">
						<Label htmlFor="passphrase">Passphrase</Label>
						<Input
							id="passphrase"
							type="password"
							placeholder="At least 12 characters"
							{...passphraseForm.register("passphrase")}
							feedback={passphraseForm.formState.errors.passphrase ? "error" : undefined}
						/>
						{passphraseForm.formState.errors.passphrase && (
							<p className="text-xs text-destructive">{passphraseForm.formState.errors.passphrase.message}</p>
						)}
					</div>

					<div className="space-y-1.5">
						<Label htmlFor="confirm-passphrase">Confirm passphrase</Label>
						<Input
							id="confirm-passphrase"
							type="password"
							placeholder="Re-enter passphrase"
							{...passphraseForm.register("confirmPassphrase")}
							feedback={passphraseForm.formState.errors.confirmPassphrase ? "error" : undefined}
						/>
						{passphraseForm.formState.errors.confirmPassphrase && (
							<p className="text-xs text-destructive">
								{passphraseForm.formState.errors.confirmPassphrase.message}
							</p>
						)}
					</div>

					<Button type="submit" variant="solid" size="md" isLoading={isLoading} loadingText={loadingText} className="mt-1 w-full">
						{submitLabel}
					</Button>
				</form>
			</TabsContent>

			<TabsContent value="pin">
				<form onSubmit={pinForm.handleSubmit(handlePinSubmit)} className="grid gap-3">
					<div className="space-y-1.5">
						<Label htmlFor="pin">PIN</Label>
						<Input
							id="pin"
							type="password"
							inputMode="numeric"
							pattern="[0-9]*"
							placeholder="At least 6 digits"
							{...pinForm.register("pin")}
							feedback={pinForm.formState.errors.pin ? "error" : undefined}
						/>
						{pinForm.formState.errors.pin && (
							<p className="text-xs text-destructive">{pinForm.formState.errors.pin.message}</p>
						)}
					</div>

					<div className="space-y-1.5">
						<Label htmlFor="confirm-pin">Confirm PIN</Label>
						<Input
							id="confirm-pin"
							type="password"
							inputMode="numeric"
							pattern="[0-9]*"
							placeholder="Re-enter PIN"
							{...pinForm.register("confirmPin")}
							feedback={pinForm.formState.errors.confirmPin ? "error" : undefined}
						/>
						{pinForm.formState.errors.confirmPin && (
							<p className="text-xs text-destructive">{pinForm.formState.errors.confirmPin.message}</p>
						)}
					</div>

					<Button type="submit" variant="solid" size="md" isLoading={isLoading} loadingText={loadingText} className="mt-1 w-full">
						{submitLabel}
					</Button>
				</form>
			</TabsContent>
		</Tabs>
	)
}
