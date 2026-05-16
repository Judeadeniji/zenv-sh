import { FileIcon } from "@untitledui/file-icons"
import { mimeToFileIconType, resolveSecretMime, type SecretMimeRow } from "#/lib/secret-mime"

type Props = Partial<SecretMimeRow> & {
	/** From lazy `secretPayloadQuery` MIME batch (`resolveSecretMimeBatchServerFn`). */
	resolvedMime?: string
	/** When set, skips resolution and only picks an icon for this MIME string. */
	forcedMime?: string
	variant?: "default" | "gray" | "solid"
	className?: string
	size?: number
}

export function SecretMimeIcon({
	name = "",
	value = "",
	kind = "text",
	metadata,
	resolvedMime,
	forcedMime,
	variant = "default",
	className,
	size = 16,
}: Props) {
	const mime = forcedMime
		? forcedMime.split(";")[0].trim().toLowerCase()
		: resolveSecretMime({ name, value, kind, metadata, resolvedMime })
	const iconType = mimeToFileIconType(mime)
	return (
		<FileIcon
			type={iconType}
			variant={variant}
			size={size}
			className={className}
			aria-hidden
			title={mime}
		/>
	)
}
