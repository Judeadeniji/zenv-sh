import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from "#/components/ui/dialog"
import { Button } from "#/components/ui/button"
import { Textarea } from "#/components/ui/textarea"
import { Label } from "#/components/ui/label"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { useUpdateSecret } from "#/lib/queries/secrets"
import { useProjectDEK } from "#/lib/queries/projects"
import { useNavStore } from "#/lib/stores/nav"
import { updateSecretSchema, type UpdateSecretInput } from "#/lib/schemas/secrets"
import { toast } from "sonner"
import { AlertCircle } from "lucide-react"

interface EditSecretDialogProps {
    projectId: string
    secret: { name_hash: string; name: string; value: string }
    open: boolean
    onOpenChange: (open: boolean) => void
}

export function EditSecretDialog({ projectId, secret, open, onOpenChange }: EditSecretDialogProps) {
    const environment = useNavStore((s) => s.activeEnvironment)
    const { data: projectDEK } = useProjectDEK(projectId)
    const update = useUpdateSecret()

    const form = useForm<UpdateSecretInput>({
        resolver: zodResolver(updateSecretSchema),
        values: { value: secret.value },
    })

    const onSubmit = (data: UpdateSecretInput) => {
        if (!projectDEK) return
        update.mutate(
            {
                projectId,
                environment,
                nameHash: secret.name_hash,
                name: secret.name,
                value: data.value,
                projectDEK,
            },
            {
                onSuccess: () => {
                    onOpenChange(false)
                    toast.success(`Updated ${secret.name}`)
                },
                onError: (err) => toast.error(err.message || "Failed to update secret"),
            },
        )
    }

    return (
        <Dialog open={open} onOpenChange={(v) => { onOpenChange(v); if (!v) { form.reset(); update.reset() } }}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Edit secret</DialogTitle>
                    <DialogDescription>
                        Update the value for <code className="rounded bg-muted px-1 py-0.5 text-xs font-semibold">{secret.name}</code>. The value is re-encrypted on your device.
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-4 py-2">
                    {update.error && (
                        <Alert variant="danger">
                            <AlertCircle />
                            <AlertDescription>{update.error.message}</AlertDescription>
                        </Alert>
                    )}

                    <div className="space-y-1.5">
                        <Label htmlFor="edit-secret-name" className="text-xs">Name</Label>
                        <code className="block rounded bg-muted px-2 py-1.5 font-mono text-xs font-semibold text-muted-foreground">
                            {secret.name}
                        </code>
                    </div>

                    <div className="space-y-1.5">
                        <Label htmlFor="edit-secret-value" className="text-xs">Value</Label>
                        <Textarea
                            id="edit-secret-value"
                            placeholder="The new value to encrypt"
                            className="font-mono text-xs"
                            rows={3}
                            {...form.register("value")}
                            feedback={form.formState.errors.value ? "error" : undefined}
                            autoFocus
                        />
                        {form.formState.errors.value && (
                            <p className="text-xs text-destructive">{form.formState.errors.value.message}</p>
                        )}
                    </div>

                    <DialogFooter>
                        <DialogClose>
                            <Button variant="ghost" size="sm" type="button">Cancel</Button>
                        </DialogClose>
                        <Button type="submit" variant="solid" size="sm" isLoading={update.isPending}>
                            Save changes
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    )
}
