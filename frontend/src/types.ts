export type CategoryIcon =
  | 'food'
  | 'market'
  | 'transport'
  | 'home'
  | 'health'
  | 'leisure'
  | 'salary'
  | 'freelance'
  | 'education'
  | 'pets'
  | 'clothing'
  | 'shopping'
  | 'travel'
  | 'phone'
  | 'cafe'
  | 'kids'
  | 'car'
  | 'utilities'
  | 'subscriptions'
  | 'gift'
  | 'investment'
  | 'other'

export type Category = {
  id: number
  name: string
  description?: string
  icon: CategoryIcon | string
  created_at: string
}

export type Member = {
  id: number
  name: string
  monthly_salary: number
  created_at: string
}

export type Transaction = {
  id: number
  category_id: number
  member_id?: number | null
  wallet_id?: number | null
  type: 'income' | 'expense'
  description: string
  amount: number
  date: string
  created_at: string
}

export type MemberForecast = {
  member_id: number
  member_name: string
  planned_salary: number
  extra_income: number
  total_available: number
  bill_share: number
  savings_share: number
  variable_expense: number
  total_to_pay: number
  remaining: number
}

export type MonthlyForecast = {
  year: number
  month: number
  planned_salary: number
  extra_income: number
  total_available: number
  planned_bills: number
  planned_savings: number
  total_expense: number
  remaining: number
  days_in_month: number
  days_elapsed: number
  days_remaining: number
  projected_expense: number
  safe_daily_spend: number
  expense_pace_ratio: number
  by_member: MemberForecast[]
}

export type BillFrequency = 'daily' | 'weekdays' | 'weekly' | 'biweekly' | 'monthly' | 'yearly'
export type BillAmountMode = 'fixed' | 'interest' | 'schedule'
export type BillRecurrence = 'ongoing' | 'until'

export type BillPayment = {
  bill_id: number
  year: number
  month: number
  paid_at: string
  paid_by_member_id?: number | null
  wallet_id?: number | null
  amount?: number
}

export type Bill = {
  id: number
  name: string
  amount: number
  amount_mode: BillAmountMode
  interest_rate: number
  category_id: number
  member_ids: number[]
  wallet_id?: number | null
  due_day: number
  frequency: BillFrequency
  recurrence: BillRecurrence
  start_month: string
  end_month?: string | null
  notes?: string
  source: 'manual' | 'statement' | string
  installment_start?: number
  installment_total?: number
  created_at: string
}

export type SavingsEndKind = 'none' | 'date' | 'amount'

export type SavingsGoal = {
  id: number
  name: string
  target_amount: number
  monthly_amount: number
  saved_amount: number
  member_ids: number[]
  notes?: string
  end_kind: SavingsEndKind
  end_month?: string | null
  cdi_annual: number
  yield_annual: number
  opening_amount?: number
  created_at: string
}

export type SavingsPlan = {
  months: number
  monthly_amount: number
  cdi_annual: number
  yield_factor: number
  yield_annual: number
  target_amount: number
  per_member: number
  member_count: number
  used_default_term: boolean
}

export type SavingsMonthAmount = {
  goal_id: number
  year: number
  month: number
  amount: number
  saved_at: string
}

export type WalletKind = 'checking' | 'savings' | 'benefit' | 'company' | 'credit'

export type Wallet = {
  id: number
  name: string
  kind: WalletKind | string
  member_id?: number | null
  balance: number
  closing_day?: number | null
  due_day?: number | null
  credit_limit?: number
  invoice_balance?: number
  created_at: string
}

export type CardInvoice = {
  wallet_id: number
  year: number
  month: number
  closing_date: string
  due_date: string
  amount: number
  paid_amount: number
  outstanding: number
  paid: boolean
  source: 'calculated' | 'statement' | string
  statement_period_start?: string | null
  statement_period_end?: string | null
  statement_balance?: number | null
  updated_at?: string | null
}

export type StatementKind = 'expense' | 'income' | 'payment' | 'refund' | 'transfer' | string

export type StatementPreviewItem = {
  index: number
  date: string
  description: string
  amount: number
  kind: StatementKind
  category_id: number
  suggested_icon: string
  already_recorded: boolean
  matched_transaction_id?: number | null
  matched_bill_id?: number | null
  selected: boolean
}

export type StatementPreview = {
  issuer: string
  statement_type: 'account' | 'credit_card' | string
  period_start?: string | null
  period_end?: string | null
  balance?: number | null
  invoice_year?: number | null
  invoice_month?: number | null
  closing_date?: string | null
  due_date?: string | null
  wallet_id?: number | null
  member_id?: number | null
  new_count: number
  matched_count: number
  skipped_count: number
  items: StatementPreviewItem[]
}

export type StatementImportResult = {
  created: number
  items: Transaction[]
}
