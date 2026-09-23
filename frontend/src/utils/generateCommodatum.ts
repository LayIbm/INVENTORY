import { jsPDF } from 'jspdf'

/**
 * Unified input type for both laptops and equipment commodatum PDFs.
 * Fields marked optional are not present on all device types.
 */
export interface CommodatumDevice {
  /** LAPTOP | TINY PC | MONITOR | ADAPTER */
  description: string
  /** Variant/sub-model; use '-' if not applicable */
  deviceType: string
  model: string
  serial: string
  employeeName: string
  employeeTalentId: string
  /** Today's date in YYYY-MM-DD format */
  date: string
  /** Only for LAPTOP and TINY PC */
  os?: string
  chargerIncluded?: boolean
  /** Only for Desktop (TINY PC) */
  monitorIncluded?: boolean
  monitorSerial?: string
}

const CLAUSES = [
  '1. Ownership and return. The assigned device(s) and any accessories are the property of the Company or its Client. I will return them in good working order, with all software and accessories, immediately upon request, role change, or termination of employment.',
  '2. Permitted use. The device is a work tool for my exclusive use in connection with my job duties. I will not transfer, share, lend, or exchange it with other users.',
  '3. Care and responsibility. I will take reasonable care of the device. If damage or loss occurs due to negligence or failure to follow Company policies (including transport and protection requirements), I may be responsible for repair or replacement costs in accordance with applicable law and Company policy.',
  '4. Support and repairs. In case of hardware or accessory failure, I will notify my manager and open a ticket with the Help Desk. I will not attempt unauthorized repairs or modifications.',
  '5. Software and configuration. I will not install unapproved software, disable security controls, or alter configuration without authorization. I will use only licensed software as approved by the Company.',
  '6. Security and monitoring. Company security controls (e.g., encryption, endpoint protection, firewall, screen lock, password policy) must remain enabled and up to date. The Company may, where permitted by policy and law, monitor, audit, or inspect Company devices and data.',
  '7. Workstation hygiene (minimum requirements).\n    \u2022 Strong password and automatic screen lock enabled\n    \u2022 Disk encryption and endpoint protection active and updated\n    \u2022 Firewall enabled\n    \u2022 No unauthorized or pirated software\n    \u2022 Compliant desktop image and approved configurations only\n    \u2022 Company data stored only in approved locations',
  '8. Confidential information. I will handle Company and Client information in accordance with applicable confidentiality, privacy, and information security policies.',
  '9. Theft or loss (mandatory actions). I will: - File a report with the competent authorities within 24 hours (if theft). - Notify my manager and the Security/IT teams and provide the incident number within 48 hours. - Cooperate with any investigation, remote lock/wipe, and asset replacement process (subject to availability).',
  '10. Financial responsibility. If the device is lost, stolen, or damaged due to negligence or policy violations, I acknowledge that I may be responsible for repair or replacement costs, subject to applicable law and with any payroll deductions made only where lawful and with my explicit authorization.',
  '11. End of responsibility. My responsibility for the assigned device ends when I return all items and the receiving area signs the official asset checklist.',
]

function buildAccessoriesLine(device: CommodatumDevice): string {
  const parts: string[] = []
  if (device.chargerIncluded) parts.push('CHARGER AND POWER CABLE')
  return parts.length > 0 ? parts.join(', ') : 'N/A'
}

function buildPeripheralsLine(device: CommodatumDevice): string | null {
  if (!device.monitorIncluded) return null
  const parts: string[] = ['Monitor']
  if (device.monitorSerial) parts.push(`Monitor Serial: ${device.monitorSerial}`)
  return parts.join(', ')
}

/**
 * Generates and immediately downloads an IBM commodatum PDF for the given device.
 */
export function generateCommodatumPDF(device: CommodatumDevice): void {
  const doc = new jsPDF({ unit: 'pt', format: 'letter' })

  const marginL = 56
  const marginR = 56
  const pageW = doc.internal.pageSize.getWidth()
  const usableW = pageW - marginL - marginR
  let y = 56

  // ── Summary block ──────────────────────────────────────────
  doc.setFontSize(10)
  doc.setFont('helvetica', 'normal')

  const summaryLines: [string, string][] = [
    ['Description:', device.description],
    ['Type:', device.deviceType || '-'],
    ['Model:', device.model],
    ['Serial:', device.serial],
    ['Employee Number:', device.employeeTalentId || '-'],
    ['Date:', device.date],
  ]
  if (device.os) summaryLines.push(['Operating System:', device.os])

  const peripherals = buildPeripheralsLine(device)
  if (peripherals) summaryLines.push(['Included Devices/Peripherals:', peripherals])

  summaryLines.push(['Included Accessories:', buildAccessoriesLine(device)])

  for (const [label, value] of summaryLines) {
    doc.setFont('helvetica', 'bold')
    doc.text(label, marginL, y)
    doc.setFont('helvetica', 'normal')
    doc.text(value, marginL + 140, y)
    y += 14
  }

  y += 10

  // ── Title block ────────────────────────────────────────────
  doc.setFontSize(12)
  doc.setFont('helvetica', 'bold')
  doc.text('Commodatum Device Assignment', marginL, y)
  y += 16

  doc.setFontSize(10)
  const titleLines = doc.splitTextToSize(
    'LETTER AGREEMENT FOR TEMPORARY ASSIGNMENT OF ELECTRONIC EQUIPMENT AND/OR PERIPHERALS',
    usableW,
  ) as string[]
  doc.text(titleLines, marginL, y)
  y += titleLines.length * 13 + 6

  // ── Opening paragraph ──────────────────────────────────────
  doc.setFont('helvetica', 'normal')
  const upperName = device.employeeName.toUpperCase()
  const intro =
    `I, ___${upperName}___, acknowledge receipt\u2014on a temporary, loan-for-use basis\u2014of the ` +
    `electronic equipment and/or peripherals described in the technical specifications and summary above. I agree to ` +
    `the following terms and conditions:`
  const introLines = doc.splitTextToSize(intro, usableW) as string[]
  doc.text(introLines, marginL, y)
  y += introLines.length * 13 + 6

  // ── Clauses ────────────────────────────────────────────────
  for (const clause of CLAUSES) {
    const lines = doc.splitTextToSize(clause, usableW) as string[]
    const blockH = lines.length * 13 + 4
    if (y + blockH > doc.internal.pageSize.getHeight() - 80) {
      doc.addPage()
      y = 56
    }
    doc.text(lines, marginL, y)
    y += blockH
  }

  y += 8

  // ── Signature block ────────────────────────────────────────
  if (y + 70 > doc.internal.pageSize.getHeight() - 40) {
    doc.addPage()
    y = 56
  }
  doc.setFont('helvetica', 'bold')
  doc.text(`Name: ___${upperName}___`, marginL, y)
  y += 16
  doc.text('Signature: _________________________________________', marginL, y)
  y += 16
  doc.text(`Date: ${device.date}`, marginL, y)

  // ── Save ───────────────────────────────────────────────────
  const slug = device.employeeName.toLowerCase().replace(/\s+/g, '_')
  const serialSlug = device.serial.toLowerCase()
  doc.save(`commodatum_${slug}_${serialSlug}.pdf`)
}
