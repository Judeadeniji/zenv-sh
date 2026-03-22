import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "#/lib/utils"

const avatarVariants = cva(
	"relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted",
	{
		variants: {
			size: {
				xs: "size-6 text-[10px]",
				sm: "size-8 text-xs",
				md: "size-10 text-sm",
				lg: "size-12 text-base",
			},
		},
		defaultVariants: {
			size: "md",
		},
	},
)

interface AvatarProps extends React.HTMLAttributes<HTMLSpanElement>, VariantProps<typeof avatarVariants> {
	src?: string | null
	alt?: string
	fallback?: string
}

function Avatar({ className, size, src, alt, fallback, ...props }: AvatarProps) {
	const [imgError, setImgError] = React.useState(false)
	const showImage = src && !imgError

	const initials = React.useMemo(() => {
		if (fallback) return fallback
		if (!alt) return "?"
		return alt
			.split(" ")
			.map((w) => w[0])
			.slice(0, 2)
			.join("")
			.toUpperCase()
	}, [alt, fallback])

	return (
		<span className={cn(avatarVariants({ size, className }))} {...props}>
			{showImage ? (
				<img
					src={src}
					alt={alt || ""}
					className="aspect-square size-full object-cover"
					onError={() => setImgError(true)}
				/>
			) : (
				<span className="font-medium text-muted-foreground">{initials}</span>
			)}
		</span>
	)
}

export { Avatar, avatarVariants }
