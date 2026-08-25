import type { BillFrequency } from './types'

export function pad2(n: number) {
  return String(n).padStart(2, '0')
}

export function billAnchorDate(startMonth: string, dueDay: number) {
  const day = Math.min(Math.max(dueDay || 1, 1), 31)
  return `${startMonth}-${pad2(day)}`
}

export function weekdayLong(dateISO: string) {
  const [y, m, d] = dateISO.split('-').map(Number)
  if (!y || !m || !d) return ''
  return new Intl.DateTimeFormat('pt-BR', { weekday: 'long', timeZone: 'UTC' }).format(
    new Date(Date.UTC(y, m - 1, d)),
  )
}

export function monthDayLong(dateISO: string) {
  const [y, m, d] = dateISO.split('-').map(Number)
  if (!y || !m || !d) return ''
  return new Intl.DateTimeFormat('pt-BR', {
    day: 'numeric',
    month: 'long',
    timeZone: 'UTC',
  }).format(new Date(Date.UTC(y, m - 1, d)))
}

export function repeatSummary(frequency: BillFrequency, startMonth: string, dueDay: number) {
  const anchor = billAnchorDate(startMonth, dueDay)
  const day = dueDay || 1
  switch (frequency) {
    case 'daily':
      return 'Todo dia'
    case 'weekdays':
      return 'Todo dia útil (Seg – Sex)'
    case 'weekly':
      return `Toda semana ${weekdayLong(anchor)}`
    case 'biweekly':
      return `A cada 2 semanas · ${weekdayLong(anchor)}`
    case 'yearly':
      return `Todo ano em ${monthDayLong(anchor)}`
    default:
      return `Todo mês no dia ${day}`
  }
}

export function monthYearShort(year: number, month: number) {
  return new Intl.DateTimeFormat('pt-BR', {
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  }).format(new Date(Date.UTC(year, month - 1, 1)))
}

export type CalendarCell = {
  year: number
  month: number
  day: number
  outside: boolean
}

export function buildCalendarGrid(viewYear: number, viewMonth: number): CalendarCell[] {
  const firstWeekday = new Date(Date.UTC(viewYear, viewMonth - 1, 1)).getUTCDay()
  let y = viewYear
  let m = viewMonth
  let d = 1 - firstWeekday

  const cells: CalendarCell[] = []
  for (let i = 0; i < 42; i++) {
    const date = new Date(Date.UTC(y, m - 1, d))
    const cy = date.getUTCFullYear()
    const cm = date.getUTCMonth() + 1
    const cd = date.getUTCDate()
    cells.push({
      year: cy,
      month: cm,
      day: cd,
      outside: cm !== viewMonth,
    })
    d++
  }
  return cells
}

export function datePartsToISO(year: number, month: number, day: number) {
  return `${year}-${pad2(month)}-${pad2(day)}`
}

export function parseAnchor(startMonth: string, dueDay: number) {
  const [y, m] = startMonth.split('-').map(Number)
  const day = Math.min(Math.max(dueDay || 1, 1), 31)
  return { year: y, month: m, day }
}

export function parseMonthKey(key: string) {
  const [y, m] = key.split('-').map(Number)
  const now = new Date()
  return {
    year: y || now.getFullYear(),
    month: m || now.getMonth() + 1,
  }
}

export function monthKey(year: number, month: number) {
  return `${year}-${pad2(month)}`
}

export function monthKeyLabel(key: string) {
  if (!key) return 'Escolher mês'
  const { year, month } = parseMonthKey(key)
  return monthYearShort(year, month)
}

export function monthShort(month: number) {
  return new Intl.DateTimeFormat('pt-BR', { month: 'short', timeZone: 'UTC' }).format(
    new Date(Date.UTC(2026, month - 1, 1)),
  )
}
