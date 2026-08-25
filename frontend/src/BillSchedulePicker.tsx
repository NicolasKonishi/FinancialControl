import { useEffect, useMemo, useState } from 'react'
import type { BillFrequency } from './types'
import {
  billAnchorDate,
  buildCalendarGrid,
  datePartsToISO,
  monthDayLong,
  monthYearShort,
  parseAnchor,
  repeatSummary,
  weekdayLong,
} from './billSchedule'

type Props = {
  startMonth: string
  dueDay: number
  frequency: BillFrequency
  onDateChange: (startMonth: string, dueDay: number) => void
  onFrequencyChange: (frequency: BillFrequency) => void
}

const WEEKDAYS = ['D', 'S', 'T', 'Q', 'Q', 'S', 'S']

function todayParts() {
  const now = new Date()
  return { year: now.getFullYear(), month: now.getMonth() + 1, day: now.getDate() }
}

export function BillSchedulePicker({
  startMonth,
  dueDay,
  frequency,
  onDateChange,
  onFrequencyChange,
}: Props) {
  const selected = parseAnchor(startMonth, dueDay)
  const selectedISO = billAnchorDate(startMonth, dueDay)
  const today = todayParts()
  const todayISO = datePartsToISO(today.year, today.month, today.day)

  const [viewYear, setViewYear] = useState(selected.year)
  const [viewMonth, setViewMonth] = useState(selected.month)
  const [repeatOpen, setRepeatOpen] = useState(false)
  const [customize, setCustomize] = useState(false)

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
    onDateChange(`${year}-${String(month).padStart(2, '0')}`, day)
    setViewYear(year)
    setViewMonth(month)
  }

  function goToday() {
    selectDate(today.year, today.month, today.day)
  }

  function pickFrequency(value: BillFrequency) {
    onFrequencyChange(value)
    setRepeatOpen(false)
    setCustomize(false)
  }

  const repeatOptions = [
    { value: 'daily' as BillFrequency, label: 'Todo dia' },
    {
      value: 'weekly' as BillFrequency,
      label: `Toda semana ${weekdayLong(selectedISO)}`,
    },
    { value: 'weekdays' as BillFrequency, label: 'Todo dia útil (Seg – Sex)' },
    {
      value: 'monthly' as BillFrequency,
      label: `Todo mês no dia ${dueDay || 1}`,
    },
    {
      value: 'yearly' as BillFrequency,
      label: `Todo ano em ${monthDayLong(selectedISO)}`,
    },
  ]

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

      <div className="bill-schedule-actions">
        <button
          type="button"
          className={`bill-schedule-btn${repeatOpen ? ' active' : ''}`}
          aria-expanded={repeatOpen}
          onClick={() => {
            setRepeatOpen((open) => !open)
            setCustomize(false)
          }}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
            <path d="M17 2l4 4-4 4" />
            <path d="M3 11v-1a4 4 0 0 1 4-4h14" />
            <path d="M7 22l-4-4 4-4" />
            <path d="M21 13v1a4 4 0 0 1-4 4H3" />
          </svg>
          <span>Repetir</span>
          <small>{repeatSummary(frequency, startMonth, dueDay)}</small>
        </button>
      </div>

      {repeatOpen ? (
        <div className="repeat-menu repeat-menu-inline" role="listbox" aria-label="Frequência">
          {!customize ? (
            <>
              {repeatOptions.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  className={frequency === option.value ? 'active' : ''}
                  role="option"
                  aria-selected={frequency === option.value}
                  onClick={() => pickFrequency(option.value)}
                >
                  {option.label}
                </button>
              ))}
              <button type="button" className="repeat-customize" onClick={() => setCustomize(true)}>
                Personalizar…
              </button>
            </>
          ) : (
            <>
              <button type="button" className="repeat-back" onClick={() => setCustomize(false)}>
                ← Voltar
              </button>
              <button
                type="button"
                className={frequency === 'biweekly' ? 'active' : ''}
                onClick={() => pickFrequency('biweekly')}
              >
                A cada 2 semanas <span className="muted">{weekdayLong(selectedISO)}</span>
              </button>
              <button
                type="button"
                className={frequency === 'weekly' ? 'active' : ''}
                onClick={() => pickFrequency('weekly')}
              >
                Toda semana
              </button>
              <button
                type="button"
                className={frequency === 'monthly' ? 'active' : ''}
                onClick={() => pickFrequency('monthly')}
              >
                Todo mês
              </button>
            </>
          )}
        </div>
      ) : null}
    </div>
  )
}
