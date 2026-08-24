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

export type MonthlyForecast = {
  year: number
  month: number
  planned_salary: number
  extra_income: number
  total_available: number
  total_expense: number
  remaining: number
  days_in_month: number
  days_elapsed: number
  days_remaining: number
  projected_expense: number
  safe_daily_spend: number
  expense_pace_ratio: number
}
