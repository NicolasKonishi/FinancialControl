import { useEffect, useState } from 'react'
import {
  monthKey,
  monthKeyLabel,
  monthShort,
  monthYearShort,
  parseMonthKey,
} from './billSchedule'

type Props = {
  value: string
  minMonth: string
  onChange: (monthKey: string) => void
}

function currentMonthKey() {
  const now = new Date()
  return monthKey(now.getFullYear(), now.getMonth() + 1)
}

export function BillEndMonthPicker({ value, minMonth, onChange }: Props) {
  const selected = parseMonthKey(value || minMonth)
  const [open, setOpen] = useState(false)
  const [viewYear, setViewYear] = useState(selected.year)

  useEffect(() => {
    setViewYear(parseMonthKey(value || minMonth).year)
  }, [value, minMonth])

  function shiftYear(delta: number) {
    setViewYear((y) => y + delta)
  }

  function pickMonth(month: number) {
    const key = monthKey(viewYear, month)
    if (key < minMonth) return
    onChange(key)
    setOpen(false)
  }

  function goCurrentMonth() {
    const now = currentMonthKey()
    const { year } = parseMonthKey(now)
    setViewYear(year)
    if (now >= minMonth) {
      onChange(now)
      setOpen(false)
    }
  }

  return (
    <div className="bill-end-month">
      <button
        type="button"
        className={`bill-schedule-btn bill-end-month-trigger${open ? ' active' : ''}`}
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
          <rect x="4" y="5" width="16" height="15" rx="2" />
          <path d="M8 3v4M16 3v4M4 10h16" />
        </svg>
        <span>Termina em</span>
        <small>{monthKeyLabel(value)}</small>
      </button>

      {open ? (
        <div className="bill-calendar bill-end-month-panel" role="dialog" aria-label="Mês final">
          <header className="bill-calendar-head">
            <span className="bill-calendar-title">{viewYear}</span>
            <div className="bill-calendar-nav">
              <button type="button" aria-label="Ano anterior" onClick={() => shiftYear(-1)}>
                ‹
              </button>
              <button type="button" className="today-dot" aria-label="Mês atual" onClick={goCurrentMonth}>
                ○
              </button>
              <button type="button" aria-label="Próximo ano" onClick={() => shiftYear(1)}>
                ›
              </button>
            </div>
          </header>

          <div className="bill-month-grid" role="grid" aria-label={`Meses de ${viewYear}`}>
            {Array.from({ length: 12 }, (_, i) => i + 1).map((month) => {
              const key = monthKey(viewYear, month)
              const disabled = key < minMonth
              const isSelected = value === key
              return (
                <button
                  key={key}
                  type="button"
                  role="gridcell"
                  disabled={disabled}
                  className={['bill-month-cell', isSelected ? 'selected' : ''].filter(Boolean).join(' ')}
                  aria-label={monthYearShort(viewYear, month)}
                  aria-pressed={isSelected}
                  onClick={() => pickMonth(month)}
                >
                  {monthShort(month).replace('.', '')}
                </button>
              )
            })}
          </div>
        </div>
      ) : null}
    </div>
  )
}
