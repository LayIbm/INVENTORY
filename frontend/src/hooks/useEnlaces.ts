import client from '../api/client'

export interface Enlace {
  id: string
  type: 'Primary' | 'Secondary'
  company: string
  public_ip: string
  ip: string
  velocity: string
  end_date_contract: string
  months_of_contract: string
  additional_service: string
  contract_number: string
  client_number: string
  identifier_link: string
  name_contact: string
  phone_contact: string
  email_contact: string
  support_phone: string
  support_clave: string
  current_po: string
  comments: string
  created_by: string
  created_at: string
  updated_at: string
}

export type CreateEnlacePayload = Omit<Enlace, 'id' | 'created_by' | 'created_at' | 'updated_at'>
export type UpdateEnlacePayload = Omit<Enlace, 'id' | 'created_by' | 'created_at' | 'updated_at'>

export async function fetchEnlaces(): Promise<Enlace[]> {
  const res = await client.get<Enlace[]>('/enlaces')
  return res.data
}

export async function createEnlace(payload: CreateEnlacePayload): Promise<Enlace> {
  const res = await client.post<Enlace>('/enlaces', payload)
  return res.data
}

export async function updateEnlace(id: string, payload: UpdateEnlacePayload): Promise<Enlace> {
  const res = await client.put<Enlace>(`/enlaces/${id}`, payload)
  return res.data
}

export async function deleteEnlace(id: string): Promise<void> {
  await client.delete(`/enlaces/${id}`)
}
