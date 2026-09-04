import type { Bill, BillFrequency } from './types'

export function parseLocaleNumber(value: string): number {
  const trimmed = value.trim()
  if (!trimmed) return NaN
  if (trimmed.includes(',')) {
    return Number(trimmed.replace(/\./g, '').replace(',', '.'))
  }
  return Number(trimmed)
}

export function billMonthKey(year: number, month: number) {
  return `${year}-${String(month).padStart(2, '0')}`
}

function utcDate(year: number, month: number, day: number) {
  return new Date(Date.UTC(year, month - 1, day))
}

function daysInMonth(year: number, month: number) {
  return new Date(Date.UTC(year, month, 0)).getUTCDate()
}

function clampDay(year: number, month: number, day: number) {
  const last = daysInMonth(year, month)
  const d = Math.max(1, Math.min(day || 1, last))
  return utcDate(year, month, d)
}

function anchorDate(bill: Bill) {
  const [y, m] = bill.start_month.split('-').map(Number)
  if (!y || !m) return null
  return clampDay(y, m, bill.due_day)
}

function endDate(bill: Bill) {
  if (bill.recurrence === 'ongoing' || !bill.end_month) return null
  const [y, m] = bill.end_month.split('-').map(Number)
  if (!y || !m) return null
  return utcDate(y, m, daysInMonth(y, m))
}

function daysBetweenInclusive(start: Date, end: Date) {
  if (end < start) return 0
  return Math.floor((end.getTime() - start.getTime()) / 86_400_000) + 1
}

function countWeekdays(start: Date, end: Date) {
  let count = 0
  const d = new Date(start)
  while (d <= end) {
    const wd = d.getUTCDay()
    if (wd !== 0 && wd !== 6) count++
    d.setUTCDate(d.getUTCDate() + 1)
  }
  return count
}

function countEveryNDays(anchor: Date, rangeStart: Date, rangeEnd: Date, step: number) {
  if (step < 1 || rangeEnd < anchor) return 0
  let first = anchor
  if (rangeStart > anchor) {
    const days = Math.floor((rangeStart.getTime() - anchor.getTime()) / 86_400_000)
    if (days % step === 0) {
      first = rangeStart
    } else {
      first = new Date(anchor)
      first.setUTCDate(first.getUTCDate() + (Math.floor(days / step) + 1) * step)
    }
  }
  if (first > rangeEnd) return 0
  let count = 0
  const d = new Date(first)
  while (d <= rangeEnd) {
    count++
    d.setUTCDate(d.getUTCDate() + step)
  }
  return count
}

function normalizeFrequency(frequency: BillFrequency | undefined): BillFrequency {
  return frequency || 'monthly'
}

function normalizeAmountMode(mode: Bill['amount_mode'] | 'schedule') {
  return mode === 'schedule' ? 'interest' : mode
}

function monthsSinceStart(startMonth: string, year: number, month: number) {
  const [sy, sm] = startMonth.split('-').map(Number)
  const target = year * 12 + month
  const start = sy * 12 + sm
  if (target < start) return null
  return target - start
}

export function installmentFromDescription(value: string) {
  const match = value.match(/(?:parcela\s*)?(\d{1,3})\s*(?:\/|de)\s*(\d{1,3})\b/i)
  if (!match) return null
  const current = Number(match[1])
  const total = Number(match[2])
  if (current < 1 || total < current || total > 120) return null
  return { current, total }
}

export function addMonthsToMonthKey(value: string, delta: number) {
  const [year, month] = value.split('-').map(Number)
  const date = new Date(Date.UTC(year || 2000, (month || 1) - 1 + delta, 1))
  return billMonthKey(date.getUTCFullYear(), date.getUTCMonth() + 1)
}

function cardDay(value: number | null | undefined, fallback: number) {
  if (!value || value < 1 || value > 31) return fallback
  return value
}

/** Invoice due-month (YYYY-MM) that a purchase on `at` belongs to. */
export function cardInvoiceMonthKey(
  wallet: { closing_day?: number | null; due_day?: number | null },
  at = new Date(),
) {
  const closingDay = cardDay(wallet.closing_day, 1)
  const dueDay = cardDay(wallet.due_day, closingDay)
  let closingYear = at.getFullYear()
  let closingMonth = at.getMonth() + 1
  if (at.getDate() >= closingDay) {
    const next = addMonthsToMonthKey(billMonthKey(closingYear, closingMonth), 1)
    const [year, month] = next.split('-').map(Number)
    closingYear = year
    closingMonth = month
  }
  if (dueDay <= closingDay) {
    return addMonthsToMonthKey(billMonthKey(closingYear, closingMonth), 1)
  }
  return billMonthKey(closingYear, closingMonth)
}

export function stripInstallmentLabel(value: string) {
  const cleaned = value
    .replace(/\s*(?:parcela\s*)?\d{1,3}\s*(?:\/|de)\s*\d{1,3}\b/gi, '')
    .replace(/\s+/g, ' ')
    .trim()
  return cleaned || value.trim()
}

export function normalizeBillName(value: string) {
  return stripInstallmentLabel(value)
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, ' ')
    .trim()
}

export function namesOverlap(a: string, b: string) {
  const left = normalizeBillName(a)
  const right = normalizeBillName(b)
  if (!left || !right) return false
  if (left === right) return true
  if (left.includes(right) || right.includes(left)) return true
  const aTokens = left.split(/\s+/).filter(Boolean)
  const bTokens = right.split(/\s+/).filter(Boolean)
  if (aTokens.length === 0 || bTokens.length === 0) return false
  const bSet = new Set(bTokens)
  const overlap = aTokens.filter((token) => bSet.has(token)).length
  const smaller = Math.min(aTokens.length, bTokens.length)
  return overlap > 0 && overlap * 2 >= smaller
}

export function cardBillMatchesTx(bill: Bill, tx: { description: string; amount: number; wallet_id?: number | null }) {
  if (bill.wallet_id && tx.wallet_id && bill.wallet_id !== tx.wallet_id) return false
  if (Math.abs(bill.amount - tx.amount) > 0.009) return false
  return namesOverlap(bill.name, tx.description)
}

export function transactionInStatementPeriod(
  tx: { date: string },
  invoice: { statement_period_start?: string | null; statement_period_end?: string | null },
) {
  const day = tx.date.slice(0, 10)
  const start = invoice.statement_period_start?.slice(0, 10)
  const end = invoice.statement_period_end?.slice(0, 10)
  if (!start || !end) return false
  return day >= start && day < end
}

export function billInstallmentPosition(bill: Bill, year: number, month: number) {
  const first = bill.installment_start ?? 0
  const total = bill.installment_total ?? 0
  const elapsed = monthsSinceStart(bill.start_month, year, month)
  if (first < 1 || total < first || elapsed == null) return null
  const current = first + elapsed
  if (current > total) return null
  return { current, total, remaining: total - current }
}

export function billOccurrencesInMonth(bill: Bill, year: number, month: number) {
  const anchor = anchorDate(bill)
  if (!anchor) return 0

  const monthStart = utcDate(year, month, 1)
  const monthEnd = utcDate(year, month, daysInMonth(year, month))

  let rangeStart = monthStart
  if (anchor > rangeStart) rangeStart = anchor

  let rangeEnd = monthEnd
  const end = endDate(bill)
  if (end && end < rangeEnd) rangeEnd = end

  if (rangeStart > rangeEnd) return 0

  switch (normalizeFrequency(bill.frequency)) {
    case 'daily':
      return daysBetweenInclusive(rangeStart, rangeEnd)
    case 'weekdays':
      return countWeekdays(rangeStart, rangeEnd)
    case 'weekly':
      return countEveryNDays(anchor, rangeStart, rangeEnd, 7)
    case 'biweekly':
      return countEveryNDays(anchor, rangeStart, rangeEnd, 14)
    case 'yearly': {
      if (month !== anchor.getUTCMonth() + 1) return 0
      const due = clampDay(year, month, bill.due_day)
      if (due < rangeStart || due > rangeEnd) return 0
      return 1
    }
    default: {
      const due = clampDay(year, month, bill.due_day)
      if (due < rangeStart || due > rangeEnd) return 0
      return 1
    }
  }
}

export function billActiveInMonth(bill: Bill, year: number, month: number) {
  return billOccurrencesInMonth(bill, year, month) > 0
}

export function billShareForMember(bill: Bill, memberId: number, year: number, month: number) {
  const ids = bill.member_ids ?? []
  if (ids.length === 0 || !ids.includes(memberId)) return 0
  return billChargeInMonth(bill, year, month) / ids.length
}

export function billChargeInMonth(bill: Bill, year: number, month: number) {
  const occurrences = billOccurrencesInMonth(bill, year, month)
  if (occurrences === 0) return 0
  if (normalizeAmountMode(bill.amount_mode) === 'interest') {
    const n = monthsSinceStart(bill.start_month, year, month)
    if (n === null || bill.amount <= 0) return 0
    if (!bill.interest_rate || bill.interest_rate <= 0) return bill.amount
    const factor = (1 + bill.interest_rate / 100) ** n
    return Math.round(bill.amount * factor * 100) / 100
  }
  return bill.amount * occurrences
}
