import { useState, useEffect } from "react"
import { Search } from "lucide-react"
import { Input } from "#/components/ui/input"

interface SearchInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange" | "value"> {
    value?: string
    onChange?: (value: string) => void
    debounceMs?: number
}

export function SearchInput({ value, onChange, debounceMs = 300, className, ...props }: SearchInputProps) {
    const [localValue, setLocalValue] = useState(value ?? "")

    useEffect(() => {
        setLocalValue(value ?? "")
    }, [value])

    useEffect(() => {
        if (!onChange) return
        const timer = setTimeout(() => {
            if (localValue !== (value ?? "")) {
                onChange(localValue)
            }
        }, debounceMs)
        return () => clearTimeout(timer)
    }, [localValue, onChange, value, debounceMs])

    return (
        <div className="relative max-w-xs flex-1">
            <Search className="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
                type="search"
                value={onChange ? localValue : value}
                onChange={(e) => {
                    if (onChange) {
                        setLocalValue(e.target.value)
                    }
                }}
                className={`pl-8 ${className || ""}`}
                inputSize="sm"
                {...props}
            />
        </div>
    )
}
