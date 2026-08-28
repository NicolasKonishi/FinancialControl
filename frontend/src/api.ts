import type {
  Bill,
  BillAmountMode,
  BillFrequency,
  BillPayment,
  BillRecurrence,
  Category,
  Member,
  MonthlyForecast,
  Transaction,
} from './types'

function resolveApiUrl() {
  const fromEnv = import.meta.env.VITE_API_URL as string | undefined
  if (fromEnv) return fromEnv.replace(/\/$/, '')
  if (typeof window === 'undefined') return 'http://localhost:8080'
  const { protocol, hostname } = window.location
  if (hostname === 'localhost' || hostname === '127.0.0.1') {
    return 'http://localhost:8080'
  }
  // Phone on the same Wi-Fi: UI at :5173, API at :8080 on this machine.
  return `${protocol}//${hostname}:8080`
}

const API_URL = resolveApiUrl()

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
  createCategory: (body: { name: string; description?: string; icon: string }) =>
    request<Category>('/categories', { method: 'POST', body: JSON.stringify(body) }),
  updateCategory: (
    id: number,
    body: { name: string; description?: string; icon: string },
  ) => request<Category>(`/categories/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteCategory: (id: number) => request<void>(`/categories/${id}`, { method: 'DELETE' }),

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
    amount_mode?: BillAmountMode
    interest_rate?: number
    category_id: number
    member_ids: number[]
    due_day: number
    frequency: BillFrequency
    recurrence: BillRecurrence
    start_month: string
    end_month?: string | null
    notes?: string
  }) => request<Bill>('/bills', { method: 'POST', body: JSON.stringify(body) }),
  updateBill: (
    id: number,
    body: {
      name: string
      amount: number
      amount_mode?: BillAmountMode
      interest_rate?: number
      category_id: number
      member_ids: number[]
      due_day: number
      frequency: BillFrequency
      recurrence: BillRecurrence
      start_month: string
      end_month?: string | null
      notes?: string
    },
  ) => request<Bill>(`/bills/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteBill: (id: number) => request<void>(`/bills/${id}`, { method: 'DELETE' }),
  listBillPayments: (year: number, month: number) =>
    request<BillPayment[]>(`/bills/payments?year=${year}&month=${month}`),
  setBillPaid: (id: number, body: { year: number; month: number; paid: boolean }) =>
    request<BillPayment | undefined>(`/bills/${id}/paid`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),

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
