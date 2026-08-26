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
