import client from '../api/client'

export interface User {
  id: string
  name: string
  username: string
  role: 'viewer' | 'manager' | 'admin'
  active: boolean
  must_change_password: boolean
  created_at: string
  updated_at: string
}

export interface CreateUserPayload {
  name: string
  username: string
  password: string
  role: string
}

export interface UpdateUserPayload {
  name: string
  username: string
  role: string
  active: boolean
}

export async function fetchUsers(): Promise<User[]> {
  const res = await client.get<User[]>('/users')
  return res.data
}

export async function createUser(payload: CreateUserPayload): Promise<User> {
  const res = await client.post<User>('/users', payload)
  return res.data
}

export async function updateUser(id: string, payload: UpdateUserPayload): Promise<User> {
  const res = await client.put<User>(`/users/${id}`, payload)
  return res.data
}

export async function toggleUserActive(id: string): Promise<User> {
  const res = await client.patch<User>(`/users/${id}/toggle-active`)
  return res.data
}

export async function deleteUser(id: string): Promise<void> {
  await client.delete(`/users/${id}`)
}
