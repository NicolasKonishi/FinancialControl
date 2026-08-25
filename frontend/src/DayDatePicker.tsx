import { useEffect, useMemo, useState } from 'react'
import {
  buildCalendarGrid,
  datePartsToISO,
  monthYearShort,
} from './billSchedule'

type Props = {
  value: string
  onChange: (isoDate: string) => void
}

const WEEKDAYS = ['D', 'S', 'T', 'Q', 'Q', 'S', 'S']

function parseISODate(value: string) {
  const [y, m, d] = value.split('-').map(Number)
  const now = new Date()
  return {
    year: y || now.getFullYear(),
    month: m || now.getMonth() + 1,
    day: d || now.getDate(),
  }
}

function todayParts() {
  const now = new Date()
  return { year: now.getFullYear(), month: now.getMonth() + 1, day: now.getDate() }
}

export function DayDatePicker({ value, onChange }: Props) {
  const selected = parseISODate(value)
  const selectedISO = datePartsToISO(selected.year, selected.month, selected.day)
  const today = todayParts()
  const todayISO = datePartsToISO(today.year, today.month, today.day)

  const [viewYear, setViewYear] = useState(selected.year)
  const [viewMonth, setViewMonth] = useState(selected.month)

  useEffect(() => {
    setViewYear(selected.year)
    setViewMonth(selected.month)
  }, [selected.year, selected.month])

  const cells = useMemo(() => buildCalendarGrid(viewYear, viewMonth), [viewYear, viewMonth])

  function shiftView(delta: number) {
    const date = new Date(Date.UTC(viewYear, viewMonth - 1 + delta, 1))
    setViewYear(date.getUTCFullYear())
    setViewMonth(date.getUTCMonth() + 1)
  }

  function selectDate(year: number, month: number, day: number) {
    onChange(datePartsToISO(year, month, day))
    setViewYear(year)
    setViewMonth(month)
  }

  function goToday() {
    selectDate(today.year, today.month, today.day)
  }

  return (
    <div className="bill-schedule">
      <div className="bill-calendar">
        <header className="bill-calendar-head">
          <span className="bill-calendar-title">{monthYearShort(viewYear, viewMonth)}</span>
          <div className="bill-calendar-nav">
            <button type="button" aria-label="Mês anterior" onClick={() => shiftView(-1)}>
              ‹
            </button>
            <button type="button" className="today-dot" aria-label="Hoje" onClick={goToday}>
              ○
            </button>
            <button type="button" aria-label="Próximo mês" onClick={() => shiftView(1)}>
              ›
            </button>
          </div>
        </header>

        <div className="bill-calendar-weekdays" aria-hidden="true">
          {WEEKDAYS.map((label, index) => (
            <span key={`${label}-${index}`}>{label}</span>
          ))}
        </div>

        <div className="bill-calendar-grid" role="grid" aria-label="Calendário">
          {cells.map((cell) => {
            const iso = datePartsToISO(cell.year, cell.month, cell.day)
            const isSelected = iso === selectedISO
            const isToday = iso === todayISO
            return (
              <button
                key={iso}
                type="button"
                role="gridcell"
                className={[
                  'bill-calendar-day',
                  cell.outside ? 'outside' : '',
                  isSelected ? 'selected' : '',
                  isToday ? 'today' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
                aria-label={iso}
                aria-pressed={isSelected}
                onClick={() => selectDate(cell.year, cell.month, cell.day)}
              >
                {cell.day}
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}
