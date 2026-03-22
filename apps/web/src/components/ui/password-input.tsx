import * as React from "react"
import { cn } from "#/lib/utils"
import { InputGroup, InputGroupInput, InputGroupAddon, InputGroupButton } from "./input-group"
import { Eye, EyeOff } from "lucide-react"
import type { Feedback } from "./input"

interface PasswordInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> {
	feedback?: Feedback
}

const PasswordInput = React.forwardRef<HTMLInputElement, PasswordInputProps>(
	({ className, feedback, ...props }, ref) => {
		const [visible, setVisible] = React.useState(false)

		return (
			<InputGroup className={cn(feedback === "error" && "shadow-[0px_0px_0px_1px_var(--danger-alpha-400),0px_0px_1px_0px_var(--danger-alpha-200)]", className)}>
				<InputGroupInput
					ref={ref}
					type={visible ? "text" : "password"}
					{...props}
				/>
				<InputGroupAddon align="inline-end">
					<InputGroupButton
						size="icon-xs"
						variant="ghost"
						onClick={() => setVisible((v) => !v)}
						aria-label={visible ? "Hide password" : "Show password"}
					>
						{visible ? <EyeOff /> : <Eye />}
					</InputGroupButton>
				</InputGroupAddon>
			</InputGroup>
		)
	},
)
PasswordInput.displayName = "PasswordInput"

export { PasswordInput }
