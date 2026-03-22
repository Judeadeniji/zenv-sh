import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { CardBox, Card, CardHeader, CardTitle, CardDescription, CardContent } from "#/components/ui/card"
import { Alert, AlertDescription } from "#/components/ui/alert"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Textarea } from "#/components/ui/textarea"
import { Label } from "#/components/ui/label"
import { Separator } from "#/components/ui/separator"
import { useCreateOrg } from "#/lib/queries/orgs"
import { useCreateProject } from "#/lib/queries/projects"
import { useNavStore } from "#/lib/stores/nav"
import {
	createOrgSchema,
	createProjectSchema,
	type CreateOrgInput,
	type CreateProjectInput,
} from "#/lib/schemas/onboarding"
import { AlertCircle, Building2, FolderKey, FileUp, ArrowRight, Check } from "lucide-react"

export const Route = createFileRoute("/_authed/_unlocked/onboarding")({
	component: OnboardingWizard,
})

type Step = "org" | "project" | "import"

function OnboardingWizard() {
	const navigate = useNavigate()
	const [step, setStep] = useState<Step>("org")
	const [orgId, setOrgId] = useState("")

	const createOrg = useCreateOrg()
	const createProject = useCreateProject()

	const orgForm = useForm<CreateOrgInput>({
		resolver: zodResolver(createOrgSchema),
		defaultValues: { name: "" },
	})

	const projectForm = useForm<CreateProjectInput>({
		resolver: zodResolver(createProjectSchema),
		defaultValues: { name: "" },
	})

	const handleCreateOrg = (data: CreateOrgInput) => {
		createOrg.mutate(
			{ name: data.name },
			{
				onSuccess: (org) => {
					const id = org.id ?? ""
					setOrgId(id)
					useNavStore.getState().setActiveOrg(id)
					setStep("project")
				},
			},
		)
	}

	const handleCreateProject = (data: CreateProjectInput) => {
		createProject.mutate(
			{ name: data.name, orgId },
			{
				onSuccess: (result) => {
					const id = result.id ?? ""
					useNavStore.getState().setActiveProject(id)
					setStep("import")
				},
			},
		)
	}

	const handleFinish = () => {
		navigate({ to: "/" })
	}

	const steps = [
		{ key: "org", label: "Organization" },
		{ key: "project", label: "Project" },
		{ key: "import", label: "Import" },
	]
	const currentIdx = steps.findIndex((s) => s.key === step)

	const stepIcons = {
		org: Building2,
		project: FolderKey,
		import: FileUp,
	}
	const StepIcon = stepIcons[step]

	const stepTitles = {
		org: "Create your organization",
		project: "Create your first project",
		import: "Import your .env file",
	}

	const stepDescriptions = {
		org: "Organizations group your team and projects together.",
		project: "Projects contain your secrets, organized by environment.",
		import: "Paste your .env contents to import secrets. You can always do this later.",
	}

	return (
		<div className="flex min-h-[calc(100vh-8rem)] flex-col items-center justify-center px-4 py-8">
			{/* Progress */}
			<div className="mb-8 flex items-center justify-center gap-2">
				{steps.map((s, i) => (
					<div key={s.key} className="flex items-center gap-2">
						<div
							className={`flex h-6 w-6 items-center justify-center rounded-full text-xs font-medium ${
								i < currentIdx
									? "bg-primary text-primary-foreground"
									: i === currentIdx
										? "bg-primary text-primary-foreground"
										: "bg-muted text-muted-foreground"
							}`}
						>
							{i < currentIdx ? <Check className="size-3" /> : i + 1}
						</div>
						<span className={`text-xs ${i <= currentIdx ? "text-foreground" : "text-muted-foreground"}`}>
							{s.label}
						</span>
						{i < steps.length - 1 && <Separator className="w-6" />}
					</div>
				))}
			</div>

			<div className="w-full max-w-100">
				<CardBox>
					<Card className="p-0">
						<CardHeader className="px-6 pt-6 text-center">
							<div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
								<StepIcon className="size-4" />
							</div>
							<CardTitle>{stepTitles[step]}</CardTitle>
							<CardDescription className="text-xs">{stepDescriptions[step]}</CardDescription>
						</CardHeader>

						<CardContent className="px-6 pt-4 pb-6">
							{/* Step 1: Create Org */}
							{step === "org" && (
								<form onSubmit={orgForm.handleSubmit(handleCreateOrg)} className="grid gap-3">
									{createOrg.error && (
										<Alert variant="danger">
											<AlertCircle />
											<AlertDescription>{createOrg.error.message}</AlertDescription>
										</Alert>
									)}

									<div className="space-y-1.5">
										<Label htmlFor="org-name" className="text-xs">Organization name</Label>
										<Input
											id="org-name"
											placeholder="Acme Inc"
											{...orgForm.register("name")}
											feedback={orgForm.formState.errors.name ? "error" : undefined}
											autoFocus
										/>
										{orgForm.formState.errors.name && (
											<p className="text-xs text-destructive">{orgForm.formState.errors.name.message}</p>
										)}
									</div>

									<Button type="submit" variant="solid" isLoading={createOrg.isPending} className="mt-1 w-full">
										Continue
										<ArrowRight />
									</Button>
								</form>
							)}

							{/* Step 2: Create Project */}
							{step === "project" && (
								<form onSubmit={projectForm.handleSubmit(handleCreateProject)} className="grid gap-3">
									{createProject.error && (
										<Alert variant="danger">
											<AlertCircle />
											<AlertDescription>{createProject.error.message}</AlertDescription>
										</Alert>
									)}

									<div className="space-y-1.5">
										<Label htmlFor="project-name" className="text-xs">Project name</Label>
										<Input
											id="project-name"
											placeholder="my-app"
											{...projectForm.register("name")}
											feedback={projectForm.formState.errors.name ? "error" : undefined}
											autoFocus
										/>
										{projectForm.formState.errors.name && (
											<p className="text-xs text-destructive">{projectForm.formState.errors.name.message}</p>
										)}
										<p className="text-xs text-muted-foreground">Lowercase letters, numbers, and hyphens.</p>
									</div>

									<Button type="submit" variant="solid" isLoading={createProject.isPending} className="mt-1 w-full">
										Continue
										<ArrowRight />
									</Button>
								</form>
							)}

							{/* Step 3: Import .env (optional) */}
							{step === "import" && (
								<div className="grid gap-3">
									<Textarea
										placeholder={"DATABASE_URL=postgres://...\nAPI_KEY=sk_live_...\nSECRET_TOKEN=abc123"}
										className="min-h-30 font-mono text-xs"
										id="env-import"
									/>
									<p className="text-xs text-muted-foreground">
										Each line should be in KEY=VALUE format. Comments (#) and empty lines are ignored.
									</p>

									{/* TODO: Wire up .env parsing + encryption + bulk create */}
									<Button variant="solid" onClick={handleFinish} className="w-full">
										Go to dashboard
										<ArrowRight />
									</Button>
								</div>
							)}
						</CardContent>

						{step === "import" && (
							<div className="border-t border-border bg-muted/30 px-6 py-3 text-center text-xs text-muted-foreground">
								<button
									type="button"
									onClick={handleFinish}
									className="font-medium text-primary hover:underline"
								>
									Skip for now
								</button>
							</div>
						)}
					</Card>
				</CardBox>
			</div>
		</div>
	)
}
