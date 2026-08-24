import type { Bill, Category, Member, MonthlyForecast, Transaction } from './types'

const API_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  if (!response.ok) {
    const message = await response.text()
    throw new Error(message.trim() || `HTTP ${response.status}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

export const api = {
  listCategories: () => request<Category[]>('/categories'),

  listMembers: () => request<Member[]>('/members'),
  createMember: (body: { name: string; monthly_salary: number }) =>
    request<Member>('/members', { method: 'POST', body: JSON.stringify(body) }),
  updateMember: (id: number, body: { name: string; monthly_salary: number }) =>
    request<Member>(`/members/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteMember: (id: number) => request<void>(`/members/${id}`, { method: 'DELETE' }),

  listBills: () => request<Bill[]>('/bills'),
  createBill: (body: {
    name: string
    amount: number
    category_id: number
    member_ids: number[]
    due_day: number
    recurrence: 'ongoing' | 'until'
    start_month: string
    end_month?: string | null
    notes?: string
  }) => request<Bill>('/bills', { method: 'POST', body: JSON.stringify(body) }),
  updateBill: (
    id: number,
    body: {
      name: string
      amount: number
      category_id: number
      member_ids: number[]
      due_day: number
      recurrence: 'ongoing' | 'until'
      start_month: string
      end_month?: string | null
      notes?: string
    },
  ) => request<Bill>(`/bills/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBill: (id: number) => request<void>(`/bills/${id}`, { method: 'DELETE' }),

  listTransactions: () => request<Transaction[]>('/transactions'),
  createTransaction: (body: {
    category_id: number
    member_id?: number | null
    type: 'income' | 'expense'
    description: string
    amount: number
    date: string
  }) => request<Transaction>('/transactions', { method: 'POST', body: JSON.stringify(body) }),
  updateTransaction: (
    id: number,
    body: {
      category_id: number
      member_id?: number | null
      type: 'income' | 'expense'
      description: string
      amount: number
      date: string
    },
  ) => request<Transaction>(`/transactions/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteTransaction: (id: number) => request<void>(`/transactions/${id}`, { method: 'DELETE' }),

  monthlyForecast: (year: number, month: number) =>
    request<MonthlyForecast>(`/forecast/monthly?year=${year}&month=${month}`),
}
