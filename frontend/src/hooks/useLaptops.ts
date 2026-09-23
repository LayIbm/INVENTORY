import client from '../api/client'

export interface HistoryEntry {
  date: string
  type: string
  tone: string
  employee: string
  notes: string
}

export interface Laptop {
  id: string
  serial: string
  model: string
  variant: string
  brand: string
  condition: string
  availability: 'DISPONIBLE' | 'ASIGNADA' | 'NO_DISPONIBLE' | 'SCRAP'
  prep: 'LISTA' | 'NECESITA_PREP' | 'NO_FUNCIONAL'
  comodato: 'FIRMADO' | 'PENDIENTE' | 'N/A'
  powers_on: string
  os: string
  win11_ready: boolean
  bios_password: string
  wifi: string
  bluetooth: string
  charger_included: boolean
  last_format_date: string
  employee_name: string | null
  employee_email: string
  employee_talent_id: string
  employee_manager_email: string
  lcd_ok: string
  expected_return_date: string
  owner: 'IBM' | 'USAA'
  hostname: string
  geography: string
  last_bios_update: string
  bios_details: string
  bluetooth_disabled_bios: boolean
  epd_status: string
  ipv6: string
  usage: string
  notes: string
  history: HistoryEntry[]
  created_at: string
  updated_at: string
}

export type CreateLaptopPayload = Omit<Laptop, 'id' | 'history' | 'created_at' | 'updated_at'>
export type UpdateLaptopPayload = Omit<Laptop, 'id' | 'serial' | 'history' | 'created_at' | 'updated_at'>

export async function fetchLaptops(): Promise<Laptop[]> {
  const res = await client.get<Laptop[]>('/laptops')
  return res.data
}

export async function createLaptop(payload: CreateLaptopPayload): Promise<Laptop> {
  const res = await client.post<Laptop>('/laptops', payload)
  return res.data
}

export async function updateLaptop(id: string, payload: UpdateLaptopPayload): Promise<Laptop> {
  const res = await client.put<Laptop>(`/laptops/${id}`, payload)
  return res.data
}

export async function deleteLaptop(id: string): Promise<void> {
  await client.delete(`/laptops/${id}`)
}

export async function exportLaptopsExcel(): Promise<Blob> {
  const res = await client.get('/export/excel', { responseType: 'blob' })
  return res.data as Blob
}

export async function downloadLaptopsTemplate(): Promise<Blob> {
  const res = await client.get('/export/template', { responseType: 'blob' })
  return res.data as Blob
}

