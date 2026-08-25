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

export type Bill = {
  id: number
  name: string
  amount: number
  amount_mode: BillAmountMode
  interest_rate: number
  category_id: number
  member_ids: number[]
  due_day: number
  frequency: BillFrequency
  recurrence: BillRecurrence
  start_month: string
  end_month?: string | null
  notes?: string
  created_at: string
}
