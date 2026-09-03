import type {
  Bill,
  BillAmountMode,
  BillFrequency,
  BillPayment,
  BillRecurrence,
  Category,
  CardInvoice,
  Member,
  MonthlyForecast,
  SavingsGoal,
  SavingsMonthAmount,
  SavingsPlan,
  StatementImportResult,
  StatementPreview,
  Transaction,
  Wallet,
  WalletKind,
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
    wallet_id?: number | null
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
      wallet_id?: number | null
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
  setBillPaid: (
    id: number,
    body: {
      year: number
      month: number
      paid: boolean
      paid_by_member_id?: number | null
      wallet_id?: number | null
    },
  ) =>
    request<BillPayment | undefined>(`/bills/${id}/paid`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),

  listTransactions: () => request<Transaction[]>('/transactions'),
  createTransaction: (body: {
    category_id: number
    member_id?: number | null
    wallet_id?: number | null
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
      wallet_id?: number | null
      type: 'income' | 'expense'
      description: string
      amount: number
      date: string
    },
  ) => request<Transaction>(`/transactions/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteTransaction: (id: number) => request<void>(`/transactions/${id}`, { method: 'DELETE' }),

  monthlyForecast: (year: number, month: number) =>
    request<MonthlyForecast>(`/forecast/monthly?year=${year}&month=${month}`),

  listSavingsGoals: () => request<SavingsGoal[]>('/savings'),
  createSavingsGoal: (body: {
    name: string
    target_amount: number
    member_ids: number[]
    notes?: string
    end_kind: 'none' | 'date' | 'amount'
    end_month?: string | null
  }) => request<SavingsGoal>('/savings', { method: 'POST', body: JSON.stringify(body) }),
  updateSavingsGoal: (
    id: number,
    body: {
      name: string
      target_amount: number
      member_ids: number[]
      notes?: string
      end_kind: 'none' | 'date' | 'amount'
      end_month?: string | null
    },
  ) => request<SavingsGoal>(`/savings/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteSavingsGoal: (id: number) => request<void>(`/savings/${id}`, { method: 'DELETE' }),
  planSavings: (query: {
    end_kind: 'none' | 'date' | 'amount'
    target: number
    end_month?: string
    members?: number
    saved?: number
  }) => {
    const params = new URLSearchParams({
      end_kind: query.end_kind,
      target: String(query.target),
    })
    if (query.end_month) params.set('end_month', query.end_month)
    if (query.members) params.set('members', String(query.members))
    if (query.saved) params.set('saved', String(query.saved))
    return request<SavingsPlan>(`/savings/plan?${params}`)
  },
  listSavingsMonths: (year: number, month: number) =>
    request<SavingsMonthAmount[]>(`/savings/months?year=${year}&month=${month}`),
  setSavingsMonth: (id: number, body: { year: number; month: number; amount: number }) =>
    request<SavingsMonthAmount | undefined>(`/savings/${id}/month`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  adjustSavings: (id: number, body: { amount: number; wallet_id?: number | null }) =>
    request<SavingsGoal>(`/savings/${id}/adjust`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),

  listWallets: () => request<Wallet[]>('/wallets'),
  createWallet: (body: {
    name: string
    kind: WalletKind | string
    member_id?: number | null
    balance: number
    closing_day?: number | null
    due_day?: number | null
    credit_limit?: number
    invoice_balance?: number
  }) => request<Wallet>('/wallets', { method: 'POST', body: JSON.stringify(body) }),
  updateWallet: (
    id: number,
    body: {
      name: string
      kind: WalletKind | string
      member_id?: number | null
      balance: number
      closing_day?: number | null
      due_day?: number | null
      credit_limit?: number
      invoice_balance?: number
    },
  ) => request<Wallet>(`/wallets/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteWallet: (id: number) => request<void>(`/wallets/${id}`, { method: 'DELETE' }),
  payWalletInvoice: (id: number, body: { amount: number; from_wallet_id: number; year: number; month: number }) =>
    request<Wallet>(`/wallets/${id}/pay-invoice`, { method: 'POST', body: JSON.stringify(body) }),

  listCardInvoices: (year: number, month: number) =>
    request<CardInvoice[]>(`/card-invoices?year=${year}&month=${month}`),

  previewStatement: async (file: File, body: {
    wallet_id?: number | null
    member_id?: number | null
    year: number
    month: number
  }) => {
    const form = new FormData()
    form.append('file', file)
    if (body.wallet_id) form.append('wallet_id', String(body.wallet_id))
    if (body.member_id) form.append('member_id', String(body.member_id))
    form.append('year', String(body.year))
    form.append('month', String(body.month))
    const response = await fetch(`${API_URL}/statements/preview`, {
      method: 'POST',
      body: form,
    })
    if (!response.ok) {
      const message = await response.text()
      throw new Error(message.trim() || `HTTP ${response.status}`)
    }
    return response.json() as Promise<StatementPreview>
  },

  importStatement: (body: {
    wallet_id?: number | null
    member_id?: number | null
    apply_to_invoice?: boolean
    statement_type?: string
    invoice_year?: number | null
    invoice_month?: number | null
    statement_balance?: number | null
    period_start?: string | null
    period_end?: string | null
    items: Array<{
      date: string
      description: string
      amount: number
      type: 'income' | 'expense'
      category_id: number
    }>
  }) => request<StatementImportResult>('/statements/import', { method: 'POST', body: JSON.stringify(body) }),
}
