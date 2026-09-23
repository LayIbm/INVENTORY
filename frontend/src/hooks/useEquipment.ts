import client from '../api/client'

export interface HistoryEntry {
  date: string
  type: string
  tone: string
  employee: string
  notes: string
}

export interface Equipment {
  id: string
  serial: string
  device_type: 'Desktop' | 'Monitor' | 'Adapter' | 'Mouse' | 'Keyboard' | 'Headset' | 'Cable' | 'Switch' | 'Router' | 'Firewall' | 'AccessPoint' | 'License' | 'OtherNetwork'
  brand: string
  model: string
  variant: string
  condition: string
  availability: 'DISPONIBLE' | 'ASIGNADA' | 'NO_DISPONIBLE' | 'SCRAP'
  assignability: 'LISTA' | 'NECESITA_PREP' | 'NO_FUNCIONAL'
  comodato: 'FIRMADO' | 'PENDIENTE' | 'N/A'
  powers_on: string
  os: string
  bios_password: string
  wifi: string
  bluetooth: string
  charger_included: boolean
  last_format_date: string
  employee_name: string | null
  employee_email: string
  employee_talent_id: string
  transaction_date: string
  expected_return_date: string
  monitor_included: boolean
  monitor_serial: string
  owner: 'IBM' | 'USAA'
  hostname: string
  geography: string
  notes: string
  history: HistoryEntry[]
  created_at: string
  updated_at: string
  // Network fields (migration 006)
  product_id: string
  net_type: string
  end_of_sale: string | null
  end_of_life: string | null
  end_contract_support: string | null
  device_cost: string | null
  contract_cost: string | null
  room: string
  eol_status: string
  contract_status: string
  device_company: string
  contact_name: string
  contact_phone: string
  contact_email: string
  ibm_network_email_support: string
  ibm_local_email_support: string
}

export type CreateEquipmentPayload = Omit<Equipment, 'id' | 'history' | 'created_at' | 'updated_at'>
export type UpdateEquipmentPayload = Omit<Equipment, 'id' | 'serial' | 'history' | 'created_at' | 'updated_at'>

export async function fetchEquipment(): Promise<Equipment[]> {
  const res = await client.get<Equipment[]>('/equipment')
  return res.data
}

export async function createEquipment(payload: CreateEquipmentPayload): Promise<Equipment> {
  const res = await client.post<Equipment>('/equipment', payload)
  return res.data
}

export async function updateEquipment(id: string, payload: UpdateEquipmentPayload): Promise<Equipment> {
  const res = await client.put<Equipment>(`/equipment/${id}`, payload)
  return res.data
}

export async function deleteEquipment(id: string): Promise<void> {
  await client.delete(`/equipment/${id}`)
}

export async function exportEquipmentExcel(): Promise<Blob> {
  const res = await client.get('/export/excel', { responseType: 'blob' })
  return res.data as Blob
}

export async function downloadEquipmentTemplate(): Promise<Blob> {
  const res = await client.get('/export/template', { responseType: 'blob' })
  return res.data as Blob
}
