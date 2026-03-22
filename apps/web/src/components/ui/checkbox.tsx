"use client"

import { Checkbox as CheckboxPrimitive } from "@base-ui/react/checkbox"

import { cn } from "#/lib/utils"
import { CheckIcon } from "lucide-react"

function Checkbox({ className, ...props }: CheckboxPrimitive.Root.Props) {
  return (
    <CheckboxPrimitive.Root
      data-slot="checkbox"
      className={cn(
        "peer relative flex size-4 shrink-0 items-center justify-center rounded-[4px] border-0 shadow-[0px_0px_0px_1px_var(--border-alpha-150),var(--shadow-input)] hover:shadow-[0px_0px_0px_1px_var(--border-alpha-300),var(--shadow-input)] transition-shadow outline-none group-has-disabled/field:opacity-50 after:absolute after:-inset-x-3 after:-inset-y-2 focus-visible:shadow-[0px_0px_0px_1px_var(--border-alpha-300),var(--shadow-input),var(--shadow-focus-ring)] disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:shadow-[0px_0px_0px_1px_var(--danger-alpha-400)] dark:bg-input/30 data-checked:bg-primary data-checked:text-primary-foreground data-checked:shadow-[0px_0px_0px_1px_var(--primary)] dark:data-checked:bg-primary",
        className
      )}
      {...props}
    >
      <CheckboxPrimitive.Indicator
        data-slot="checkbox-indicator"
        className="grid place-content-center text-current transition-none [&>svg]:size-3.5"
      >
        <CheckIcon
        />
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  )
}

export { Checkbox }
