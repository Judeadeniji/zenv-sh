import * as React from "react"
import { createFileRoute } from "@tanstack/react-router"
import {
	AlertCircle, CheckCircle, ChevronRight, Copy, Eye,
	FolderOpen, Info, MoreHorizontal, Plus, Search, Settings,
	Trash2, TriangleAlert, Home,
} from "lucide-react"
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "#/components/ui/accordion"
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert"
import { Avatar } from "#/components/ui/avatar"
import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "#/components/ui/breadcrumb"
import { ActionCard, Card, CardBox, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "#/components/ui/card"
import { Checkbox } from "#/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "#/components/ui/dialog"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "#/components/ui/dropdown-menu"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "#/components/ui/empty"
import { Input } from "#/components/ui/input"
import { InputGroup, InputGroupAddon, InputGroupInput, InputGroupButton, InputGroupText } from "#/components/ui/input-group"
import { InputOTP, InputOTPGroup, InputOTPSlot, InputOTPSeparator } from "#/components/ui/input-otp"
import { Kbd } from "#/components/ui/kbd"
import { Label } from "#/components/ui/label"
import { OneTimeDisplay } from "#/components/ui/one-time-display"
import { PageHeader } from "#/components/ui/page-header"
import { Progress } from "#/components/ui/progress"
import { RadioGroup, RadioGroupItem } from "#/components/ui/radio-group"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "#/components/ui/select"
import { Separator } from "#/components/ui/separator"
import { Skeleton } from "#/components/ui/skeleton"
import { Slider } from "#/components/ui/slider"
import { Spinner } from "#/components/ui/spinner"
import { Switch } from "#/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "#/components/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "#/components/ui/tabs"
import { Textarea } from "#/components/ui/textarea"
import { Toggle } from "#/components/ui/toggle"
import { ToggleGroup, ToggleGroupItem } from "#/components/ui/toggle-group"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "#/components/ui/tooltip"

export const Route = createFileRoute("/components")({ component: ComponentGallery })

function ComponentGallery() {
	return (
		<div className="space-y-16 pb-24">
			<div>
				<h1 className="mb-2 text-2xl font-bold tracking-tight">Component Gallery</h1>
				<p className="text-sm text-muted-foreground">
					zEnv design system — ported from Clerk UI foundations. Shadow-based borders, alpha color scales, polished focus rings.
				</p>
			</div>

			{/* Layout Components */}
			<PageHeaderSection />
			<EmptyStateSection />
			<OneTimeDisplaySection />
			<BreadcrumbSection />

			{/* Primitives */}
			<ButtonSection />
			<SpinnerSection />
			<InputSection />
			<InputGroupSection />
			<InputOTPSection />
			<TextareaSection />
			<SelectSection />
			<CheckboxSection />
			<SwitchSection />
			<RadioGroupSection />
			<ToggleSection />
			<SliderSection />

			{/* Data Display */}
			<BadgeSection />
			<KbdSection />
			<AlertSection />
			<AvatarSection />
			<SeparatorSection />
			<ProgressSection />
			<SkeletonSection />

			{/* Composed Elements */}
			<CardSection />
			<TableSection />
			<AccordionSection />
			<TabsSection />

			{/* Overlays */}
			<DialogSection />
			<DropdownSection />
			<TooltipSection />
		</div>
	)
}

/* ── Page Header ── */
function PageHeaderSection() {
	return (
		<Section title="Page Header" description="Title + description + actions. Used at the top of every page.">
			<div className="space-y-6">
				<PageHeader title="Secrets" description="Manage your encrypted secrets." actions={<Button size="sm"><Plus /> Add Secret</Button>} />
				<Separator />
				<PageHeader title="Project Settings" actions={<><Button variant="outline" size="sm">Cancel</Button><Button variant="danger" size="sm">Delete Project</Button></>} />
			</div>
		</Section>
	)
}

/* ── Empty State ── */
function EmptyStateSection() {
	return (
		<Section title="Empty State" description="Shown when lists are empty. Icon + message + CTA.">
			<div className="max-w-md">
				<Empty className="rounded-lg border border-dashed border-border py-12">
					<EmptyHeader>
						<EmptyMedia variant="icon"><FolderOpen /></EmptyMedia>
						<EmptyTitle>No secrets yet</EmptyTitle>
						<EmptyDescription>Add your first secret to get started. All secrets are encrypted client-side before being stored.</EmptyDescription>
					</EmptyHeader>
					<EmptyContent>
						<Button size="sm"><Plus /> Add Secret</Button>
					</EmptyContent>
				</Empty>
			</div>
		</Section>
	)
}

/* ── One-Time Display ── */
function OneTimeDisplaySection() {
	return (
		<Section title="One-Time Display" description="Reveals a sensitive value once with copy button and warning.">
			<div className="max-w-md space-y-6">
				<OneTimeDisplay label="Service Token" value="zenv_st_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6" warning="This token will only be shown once. Copy it now and store it securely." />
				<OneTimeDisplay label="Vault Key" value="correct-horse-battery-staple-42" masked={false} />
			</div>
		</Section>
	)
}

/* ── Breadcrumb ── */
function BreadcrumbSection() {
	return (
		<Section title="Breadcrumb" description="Navigation path showing org > project > section.">
			<Breadcrumb>
				<BreadcrumbList>
					<BreadcrumbItem><BreadcrumbLink href="#"><Home className="size-3.5" /></BreadcrumbLink></BreadcrumbItem>
					<BreadcrumbSeparator><ChevronRight className="size-3" /></BreadcrumbSeparator>
					<BreadcrumbItem><BreadcrumbLink href="#">Acme Corp</BreadcrumbLink></BreadcrumbItem>
					<BreadcrumbSeparator><ChevronRight className="size-3" /></BreadcrumbSeparator>
					<BreadcrumbItem><BreadcrumbLink href="#">my-api</BreadcrumbLink></BreadcrumbItem>
					<BreadcrumbSeparator><ChevronRight className="size-3" /></BreadcrumbSeparator>
					<BreadcrumbItem><BreadcrumbPage>Secrets</BreadcrumbPage></BreadcrumbItem>
				</BreadcrumbList>
			</Breadcrumb>
		</Section>
	)
}

/* ── Button ── */
function ButtonSection() {
	return (
		<Section title="Button" description="Shadow-based borders, gradient overlay on solid variant.">
			<Subsection title="Variants">
				<Row>
					<Button variant="solid">Solid</Button>
					<Button variant="outline">Outline</Button>
					<Button variant="ghost">Ghost</Button>
					<Button variant="danger">Danger</Button>
					<Button variant="link">Link</Button>
				</Row>
			</Subsection>
			<Subsection title="Sizes">
				<Row>
					<Button size="xs">Extra Small</Button>
					<Button size="sm">Small</Button>
					<Button size="md">Medium</Button>
					<Button size="icon"><Plus /></Button>
					<Button size="icon-sm"><Settings /></Button>
				</Row>
			</Subsection>
			<Subsection title="States">
				<Row>
					<Button disabled>Disabled</Button>
					<Button isLoading>Loading</Button>
					<Button isLoading loadingText="Saving...">Saving</Button>
				</Row>
			</Subsection>
			<Subsection title="All variants × sizes">
				<div className="space-y-2">
					{(["solid", "outline", "ghost", "danger"] as const).map((v) => (
						<Row key={v}>
							<span className="w-14 text-xs text-muted-foreground">{v}</span>
							<Button variant={v} size="xs">xs</Button>
							<Button variant={v} size="sm">sm</Button>
							<Button variant={v} size="md">md</Button>
						</Row>
					))}
				</div>
			</Subsection>
		</Section>
	)
}

/* ── Spinner ── */
function SpinnerSection() {
	return (
		<Section title="Spinner" description="Animated loading indicator with size variants.">
			<Row>
				<LabeledItem label="xs"><Spinner size="xs" /></LabeledItem>
				<LabeledItem label="sm"><Spinner size="sm" /></LabeledItem>
				<LabeledItem label="md"><Spinner size="md" /></LabeledItem>
				<LabeledItem label="lg"><Spinner size="lg" /></LabeledItem>
				<LabeledItem label="xl"><Spinner size="xl" /></LabeledItem>
			</Row>
		</Section>
	)
}

/* ── Input ── */
function InputSection() {
	return (
		<Section title="Input" description="Shadow-based borders with idle, hover, focus transitions and feedback states.">
			<Subsection title="Default">
				<div className="grid max-w-sm gap-3">
					<div className="space-y-1.5">
						<Label htmlFor="demo-email">Email</Label>
						<Input id="demo-email" type="email" placeholder="you@example.com" />
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="demo-pass">Password</Label>
						<Input id="demo-pass" type="password" placeholder="Enter password" />
					</div>
				</div>
			</Subsection>
			<Subsection title="Sizes">
				<div className="grid max-w-sm gap-3">
					<Input inputSize="sm" placeholder="Small input" />
					<Input inputSize="md" placeholder="Medium input (default)" />
				</div>
			</Subsection>
			<Subsection title="Feedback states">
				<div className="grid max-w-sm gap-3">
					<Input feedback="error" defaultValue="Invalid email" />
					<Input feedback="warning" defaultValue="Weak password" />
					<Input feedback="success" defaultValue="Looks good!" />
				</div>
			</Subsection>
			<Subsection title="Disabled">
				<div className="max-w-sm">
					<Input disabled placeholder="Disabled input" />
				</div>
			</Subsection>
		</Section>
	)
}

/* ── Input Group ── */
function InputGroupSection() {
	return (
		<Section title="Input Group" description="Input with addons — icons, buttons, or text.">
			<div className="grid max-w-sm gap-3">
				<InputGroup>
					<InputGroupAddon align="inline-start">
						<InputGroupText><Search className="size-4" /></InputGroupText>
					</InputGroupAddon>
					<InputGroupInput placeholder="Search secrets..." />
				</InputGroup>
				<InputGroup>
					<InputGroupInput placeholder="Enter vault key" type="password" />
					<InputGroupAddon align="inline-end">
						<InputGroupButton size="xs" variant="ghost"><Eye className="size-3.5" /></InputGroupButton>
					</InputGroupAddon>
				</InputGroup>
				<InputGroup>
					<InputGroupAddon align="inline-start">
						<InputGroupText><span>https://</span></InputGroupText>
					</InputGroupAddon>
					<InputGroupInput placeholder="api.example.com" />
				</InputGroup>
			</div>
		</Section>
	)
}

/* ── Input OTP ── */
function InputOTPSection() {
	return (
		<Section title="Input OTP" description="One-time password input for 2FA flows.">
			<InputOTP maxLength={6}>
				<InputOTPGroup>
					<InputOTPSlot index={0} />
					<InputOTPSlot index={1} />
					<InputOTPSlot index={2} />
				</InputOTPGroup>
				<InputOTPSeparator />
				<InputOTPGroup>
					<InputOTPSlot index={3} />
					<InputOTPSlot index={4} />
					<InputOTPSlot index={5} />
				</InputOTPGroup>
			</InputOTP>
		</Section>
	)
}

/* ── Textarea ── */
function TextareaSection() {
	return (
		<Section title="Textarea" description="Multi-line input with same shadow pattern as Input.">
			<div className="grid max-w-sm gap-3">
				<div className="space-y-1.5">
					<Label htmlFor="demo-textarea">Description</Label>
					<Textarea id="demo-textarea" placeholder="Enter a description..." />
				</div>
				<Textarea feedback="error" defaultValue="This value is invalid" />
			</div>
		</Section>
	)
}

/* ── Select ── */
function SelectSection() {
	return (
		<Section title="Select" description="Dropdown select with input shadow pattern on trigger.">
			<div className="flex flex-wrap gap-3">
				<Select>
					<SelectTrigger>
						<SelectValue placeholder="Select environment" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="production">Production</SelectItem>
						<SelectItem value="staging">Staging</SelectItem>
						<SelectItem value="development">Development</SelectItem>
					</SelectContent>
				</Select>
				<Select>
					<SelectTrigger size="sm">
						<SelectValue placeholder="Size sm" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="a">Option A</SelectItem>
						<SelectItem value="b">Option B</SelectItem>
					</SelectContent>
				</Select>
			</div>
		</Section>
	)
}

/* ── Checkbox ── */
function CheckboxSection() {
	return (
		<Section title="Checkbox" description="Shadow-based borders, checked state uses primary color.">
			<div className="space-y-3">
				<div className="flex items-center gap-2">
					<Checkbox id="cb1" />
					<Label htmlFor="cb1">Accept terms and conditions</Label>
				</div>
				<div className="flex items-center gap-2">
					<Checkbox id="cb2" defaultChecked />
					<Label htmlFor="cb2">Enable notifications</Label>
				</div>
				<div className="flex items-center gap-2">
					<Checkbox id="cb3" disabled />
					<Label htmlFor="cb3" className="opacity-50">Disabled</Label>
				</div>
			</div>
		</Section>
	)
}

/* ── Switch ── */
function SwitchSection() {
	return (
		<Section title="Switch" description="Toggle switch with shadow borders. Default and small sizes.">
			<div className="space-y-3">
				<div className="flex items-center gap-2">
					<Switch id="sw1" />
					<Label htmlFor="sw1">Auto-rotate keys</Label>
				</div>
				<div className="flex items-center gap-2">
					<Switch id="sw2" defaultChecked />
					<Label htmlFor="sw2">Enabled</Label>
				</div>
				<div className="flex items-center gap-2">
					<Switch id="sw3" size="sm" />
					<Label htmlFor="sw3">Small switch</Label>
				</div>
				<div className="flex items-center gap-2">
					<Switch id="sw4" disabled />
					<Label htmlFor="sw4" className="opacity-50">Disabled</Label>
				</div>
			</div>
		</Section>
	)
}

/* ── Radio Group ── */
function RadioGroupSection() {
	return (
		<Section title="Radio Group" description="Shadow-based radio with primary fill on selection.">
			<RadioGroup defaultValue="production">
				<div className="flex items-center gap-2">
					<RadioGroupItem value="production" id="rg-prod" />
					<Label htmlFor="rg-prod">Production</Label>
				</div>
				<div className="flex items-center gap-2">
					<RadioGroupItem value="staging" id="rg-stg" />
					<Label htmlFor="rg-stg">Staging</Label>
				</div>
				<div className="flex items-center gap-2">
					<RadioGroupItem value="development" id="rg-dev" />
					<Label htmlFor="rg-dev">Development</Label>
				</div>
			</RadioGroup>
		</Section>
	)
}

/* ── Toggle ── */
function ToggleSection() {
	return (
		<Section title="Toggle / Toggle Group" description="Pressable toggle with outline variant.">
			<Subsection title="Single">
				<Row>
					<Toggle aria-label="Bold"><span className="font-bold">B</span></Toggle>
					<Toggle variant="outline" aria-label="Italic"><span className="italic">I</span></Toggle>
				</Row>
			</Subsection>
			<Subsection title="Group">
				<ToggleGroup multiple>
					<ToggleGroupItem value="bold" aria-label="Bold"><span className="font-bold">B</span></ToggleGroupItem>
					<ToggleGroupItem value="italic" aria-label="Italic"><span className="italic">I</span></ToggleGroupItem>
					<ToggleGroupItem value="underline" aria-label="Underline"><span className="underline">U</span></ToggleGroupItem>
				</ToggleGroup>
			</Subsection>
		</Section>
	)
}

/* ── Slider ── */
function SliderSection() {
	return (
		<Section title="Slider" description="Range slider with primary-colored thumb and focus ring.">
			<div className="max-w-sm space-y-4">
				<Slider defaultValue={[40]} />
				<Slider defaultValue={[20, 80]} />
			</div>
		</Section>
	)
}

/* ── Badge ── */
function BadgeSection() {
	return (
		<Section title="Badge" description="Shadow-based borders with color-specific tinting.">
			<Row>
				<Badge variant="neutral">Neutral</Badge>
				<Badge variant="primary">Primary</Badge>
				<Badge variant="success">Success</Badge>
				<Badge variant="warning">Warning</Badge>
				<Badge variant="danger">Danger</Badge>
			</Row>
		</Section>
	)
}

/* ── Kbd ── */
function KbdSection() {
	return (
		<Section title="Kbd" description="Keyboard shortcut indicator.">
			<Row>
				<Kbd>⌘</Kbd>
				<Kbd>K</Kbd>
				<span className="text-xs text-muted-foreground">or</span>
				<Kbd>Ctrl</Kbd>
				<Kbd>Shift</Kbd>
				<Kbd>P</Kbd>
			</Row>
		</Section>
	)
}

/* ── Alert ── */
function AlertSection() {
	return (
		<Section title="Alert" description="Color-coded alert boxes with icon support.">
			<div className="grid max-w-lg gap-4">
				<Alert variant="info">
					<Info />
					<div>
						<AlertTitle>Info</AlertTitle>
						<AlertDescription>Your vault is locked. Unlock it to access secrets.</AlertDescription>
					</div>
				</Alert>
				<Alert variant="success">
					<CheckCircle />
					<div>
						<AlertTitle>Success</AlertTitle>
						<AlertDescription>Secret saved and encrypted client-side.</AlertDescription>
					</div>
				</Alert>
				<Alert variant="warning">
					<TriangleAlert />
					<div>
						<AlertTitle>Warning</AlertTitle>
						<AlertDescription>Your vault key is only shown once. Save it now.</AlertDescription>
					</div>
				</Alert>
				<Alert variant="danger">
					<AlertCircle />
					<div>
						<AlertTitle>Danger</AlertTitle>
						<AlertDescription>Failed to decrypt. Check your vault key.</AlertDescription>
					</div>
				</Alert>
			</div>
		</Section>
	)
}

/* ── Avatar ── */
function AvatarSection() {
	return (
		<Section title="Avatar" description="Image with fallback initials, multiple sizes.">
			<Row>
				<LabeledItem label="xs"><Avatar size="xs" alt="Ada Lovelace" /></LabeledItem>
				<LabeledItem label="sm"><Avatar size="sm" alt="Ada Lovelace" /></LabeledItem>
				<LabeledItem label="md"><Avatar size="md" alt="Ada Lovelace" /></LabeledItem>
				<LabeledItem label="lg"><Avatar size="lg" alt="Ada Lovelace" /></LabeledItem>
			</Row>
			<Row>
				<LabeledItem label="fallback"><Avatar alt="John Doe" /></LabeledItem>
				<LabeledItem label="custom"><Avatar fallback="Z" /></LabeledItem>
			</Row>
		</Section>
	)
}

/* ── Separator ── */
function SeparatorSection() {
	return (
		<Section title="Separator" description="Horizontal and vertical dividers.">
			<div className="space-y-4">
				<div>
					<p className="mb-2 text-sm">Above</p>
					<Separator />
					<p className="mt-2 text-sm">Below</p>
				</div>
				<div className="flex h-8 items-center gap-4">
					<span className="text-sm">Left</span>
					<Separator orientation="vertical" />
					<span className="text-sm">Right</span>
				</div>
			</div>
		</Section>
	)
}

/* ── Progress ── */
function ProgressSection() {
	return (
		<Section title="Progress" description="Determinate progress bar.">
			<div className="max-w-sm space-y-3">
				<Progress value={0} />
				<Progress value={33} />
				<Progress value={66} />
				<Progress value={100} />
			</div>
		</Section>
	)
}

/* ── Skeleton ── */
function SkeletonSection() {
	return (
		<Section title="Skeleton" description="Loading placeholder.">
			<div className="flex items-center gap-3">
				<Skeleton className="size-8 rounded-full" />
				<div className="space-y-2">
					<Skeleton className="h-3 w-40" />
					<Skeleton className="h-3 w-24" />
				</div>
			</div>
		</Section>
	)
}

/* ── Card ── */
function CardSection() {
	return (
		<Section title="Card" description="CardBox (outer shadow) + Card (inner content) compound pattern.">
			<div className="grid max-w-lg gap-6">
				<CardBox>
					<Card>
						<CardHeader>
							<CardTitle>Vault Setup</CardTitle>
							<CardDescription>Create a vault key to encrypt your secrets.</CardDescription>
						</CardHeader>
						<CardContent>
							<div className="space-y-3">
								<div className="space-y-1.5">
									<Label htmlFor="vault-key">Vault Key</Label>
									<Input id="vault-key" type="password" placeholder="Enter a strong passphrase" />
								</div>
							</div>
						</CardContent>
						<CardFooter>
							<Button variant="solid">Create Vault</Button>
							<Button variant="ghost">Cancel</Button>
						</CardFooter>
					</Card>
				</CardBox>
				<Subsection title="Action Card">
					<ActionCard>
						<div className="flex items-center justify-between">
							<div>
								<p className="text-sm font-medium">DATABASE_URL</p>
								<p className="text-xs text-muted-foreground">Last updated 2 hours ago</p>
							</div>
							<Button variant="outline" size="icon-sm"><Copy /></Button>
						</div>
					</ActionCard>
				</Subsection>
			</div>
		</Section>
	)
}

/* ── Table ── */
function TableSection() {
	return (
		<Section title="Table" description="Shadow-based border wrapping, minimal row borders.">
			<div className="max-w-lg">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Name</TableHead>
							<TableHead>Environment</TableHead>
							<TableHead>Updated</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						<TableRow>
							<TableCell className="font-medium">DATABASE_URL</TableCell>
							<TableCell><Badge variant="primary">production</Badge></TableCell>
							<TableCell className="text-muted-foreground">2 hours ago</TableCell>
						</TableRow>
						<TableRow>
							<TableCell className="font-medium">API_KEY</TableCell>
							<TableCell><Badge variant="warning">staging</Badge></TableCell>
							<TableCell className="text-muted-foreground">1 day ago</TableCell>
						</TableRow>
						<TableRow>
							<TableCell className="font-medium">REDIS_URL</TableCell>
							<TableCell><Badge variant="neutral">development</Badge></TableCell>
							<TableCell className="text-muted-foreground">3 days ago</TableCell>
						</TableRow>
					</TableBody>
				</Table>
			</div>
		</Section>
	)
}

/* ── Accordion ── */
function AccordionSection() {
	return (
		<Section title="Accordion" description="Collapsible content sections.">
			<div className="max-w-lg">
				<Accordion>
					<AccordionItem value="item-1">
						<AccordionTrigger>What is zero-knowledge encryption?</AccordionTrigger>
						<AccordionContent>
							Your secrets are encrypted on your device before being sent to the server. The server only stores ciphertext and can never access your plaintext data.
						</AccordionContent>
					</AccordionItem>
					<AccordionItem value="item-2">
						<AccordionTrigger>What happens if I lose my vault key?</AccordionTrigger>
						<AccordionContent>
							Without your vault key, your secrets cannot be decrypted. We recommend storing a recovery kit in a secure location.
						</AccordionContent>
					</AccordionItem>
					<AccordionItem value="item-3">
						<AccordionTrigger>Can team members access my secrets?</AccordionTrigger>
						<AccordionContent>
							Team members with appropriate permissions can access project secrets. Access is controlled through organization roles and public-key cryptography.
						</AccordionContent>
					</AccordionItem>
				</Accordion>
			</div>
		</Section>
	)
}

/* ── Tabs ── */
function TabsSection() {
	return (
		<Section title="Tabs" description="Default (pill) and line variants for environment switching.">
			<Subsection title="Default variant">
				<Tabs defaultValue="production">
					<TabsList>
						<TabsTrigger value="production">Production</TabsTrigger>
						<TabsTrigger value="staging">Staging</TabsTrigger>
						<TabsTrigger value="development">Development</TabsTrigger>
					</TabsList>
					<TabsContent value="production">
						<p className="pt-3 text-sm text-muted-foreground">Production secrets (3 items)</p>
					</TabsContent>
					<TabsContent value="staging">
						<p className="pt-3 text-sm text-muted-foreground">Staging secrets (1 item)</p>
					</TabsContent>
					<TabsContent value="development">
						<p className="pt-3 text-sm text-muted-foreground">Development secrets (5 items)</p>
					</TabsContent>
				</Tabs>
			</Subsection>
			<Subsection title="Line variant">
				<Tabs defaultValue="secrets">
					<TabsList variant="line">
						<TabsTrigger value="secrets">Secrets</TabsTrigger>
						<TabsTrigger value="tokens">Tokens</TabsTrigger>
						<TabsTrigger value="settings">Settings</TabsTrigger>
					</TabsList>
					<TabsContent value="secrets">
						<p className="pt-3 text-sm text-muted-foreground">Secrets content</p>
					</TabsContent>
				</Tabs>
			</Subsection>
		</Section>
	)
}

/* ── Dialog ── */
function DialogSection() {
	return (
		<Section title="Dialog" description="Modal overlay with card shadow pattern.">
			<Dialog>
				<DialogTrigger render={<Button variant="outline">Open Dialog</Button>} />
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Delete Secret</DialogTitle>
						<DialogDescription>
							Are you sure you want to delete this secret? This action cannot be undone.
						</DialogDescription>
					</DialogHeader>
					<DialogFooter>
						<Button variant="danger" size="sm">Delete</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</Section>
	)
}

/* ── Dropdown Menu ── */
function DropdownSection() {
	return (
		<Section title="Dropdown Menu" description="Context menu with Clerk's menu shadow pattern.">
			<DropdownMenu>
				<DropdownMenuTrigger render={<Button variant="outline" size="icon"><MoreHorizontal /></Button>} />
				<DropdownMenuContent>
					<DropdownMenuItem>
						<Copy /> Copy value
					</DropdownMenuItem>
					<DropdownMenuItem>
						<Settings /> Edit secret
					</DropdownMenuItem>
					<DropdownMenuSeparator />
					<DropdownMenuItem variant="destructive">
						<Trash2 /> Delete
					</DropdownMenuItem>
				</DropdownMenuContent>
			</DropdownMenu>
		</Section>
	)
}

/* ── Tooltip ── */
function TooltipSection() {
	return (
		<Section title="Tooltip" description="Small info tooltip on hover.">
			<TooltipProvider>
				<Tooltip>
					<TooltipTrigger render={<Button variant="outline" size="sm">Hover me</Button>} />
					<TooltipContent>
						<p>Encrypted client-side with AES-256-GCM</p>
					</TooltipContent>
				</Tooltip>
			</TooltipProvider>
		</Section>
	)
}

/* ── Gallery layout helpers ── */

function Section({ title, description, children }: { title: string; description: string; children: React.ReactNode }) {
	return (
		<section>
			<div className="mb-6 border-b border-border pb-3">
				<h2 className="text-lg font-semibold">{title}</h2>
				<p className="text-sm text-muted-foreground">{description}</p>
			</div>
			<div className="space-y-8">{children}</div>
		</section>
	)
}

function Subsection({ title, children }: { title: string; children: React.ReactNode }) {
	return (
		<div>
			<h3 className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">{title}</h3>
			{children}
		</div>
	)
}

function Row({ children }: { children: React.ReactNode }) {
	return <div className="flex flex-wrap items-center gap-3">{children}</div>
}

function LabeledItem({ label, children }: { label: string; children: React.ReactNode }) {
	return (
		<div className="flex flex-col items-center gap-2">
			{children}
			<span className="text-xs text-muted-foreground">{label}</span>
		</div>
	)
}
