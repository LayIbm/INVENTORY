import client from '../api/client'

export interface Invoice {
  id: string
  number: string
  client: string
  concept: string
  amount: number
  status: 'PENDIENTE' | 'PAGADA' | 'CANCELADA'
  issue_date: string
  due_date: string
  notes: string
  created_by: string
  created_at: string
  updated_at: string
}

export type CreateInvoicePayload = Omit<Invoice, 'id' | 'created_by' | 'created_at' | 'updated_at'>
export type UpdateInvoicePayload = Omit<Invoice, 'id' | 'number' | 'created_by' | 'created_at' | 'updated_at'>

export async function fetchInvoices(): Promise<Invoice[]> {
  const res = await client.get<Invoice[]>('/invoices')
  return res.data
}

export async function createInvoice(payload: CreateInvoicePayload): Promise<Invoice> {
  const res = await client.post<Invoice>('/invoices', payload)
  return res.data
}

export async function updateInvoice(id: string, payload: UpdateInvoicePayload): Promise<Invoice> {
  const res = await client.put<Invoice>(`/invoices/${id}`, payload)
  return res.data
}

export async function deleteInvoice(id: string): Promise<void> {
  await client.delete(`/invoices/${id}`)
}
