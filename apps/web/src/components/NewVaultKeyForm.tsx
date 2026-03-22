import { useState } from "react"
import { useForm, Controller } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "#/components/ui/button"
import { PasswordInput } from "#/components/ui/password-input"
import { Label } from "#/components/ui/label"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "#/components/ui/tabs"
import { InputOTP, InputOTPGroup, InputOTPSlot, InputOTPSeparator } from "#/components/ui/input-otp"

const PIN_LENGTH = 6

const pinSchema = z
	.object({
		pin: z
			.string()
			.length(PIN_LENGTH, `PIN must be ${PIN_LENGTH} digits`)
			.regex(/^\d+$/, "PIN must contain only digits"),
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

function PinField({
	label,
	value,
	onChange,
	error,
	autoFocus,
}: {
	label: string
	value: string
	onChange: (value: string) => void
	error?: string
	autoFocus?: boolean
}) {
	return (
		<div className="space-y-1.5">
			<Label className="text-xs text-center">{label}</Label>
			<InputOTP
				maxLength={PIN_LENGTH}
				value={value}
				onChange={onChange}
				inputMode="numeric"
				pattern="[0-9]*"
				autoFocus={autoFocus}
				containerClassName="justify-center w-full mt-3"
				textAlign="center"
				pushPasswordManagerStrategy="none"
			>
				<InputOTPGroup>
					<InputOTPSlot index={0} masked className="size-10 text-lg" />
					<InputOTPSlot index={1} masked className="size-10 text-lg" />
					<InputOTPSlot index={2} masked className="size-10 text-lg" />
				</InputOTPGroup>
				<InputOTPSeparator />
				<InputOTPGroup>
					<InputOTPSlot index={3} masked className="size-10 text-lg" />
					<InputOTPSlot index={4} masked className="size-10 text-lg" />
					<InputOTPSlot index={5} masked className="size-10 text-lg" />
				</InputOTPGroup>
			</InputOTP>
			{error && <p className="text-center text-xs text-destructive">{error}</p>}
		</div>
	)
}

export function NewVaultKeyForm({
	onSubmit,
	isLoading,
	loadingText = "Setting up vault...",
	submitLabel = "Continue",
}: NewVaultKeyFormProps) {
	const [keyType, setKeyType] = useState<"pin" | "passphrase">("passphrase")
	const [pinStep, setPinStep] = useState<"enter" | "confirm">("enter")

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
						<Label htmlFor="passphrase" className="text-xs">Passphrase</Label>
						<PasswordInput
							id="passphrase"
							placeholder="At least 12 characters"
							{...passphraseForm.register("passphrase")}
							feedback={passphraseForm.formState.errors.passphrase ? "error" : undefined}
						/>
						{passphraseForm.formState.errors.passphrase && (
							<p className="text-xs text-destructive">{passphraseForm.formState.errors.passphrase.message}</p>
						)}
					</div>

					<div className="space-y-1.5">
						<Label htmlFor="confirm-passphrase" className="text-xs">Confirm passphrase</Label>
						<PasswordInput
							id="confirm-passphrase"
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

					<Button type="submit" variant="solid" isLoading={isLoading} loadingText={loadingText} className="mt-1 w-full">
						{submitLabel}
					</Button>
				</form>
			</TabsContent>

			<TabsContent value="pin">
				<form onSubmit={pinForm.handleSubmit(handlePinSubmit)} className="grid gap-4">
					{pinStep === "enter" ? (
						<Controller
							control={pinForm.control}
							name="pin"
							render={({ field, fieldState }) => (
								<PinField
									label="Enter a 6-digit PIN"
									value={field.value}
									onChange={(val) => {
										field.onChange(val)
										if (val.length === PIN_LENGTH) {
											setPinStep("confirm")
										}
									}}
									error={fieldState.error?.message}
									autoFocus
								/>
							)}
						/>
					) : (
						<>
							<Controller
								control={pinForm.control}
								name="confirmPin"
								render={({ field, fieldState }) => (
									<PinField
										label="Confirm your PIN"
										value={field.value}
										onChange={field.onChange}
										error={fieldState.error?.message || (pinForm.formState.errors.confirmPin?.message)}
										autoFocus
									/>
								)}
							/>
							<button
								type="button"
								className="text-xs text-muted-foreground hover:text-foreground"
								onClick={() => {
									pinForm.setValue("confirmPin", "")
									setPinStep("enter")
								}}
							>
								Re-enter PIN
							</button>
						</>
					)}

					<Button
						type="submit"
						variant="solid"
						isLoading={isLoading}
						loadingText={loadingText}
						className="mt-1 w-full"
						disabled={pinStep === "enter"}
					>
						{submitLabel}
					</Button>
				</form>
			</TabsContent>
		</Tabs>
	)
}
