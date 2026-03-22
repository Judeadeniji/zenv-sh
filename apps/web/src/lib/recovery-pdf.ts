import { jsPDF } from "jspdf"

/**
 * Generate a Recovery Kit PDF with the user's mnemonic words.
 * Returns a Blob that can be downloaded.
 */
export function generateRecoveryKitPDF(email: string, mnemonic: string, date: string): Blob {
	const doc = new jsPDF({ unit: "mm", format: "a4" })
	const words = mnemonic.split(" ")
	const pageWidth = doc.internal.pageSize.getWidth()
	const margin = 20

	// Header
	doc.setFontSize(22)
	doc.setFont("helvetica", "bold")
	doc.text("zEnv Recovery Kit", margin, 30)

	doc.setFontSize(10)
	doc.setFont("helvetica", "normal")
	doc.setTextColor(100)
	doc.text(`Generated for ${email} on ${date}`, margin, 38)

	// Warning box
	doc.setDrawColor(239, 68, 68)
	doc.setFillColor(254, 242, 242)
	doc.roundedRect(margin, 46, pageWidth - margin * 2, 24, 2, 2, "FD")

	doc.setFontSize(10)
	doc.setFont("helvetica", "bold")
	doc.setTextColor(185, 28, 28)
	doc.text("KEEP THIS DOCUMENT SAFE", margin + 4, 54)

	doc.setFont("helvetica", "normal")
	doc.setFontSize(9)
	doc.setTextColor(127, 29, 29)
	doc.text(
		"This recovery kit is the ONLY way to regain access to your vault if you forget your Vault Key.",
		margin + 4,
		60,
	)
	doc.text("Store it in a secure location. Anyone with these words can access your secrets.", margin + 4, 65)

	// Recovery words grid
	doc.setTextColor(0)
	doc.setFontSize(12)
	doc.setFont("helvetica", "bold")
	doc.text("Recovery Words", margin, 84)

	const colWidth = (pageWidth - margin * 2) / 3
	const startY = 92

	doc.setFontSize(11)
	words.forEach((word, i) => {
		const col = i % 3
		const row = Math.floor(i / 3)
		const x = margin + col * colWidth
		const y = startY + row * 10

		doc.setFont("helvetica", "normal")
		doc.setTextColor(140)
		doc.text(`${(i + 1).toString().padStart(2, "0")}`, x, y)

		doc.setFont("helvetica", "bold")
		doc.setTextColor(0)
		doc.text(word, x + 10, y)
	})

	// Footer instructions
	const footerY = startY + Math.ceil(words.length / 3) * 10 + 10
	doc.setFontSize(9)
	doc.setFont("helvetica", "normal")
	doc.setTextColor(100)

	const instructions = [
		"1. Print this document or save it to an encrypted drive.",
		"2. Do NOT store it alongside your Vault Key.",
		"3. Do NOT share it with anyone you do not fully trust.",
		"4. If this document is compromised, regenerate your Recovery Kit immediately.",
	]
	instructions.forEach((line, i) => {
		doc.text(line, margin, footerY + i * 6)
	})

	return doc.output("blob")
}
