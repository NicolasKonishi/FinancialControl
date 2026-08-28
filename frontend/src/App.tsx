import { useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from './api'
import { CATEGORY_ICONS, CategoryGlyph, iconLabel, PencilIcon, TrashIcon } from './icons'
import { applyTheme, getPreferredTheme, persistTheme, type Theme } from './theme'
import { BillEndMonthPicker } from './BillEndMonthPicker'
import { BillSchedulePicker } from './BillSchedulePicker'
import { DayDatePicker } from './DayDatePicker'
import { repeatSummary } from './billSchedule'
import {
  billActiveInMonth,
  billChargeInMonth,
  billShareForMember,
  parseLocaleNumber,
} from './billUtils'
import {
  CREDIT_CARDS,
  creditCardFromDescription,
  creditCardLabel,
} from './creditCards'
import { InstallBanner } from './InstallBanner'
import { createInstallHint, type InstallHint } from './pwaInstall'
import {
  clearQuickAdd,
  consumeQuickAdd,
  isQuickLaunchUrl,
  matchCategoryId,
  matchMemberId,
  nowLocal,
  setQuickLaunchUrl,
} from './quickAdd'
import { QuickLaunch } from './QuickLaunch'
import type {
  Bill,
  BillAmountMode,
  BillFrequency,
  BillPayment,
  BillRecurrence,
  Category,
  CategoryIcon,
  Member,
  MonthlyForecast,
  Transaction,
} from './types'
import './index.css'

type View = 'home' | 'balance' | 'ledger' | 'bills' | 'family' | 'categories' | 'statistics'
type SheetMode = 'expense' | 'income' | 'freelance' | 'member' | 'bill' | 'category' | null
type TxPrefill = {
  category_id?: number
  member_id?: number
  description?: string
  amount?: string
  date?: string
  credit_card?: string
}

const currency = new Intl.NumberFormat('pt-BR', {
  style: 'currency',
  currency: 'BRL',
})

const monthLabel = new Intl.DateTimeFormat('pt-BR', {
  month: 'long',
  year: 'numeric',
})

function todayISO() {
  return nowLocal().date
}

function formatWhen(date: string, createdAt?: string) {
  if (createdAt) {
    const parsed = new Date(createdAt)
    if (!Number.isNaN(parsed.getTime())) {
      return new Intl.DateTimeFormat('pt-BR', {
        day: '2-digit',
        month: 'short',
        hour: '2-digit',
        minute: '2-digit',
      }).format(parsed)
    }
  }
  return date.slice(0, 10)
}

function currentMonthKey() {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

function formatDate(value: string) {
  return value.slice(0, 10)
}

function isBillOverdue(dueDay: number, year: number, month: number) {
  const now = new Date()
  const viewed = year * 12 + month
  const current = now.getFullYear() * 12 + (now.getMonth() + 1)
  if (viewed < current) return true
  if (viewed > current) return false
  return dueDay < now.getDate()
}

export default function App() {
  const now = new Date()
  const [theme, setTheme] = useState<Theme>(() => getPreferredTheme())
  const [view, setView] = useState<View>('home')
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)

  const [categories, setCategories] = useState<Category[]>([])
  const [members, setMembers] = useState<Member[]>([])
  const [bills, setBills] = useState<Bill[]>([])
  const [billPayments, setBillPayments] = useState<BillPayment[]>([])
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [forecast, setForecast] = useState<MonthlyForecast | null>(null)
  const [memberFilter, setMemberFilter] = useState<number | 'all'>('all')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [confirmDelete, setConfirmDelete] = useState<{
    title: string
    message: string
    onConfirm: () => Promise<void>
  } | null>(null)

  const [installHint, setInstallHint] = useState<InstallHint>({
    visible: false,
    isIOS: false,
    canPrompt: false,
  })
  const installCtrl = useRef<ReturnType<typeof createInstallHint> | null>(null)
  const [quickLaunch, setQuickLaunch] = useState(() =>
    typeof window !== 'undefined' ? isQuickLaunchUrl() : false,
  )
  const [quickLaunchSaving, setQuickLaunchSaving] = useState(false)

  const [sheet, setSheet] = useState<SheetMode>(null)
  const [editingTxId, setEditingTxId] = useState<number | null>(null)
  const [editingMemberId, setEditingMemberId] = useState<number | null>(null)
  const [editingBillId, setEditingBillId] = useState<number | null>(null)
  const [editingCategoryId, setEditingCategoryId] = useState<number | null>(null)

  const [txForm, setTxForm] = useState({
    category_id: 0,
    member_id: 0,
    description: '',
    amount: '',
    date: todayISO(),
    credit_card: '',
  })
  const [memberForm, setMemberForm] = useState({ name: '', monthly_salary: '' })
  const [categoryForm, setCategoryForm] = useState({
    name: '',
    description: '',
    icon: 'other' as CategoryIcon,
  })
  const [billForm, setBillForm] = useState({
    name: '',
    amount: '',
    amount_mode: 'fixed' as BillAmountMode,
    interest_rate: '',
    category_id: 0,
    member_ids: [] as number[],
    due_day: '10',
    frequency: 'monthly' as BillFrequency,
    recurrence: 'ongoing' as BillRecurrence,
    start_month: currentMonthKey(),
    end_month: '',
    notes: '',
  })

  const categoryById = useMemo(() => new Map(categories.map((c) => [c.id, c])), [categories])
  const memberById = useMemo(() => new Map(members.map((m) => [m.id, m])), [members])

  const monthTransactions = useMemo(() => {
    const prefix = `${year}-${String(month).padStart(2, '0')}`
    return [...transactions]
      .filter((tx) => formatDate(tx.date).startsWith(prefix))
      .filter((tx) => {
        if (memberFilter === 'all') return true
        return tx.member_id === memberFilter
      })
      .sort((a, b) => formatDate(b.date).localeCompare(formatDate(a.date)) || b.id - a.id)
  }, [transactions, year, month, memberFilter])

  const listedBills = useMemo(() => {
    return bills
      .filter((bill) => billActiveInMonth(bill, year, month))
      .filter((bill) => {
        if (memberFilter === 'all') return true
        return (bill.member_ids ?? []).includes(memberFilter)
      })
      .sort((a, b) => a.due_day - b.due_day || a.name.localeCompare(b.name))
  }, [bills, year, month, memberFilter])

  const paidBillIds = useMemo(
    () => new Set(billPayments.map((payment) => payment.bill_id)),
    [billPayments],
  )

  const unpaidBills = useMemo(
    () => listedBills.filter((bill) => !paidBillIds.has(bill.id)),
    [listedBills, paidBillIds],
  )
  const paidBills = useMemo(
    () => listedBills.filter((bill) => paidBillIds.has(bill.id)),
    [listedBills, paidBillIds],
  )
  const unpaidTotal = useMemo(() => {
    return unpaidBills.reduce((sum, bill) => {
      if (memberFilter === 'all') return sum + billChargeInMonth(bill, year, month)
      return sum + billShareForMember(bill, memberFilter, year, month)
    }, 0)
  }, [unpaidBills, memberFilter, year, month])

  const memberBalances = useMemo(() => {
    const list = forecast?.by_member ?? []
    const filtered =
      memberFilter === 'all' ? list : list.filter((item) => item.member_id === memberFilter)
    const monthBills = bills.filter((bill) => billActiveInMonth(bill, year, month))
    return filtered.map((item) => {
      let paidShare = 0
      let unpaidShare = 0
      for (const bill of monthBills) {
        const share = billShareForMember(bill, item.member_id, year, month)
        if (paidBillIds.has(bill.id)) paidShare += share
        else unpaidShare += share
      }
      return {
        ...item,
        paidShare,
        unpaidShare,
        current: item.total_available - item.variable_expense - paidShare,
      }
    })
  }, [forecast, memberFilter, bills, year, month, paidBillIds])

  const selectedMemberForecast = useMemo(() => {
    const list = forecast?.by_member ?? []
    if (memberFilter === 'all') return null
    return list.find((item) => item.member_id === memberFilter) ?? null
  }, [forecast, memberFilter])

  async function refresh() {
    setError('')
    setLoading(true)
    try {
      const [cats, mems, txs, billList, payments, fc] = await Promise.all([
        api.listCategories(),
        api.listMembers(),
        api.listTransactions(),
        api.listBills(),
        api.listBillPayments(year, month),
        api.monthlyForecast(year, month),
      ])
      setCategories(cats)
      setMembers(mems)
      setTransactions(txs)
      setBills(billList)
      setBillPayments(payments)
      setForecast(fc)

      const expenseCat = cats.find((c) => c.icon === 'market') ?? cats[0]
      const homeCat = cats.find((c) => c.icon === 'home') ?? expenseCat
      setTxForm((prev) => ({
        ...prev,
        category_id: prev.category_id || expenseCat?.id || 0,
        member_id: prev.member_id || mems[0]?.id || 0,
      }))
      setBillForm((prev) => ({
        ...prev,
        category_id: prev.category_id || homeCat?.id || 0,
      }))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha ao carregar')
    } finally {
      setLoading(false)
    }
  }

  function askDelete(title: string, message: string, action: () => Promise<void>) {
    setConfirmDelete({
      title,
      message,
      onConfirm: async () => {
        try {
          await action()
          setConfirmDelete(null)
          await refresh()
        } catch (err) {
          const msg = err instanceof Error ? err.message : 'Erro ao excluir'
          setError(
            msg.includes('FOREIGN KEY') || msg.includes('constraint')
              ? 'Não é possível excluir: item ainda está em uso.'
              : msg,
          )
          setConfirmDelete(null)
        }
      },
    })
  }

  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  useEffect(() => {
    void refresh()
  }, [year, month])

  useEffect(() => {
    installCtrl.current = createInstallHint(setInstallHint)
    return () => installCtrl.current?.stop()
  }, [])

  useEffect(() => {
    const sync = () => setQuickLaunch(isQuickLaunchUrl())
    sync()
    window.addEventListener('popstate', sync)
    window.addEventListener('hashchange', sync)
    return () => {
      window.removeEventListener('popstate', sync)
      window.removeEventListener('hashchange', sync)
    }
  }, [])

  useEffect(() => {
    document.title = quickLaunch ? 'Lançar no Fluxo' : 'Fluxo'
  }, [quickLaunch])

  useEffect(() => {
    if (loading) return
    const draft = consumeQuickAdd()
    if (!draft) return

    const fallback =
      draft.mode === 'freelance'
        ? (categories.find((c) => c.icon === 'freelance') ?? categories[0])
        : draft.mode === 'income'
          ? (categories.find((c) => c.icon === 'salary') ?? categories[0])
          : (categories.find((c) => c.icon === 'market' || c.icon === 'food') ?? categories[0])

    openSheet(draft.mode, {
      category_id: matchCategoryId(categories, draft.categoryQuery, fallback?.id || 0),
      member_id: matchMemberId(members, draft.memberQuery, members[0]?.id || 0),
      description: draft.description,
      amount: draft.amount,
      date: draft.date || todayISO(),
      credit_card: draft.credit_card,
    })
    clearQuickAdd()
  }, [loading])

  function toggleTheme() {
    const next: Theme = theme === 'dark' ? 'light' : 'dark'
    setTheme(next)
    persistTheme(next)
  }
  function shiftMonth(delta: number) {
    const date = new Date(Date.UTC(year, month - 1 + delta, 1))
    setYear(date.getUTCFullYear())
    setMonth(date.getUTCMonth() + 1)
  }

  function openQuickLaunch() {
    if (!isQuickLaunchUrl()) setQuickLaunchUrl(true)
    setQuickLaunch(true)
    setError('')
  }

  function closeQuickLaunch() {
    if (isQuickLaunchUrl()) setQuickLaunchUrl(false)
    setQuickLaunch(false)
  }

  async function submitQuickLaunch(place: string, amountText: string) {
    const amount = parseLocaleNumber(amountText)
    if (!place.trim()) {
      setError('Informe o lugar.')
      return false
    }
    if (!(amount > 0)) {
      setError('Informe um valor válido.')
      return false
    }
    const stamp = nowLocal()
    const category =
      categories.find((c) => c.icon === 'market' || c.icon === 'food' || c.icon === 'shopping') ??
      categories[0]
    if (!category) {
      setError('Crie uma categoria antes de lançar.')
      return false
    }
    setQuickLaunchSaving(true)
    setError('')
    try {
      await api.createTransaction({
        category_id: category.id,
        member_id: members[0]?.id || null,
        type: 'expense',
        description: place.trim(),
        amount,
        date: stamp.date,
      })
      setYear(stamp.year)
      setMonth(stamp.month)
      await refresh()
      return true
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao lançar')
      return false
    } finally {
      setQuickLaunchSaving(false)
    }
  }

  function openSheet(mode: SheetMode, prefill?: TxPrefill) {
    setEditingTxId(null)
    setEditingMemberId(null)
    setEditingBillId(null)
    setEditingCategoryId(null)
    if (mode === 'member') {
      setMemberForm({ name: '', monthly_salary: '' })
    } else if (mode === 'category') {
      setCategoryForm({ name: '', description: '', icon: 'other' })
    } else if (mode === 'bill') {
      const home = categories.find((c) => c.icon === 'home') ?? categories[0]
      const due = new Date().getDate()
      setBillForm({
        name: '',
        amount: '',
        amount_mode: 'fixed',
        interest_rate: '',
        category_id: home?.id || 0,
        member_ids: members.map((m) => m.id),
        due_day: String(due),
        frequency: 'monthly',
        recurrence: 'ongoing',
        start_month: currentMonthKey(),
        end_month: '',
        notes: '',
      })
    } else if (mode === 'freelance') {
      const freelance = categories.find((c) => c.icon === 'freelance') ?? categories[0]
      setTxForm({
        category_id: prefill?.category_id || freelance?.id || 0,
        member_id: prefill?.member_id || members[0]?.id || 0,
        description: prefill?.description || 'Freelancer',
        amount: prefill?.amount ?? '',
        date: prefill?.date || todayISO(),
        credit_card: prefill?.credit_card ?? '',
      })
    } else if (mode === 'income') {
      const salary = categories.find((c) => c.icon === 'salary') ?? categories[0]
      setTxForm({
        category_id: prefill?.category_id || salary?.id || 0,
        member_id: prefill?.member_id || members[0]?.id || 0,
        description: prefill?.description || 'Entrada extra',
        amount: prefill?.amount ?? '',
        date: prefill?.date || todayISO(),
        credit_card: prefill?.credit_card ?? '',
      })
    } else if (mode === 'expense') {
      const market = categories.find((c) => c.icon === 'market' || c.icon === 'food') ?? categories[0]
      const credit_card = prefill?.credit_card ?? ''
      setTxForm({
        category_id: prefill?.category_id || market?.id || 0,
        member_id: prefill?.member_id || members[0]?.id || 0,
        description: prefill?.description || creditCardLabel(credit_card),
        amount: prefill?.amount ?? '',
        date: prefill?.date || todayISO(),
        credit_card,
      })
    }
    setSheet(mode)
  }

  async function submitTransaction(event: FormEvent) {
    event.preventDefault()
    if (!txForm.category_id) {
      setError('Crie ou selecione uma categoria.')
      return
    }
    const amount = parseLocaleNumber(txForm.amount)
    if (!(amount > 0)) {
      setError('Informe um valor válido.')
      return
    }
    const type = sheet === 'expense' ? 'expense' : 'income'
    const payload = {
      category_id: txForm.category_id,
      member_id: txForm.member_id || null,
      type: type as 'income' | 'expense',
      description: txForm.description.trim(),
      amount,
      date: txForm.date,
    }
    try {
      if (editingTxId) {
        await api.updateTransaction(editingTxId, payload)
      } else {
        await api.createTransaction(payload)
      }
      setSheet(null)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao salvar lançamento')
    }
  }

  async function submitMember(event: FormEvent) {
    event.preventDefault()
    const payload = {
      name: memberForm.name,
      monthly_salary: Number(memberForm.monthly_salary || 0),
    }
    try {
      if (editingMemberId) {
        await api.updateMember(editingMemberId, payload)
      } else {
        await api.createMember(payload)
      }
      setSheet(null)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao salvar membro')
    }
  }

  async function submitCategory(event: FormEvent) {
    event.preventDefault()
    const payload = {
      name: categoryForm.name.trim(),
      description: categoryForm.description.trim(),
      icon: categoryForm.icon,
    }
    try {
      if (editingCategoryId) {
        await api.updateCategory(editingCategoryId, payload)
      } else {
        await api.createCategory(payload)
      }
      setSheet(null)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao salvar categoria')
    }
  }

  async function submitBill(event: FormEvent) {
    event.preventDefault()
    if (!billForm.category_id) {
      setError('Selecione uma categoria para a conta.')
      return
    }
    if (billForm.recurrence === 'until' && !billForm.end_month) {
      setError('Escolha o mês final.')
      return
    }
    const isInterest = billForm.amount_mode === 'interest'
    const amount = parseLocaleNumber(billForm.amount)
    const interestRate = parseLocaleNumber(billForm.interest_rate)
    if (isInterest && !(amount > 0)) {
      setError('Informe o encargo do 1º mês (use vírgula ou ponto para decimais).')
      return
    }
    if (isInterest && billForm.interest_rate !== '' && (Number.isNaN(interestRate) || interestRate < 0)) {
      setError('Juros ao mês inválido.')
      return
    }
    if (!isInterest && !(amount > 0)) {
      setError('Informe o valor da conta.')
      return
    }
    const payload = {
      name: billForm.name,
      amount,
      amount_mode: billForm.amount_mode,
      interest_rate: isInterest ? (Number.isNaN(interestRate) ? 0 : interestRate) : 0,
      category_id: billForm.category_id,
      member_ids: billForm.member_ids,
      due_day: Number(billForm.due_day),
      frequency: billForm.frequency,
      recurrence: billForm.recurrence,
      start_month: billForm.start_month,
      end_month: billForm.recurrence === 'until' ? billForm.end_month : null,
      notes: billForm.notes,
    }
    try {
      if (editingBillId) {
        await api.updateBill(editingBillId, payload)
      } else {
        await api.createBill(payload)
      }
      setSheet(null)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao salvar conta')
    }
  }

  async function toggleBillPaid(bill: Bill, paid: boolean) {
    const previous = billPayments
    setBillPayments((current) => {
      if (paid) {
        if (current.some((item) => item.bill_id === bill.id)) return current
        return [
          ...current,
          { bill_id: bill.id, year, month, paid_at: new Date().toISOString() },
        ]
      }
      return current.filter((item) => item.bill_id !== bill.id)
    })
    try {
      await api.setBillPaid(bill.id, { year, month, paid })
    } catch (err) {
      setBillPayments(previous)
      setError(err instanceof Error ? err.message : 'Erro ao atualizar pagamento')
    }
  }

  const usedRatio = forecast
    ? Math.min(1, forecast.total_available > 0 ? forecast.projected_expense / forecast.total_available : 0)
    : 0
  const paceClass = usedRatio >= 1 ? 'danger' : usedRatio >= 0.85 ? 'warn' : ''

  return (
    <div className="app">
      <div className="app-main">
      <header className="top">
        <div>
          <h1>Fluxo</h1>
          <p>Família · contas · gastos</p>
        </div>
        <div className="top-controls">
          <button
            type="button"
            className="theme-toggle"
            onClick={toggleTheme}
            aria-label={theme === 'dark' ? 'Ativar modo claro' : 'Ativar modo escuro'}
            title={theme === 'dark' ? 'Modo claro' : 'Modo escuro'}
          >
            {theme === 'dark' ? (
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
                <circle cx="12" cy="12" r="4" />
                <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
                <path d="M21 14.5A8.5 8.5 0 1 1 9.5 3a7 7 0 0 0 11.5 11.5Z" />
              </svg>
            )}
          </button>
          <div className="month-switch">
            <button type="button" aria-label="Mês anterior" onClick={() => shiftMonth(-1)}>
              ‹
            </button>
            <span>{monthLabel.format(new Date(year, month - 1, 1))}</span>
            <button type="button" aria-label="Próximo mês" onClick={() => shiftMonth(1)}>
              ›
            </button>
          </div>
        </div>
      </header>

      {error && !quickLaunch ? <div className="error">{error}</div> : null}
      <InstallBanner
        hint={installHint}
        onInstall={() => void installCtrl.current?.prompt()}
        onDismiss={() => installCtrl.current?.dismiss()}
      />
      {loading ? <div className="empty">Carregando…</div> : null}

      <div className="filter-bar" aria-label="Filtro por pessoa">
        <button
          type="button"
          className={memberFilter === 'all' ? 'chip active' : 'chip'}
          onClick={() => setMemberFilter('all')}
        >
          Toda a família
        </button>
        {members.map((member) => (
          <button
            key={member.id}
            type="button"
            className={memberFilter === member.id ? 'chip active' : 'chip'}
            onClick={() => setMemberFilter(member.id)}
          >
            {member.name}
          </button>
        ))}
      </div>

      {memberFilter !== 'all' && selectedMemberForecast && view !== 'balance' && (
        <section className="card">
          <h2>{selectedMemberForecast.member_name} neste mês</h2>
          <p className={`hero-value ${selectedMemberForecast.remaining < 0 ? 'neg' : 'pos'}`}>
            {currency.format(selectedMemberForecast.remaining)}
          </p>
          <p className="meta">sobra pessoal após contas e gastos</p>
          <div className="metrics">
            <div className="metric">
              <span>Disponível</span>
              <strong>{currency.format(selectedMemberForecast.total_available)}</strong>
            </div>
            <div className="metric">
              <span>A pagar (contas)</span>
              <strong>{currency.format(selectedMemberForecast.bill_share)}</strong>
            </div>
            <div className="metric">
              <span>Gastos variáveis</span>
              <strong>{currency.format(selectedMemberForecast.variable_expense)}</strong>
            </div>
            <div className="metric">
              <span>Total a pagar</span>
              <strong>{currency.format(selectedMemberForecast.total_to_pay)}</strong>
            </div>
          </div>
        </section>
      )}

      {view === 'balance' && (
        <>
          <section className="card">
            <h2>Saldo atual</h2>
            <p className="meta" style={{ marginBottom: '0.85rem' }}>
              Salário e extras menos gastos e contas já pagas neste mês.
            </p>
            {memberBalances.length === 0 ? (
              <p className="empty">Adicione pessoas na aba Família para ver o saldo de cada um.</p>
            ) : (
              <div className="list">
                {memberBalances.map((item) => (
                  <article key={item.member_id} className="row">
                    <div className="icon-wrap">
                      <CategoryGlyph icon="salary" />
                    </div>
                    <div className="row-main">
                      <h3>{item.member_name}</h3>
                      <p>
                        ainda a pagar {currency.format(item.unpaidShare)} · sobra prevista{' '}
                        {currency.format(item.remaining)}
                      </p>
                    </div>
                    <div className={`amount ${item.current < 0 ? 'expense' : 'income'}`}>
                      {currency.format(item.current)}
                    </div>
                  </article>
                ))}
              </div>
            )}
          </section>

          <section className="card">
            <h2>Contas do mês</h2>
            <p className="meta" style={{ marginBottom: '0.85rem' }}>
              {listedBills.length === 0
                ? 'Nenhuma conta ativa neste mês.'
                : `${paidBills.length} de ${listedBills.length} pagas · falta ${currency.format(unpaidTotal)}.`}
            </p>
            {listedBills.length > 0 && (
              <div
                className={`progress checklist-progress ${
                  unpaidBills.length === 0 ? '' : unpaidBills.some((bill) => isBillOverdue(bill.due_day, year, month)) ? 'warn' : ''
                }`}
              >
                <i
                  style={{
                    width: `${listedBills.length === 0 ? 0 : (paidBills.length / listedBills.length) * 100}%`,
                  }}
                />
              </div>
            )}
            {listedBills.length === 0 ? null : (
              <>
                <p className="section-label">Ainda faltam</p>
                {unpaidBills.length === 0 ? (
                  <p className="empty">Todas as contas deste mês já foram pagas.</p>
                ) : (
                  <div className="list">
                    {unpaidBills.map((bill) => {
                      const cat = categoryById.get(bill.category_id)
                      const monthAmount = billChargeInMonth(bill, year, month)
                      const payers = (bill.member_ids ?? [])
                        .map((id) => memberById.get(id)?.name)
                        .filter(Boolean)
                        .join(', ')
                      const overdue = isBillOverdue(bill.due_day, year, month)
                      const ownShare =
                        memberFilter === 'all'
                          ? null
                          : billShareForMember(bill, memberFilter, year, month)
                      return (
                        <button
                          key={bill.id}
                          type="button"
                          className={`row checklist-row${overdue ? ' overdue' : ''}`}
                          aria-pressed="false"
                          onClick={() => void toggleBillPaid(bill, true)}
                        >
                          <span className="check-box" aria-hidden="true" />
                          <div className="row-main">
                            <h3>{bill.name}</h3>
                            <p>
                              vence dia {bill.due_day}
                              {overdue ? ' · atrasada' : ''}
                              {payers ? ` · ${payers}` : ''}
                              {cat ? ` · ${cat.name}` : ''}
                              {ownShare != null && (bill.member_ids ?? []).length > 1
                                ? ` · sua parte ${currency.format(ownShare)}`
                                : ''}
                            </p>
                          </div>
                          <div className="amount expense">{currency.format(monthAmount)}</div>
                        </button>
                      )
                    })}
                  </div>
                )}
                <p className="section-label">Já pagas</p>
                {paidBills.length === 0 ? (
                  <p className="empty">Nenhuma conta marcada como paga ainda.</p>
                ) : (
                  <div className="list">
                    {paidBills.map((bill) => {
                      const cat = categoryById.get(bill.category_id)
                      const monthAmount = billChargeInMonth(bill, year, month)
                      const payers = (bill.member_ids ?? [])
                        .map((id) => memberById.get(id)?.name)
                        .filter(Boolean)
                        .join(', ')
                      const ownShare =
                        memberFilter === 'all'
                          ? null
                          : billShareForMember(bill, memberFilter, year, month)
                      return (
                        <button
                          key={bill.id}
                          type="button"
                          className="row checklist-row paid"
                          aria-pressed="true"
                          onClick={() => void toggleBillPaid(bill, false)}
                        >
                          <span className="check-box on" aria-hidden="true">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4">
                              <path d="M5 12.5 9.5 17 19 7" />
                            </svg>
                          </span>
                          <div className="row-main">
                            <h3>{bill.name}</h3>
                            <p>
                              vence dia {bill.due_day}
                              {payers ? ` · ${payers}` : ''}
                              {cat ? ` · ${cat.name}` : ''}
                              {ownShare != null && (bill.member_ids ?? []).length > 1
                                ? ` · sua parte ${currency.format(ownShare)}`
                                : ''}
                            </p>
                          </div>
                          <div className="amount income">{currency.format(monthAmount)}</div>
                        </button>
                      )
                    })}
                  </div>
                )}
              </>
            )}
          </section>
        </>
      )}

      {view === 'home' && (
        <>
          {memberFilter === 'all' && (
            <div className="home-dash">
              {(forecast?.by_member?.length ?? 0) > 0 && (
                <section className="card">
                  <h2>Por pessoa</h2>
                  <p className="meta" style={{ marginBottom: '0.85rem' }}>
                    Quanto cada um paga e quanto sobra no mês.
                  </p>
                  <div className="list">
                    {(forecast?.by_member ?? []).map((item) => (
                      <article key={item.member_id} className="row">
                        <div className="icon-wrap">
                          <CategoryGlyph icon="salary" />
                        </div>
                        <div>
                          <h3>{item.member_name}</h3>
                          <p>
                            pagar {currency.format(item.total_to_pay)} · contas{' '}
                            {currency.format(item.bill_share)}
                          </p>
                        </div>
                        <div className={`amount ${item.remaining < 0 ? 'expense' : 'income'}`}>
                          {currency.format(item.remaining)}
                        </div>
                      </article>
                    ))}
                  </div>
                </section>
              )}
              <section className="card">
                <h2>Previsão do mês</h2>
                <p className={`hero-value ${(forecast?.remaining ?? 0) < 0 ? 'neg' : 'pos'}`}>
                  {currency.format(forecast?.remaining ?? 0)}
                </p>
                <p className="meta">
                  sobrando de {currency.format(forecast?.total_available ?? 0)} disponíveis
                </p>
                <div className="metrics">
                  <div className="metric">
                    <span>Salários planejados</span>
                    <strong>{currency.format(forecast?.planned_salary ?? 0)}</strong>
                  </div>
                  <div className="metric">
                    <span>Extras / freelancer</span>
                    <strong>{currency.format(forecast?.extra_income ?? 0)}</strong>
                  </div>
                  <div className="metric">
                    <span>Contas do mês</span>
                    <strong>{currency.format(forecast?.planned_bills ?? 0)}</strong>
                  </div>
                  <div className="metric">
                    <span>Total comprometido</span>
                    <strong>{currency.format(forecast?.total_expense ?? 0)}</strong>
                  </div>
                </div>
                <div className={`progress ${paceClass}`} title="Ritmo de gasto projetado">
                  <i style={{ width: `${usedRatio * 100}%` }} />
                </div>
                <p className="meta">
                  Pode gastar cerca de {currency.format(forecast?.safe_daily_spend ?? 0)} por dia nos próximos{' '}
                  {forecast?.days_remaining ?? 0} dias.
                </p>
              </section>
            </div>
          )}

          <div className="actions">
            <button type="button" className="launch-action" onClick={openQuickLaunch}>
              <strong>Lançar</strong>
              <span>lugar e valor</span>
            </button>
            <button type="button" onClick={() => openSheet('income')}>
              <strong>Entrada</strong>
              <span>dinheiro</span>
            </button>
            <button type="button" onClick={() => openSheet('freelance')}>
              <strong>Extra</strong>
              <span>freelancer</span>
            </button>
          </div>

          <section className="card">
            <h2>Últimos lançamentos</h2>
            {monthTransactions.length === 0 ? (
              <p className="meta">Nada neste mês ainda.</p>
            ) : (
              <div className="list">
                {monthTransactions.slice(0, 6).map((tx) => {
                  const cat = categoryById.get(tx.category_id)
                  return (
                    <article key={tx.id} className="row">
                      <div className="icon-wrap">
                        <CategoryGlyph icon={cat?.icon ?? 'other'} />
                      </div>
                      <div>
                        <h3>{tx.description}</h3>
                        <p>
                          {cat?.name ?? 'Categoria'} · {formatWhen(tx.date, tx.created_at)}
                        </p>
                      </div>
                      <div className={`amount ${tx.type}`}>
                        {tx.type === 'expense' ? '−' : '+'}
                        {currency.format(tx.amount)}
                      </div>
                    </article>
                  )
                })}
              </div>
            )}
          </section>
        </>
      )}

      {view === 'ledger' && (
        <section className="card">
          <h2>Tabela do mês</h2>
          <p className="meta" style={{ marginBottom: '0.85rem' }}>
            {memberFilter === 'all'
              ? 'Entradas e saídas com categoria.'
              : `Filtrado por ${selectedMemberForecast?.member_name ?? 'pessoa'}.`}
          </p>
          {monthTransactions.length === 0 ? (
            <p className="empty">Sem lançamentos neste mês.</p>
          ) : (
            <div className="list">
              {monthTransactions.map((tx) => {
                const cat = categoryById.get(tx.category_id)
                const member = tx.member_id ? memberById.get(tx.member_id) : null
                return (
                  <article key={tx.id} className="row">
                    <div className="icon-wrap">
                      <CategoryGlyph icon={cat?.icon ?? 'other'} />
                    </div>
                    <div className="row-main">
                      <h3>{tx.description}</h3>
                      <p>
                        {cat?.name ?? 'Categoria'}
                        {member ? ` · ${member.name}` : ''} · {formatWhen(tx.date, tx.created_at)}
                      </p>
                    </div>
                    <div className="row-side">
                      <div className="row-actions">
                        <button
                          type="button"
                          className="icon-btn"
                          aria-label="Editar lançamento"
                          title="Editar"
                          onClick={() => {
                            setEditingTxId(tx.id)
                            setTxForm({
                              category_id: tx.category_id,
                              member_id: tx.member_id ?? 0,
                              description: tx.description,
                              amount: String(tx.amount),
                              date: formatDate(tx.date),
                              credit_card:
                                tx.type === 'expense'
                                  ? creditCardFromDescription(tx.description)
                                  : '',
                            })
                            setSheet(tx.type === 'expense' ? 'expense' : 'income')
                          }}
                        >
                          <PencilIcon />
                        </button>
                        <button
                          type="button"
                          className="icon-btn danger"
                          aria-label="Excluir lançamento"
                          title="Excluir"
                          onClick={() =>
                            askDelete(
                              'Excluir lançamento?',
                              `Tem certeza que deseja excluir “${tx.description}”? Essa ação não pode ser desfeita.`,
                              () => api.deleteTransaction(tx.id),
                            )
                          }
                        >
                          <TrashIcon />
                        </button>
                      </div>
                      <div className={`amount ${tx.type}`}>
                        {tx.type === 'expense' ? '−' : '+'}
                        {currency.format(tx.amount)}
                      </div>
                    </div>
                  </article>
                )
              })}
            </div>
          )}
          <button type="button" className="primary" onClick={openQuickLaunch}>
            Lançar lugar e valor
          </button>
          <button type="button" className="ghost ledger-extra" onClick={() => openSheet('expense')}>
            Saída completa
          </button>
        </section>
      )}

      {view === 'bills' && (
        <section className="card">
          <h2>Contas mensais</h2>
          <p className="meta" style={{ marginBottom: '0.85rem' }}>
            {memberFilter === 'all'
              ? 'Escolha a frequência (mês, semana, dia…) e se termina ou não.'
              : `Contas em que ${selectedMemberForecast?.member_name ?? 'esta pessoa'} participa.`}
          </p>
          {listedBills.length === 0 ? (
            <p className="empty">Nenhuma conta ativa neste mês.</p>
          ) : (
            <div className="list">
              {listedBills.map((bill) => {
                const cat = categoryById.get(bill.category_id)
                const monthAmount = billChargeInMonth(bill, year, month)
                const payers = (bill.member_ids ?? [])
                  .map((id) => memberById.get(id)?.name)
                  .filter(Boolean)
                  .join(', ')
                return (
                  <article key={bill.id} className="row">
                    <div className="icon-wrap">
                      <CategoryGlyph icon={cat?.icon ?? 'home'} />
                    </div>
                    <div className="row-main">
                      <h3>{bill.name}</h3>
                      <p>
                        {currency.format(monthAmount)} ·{' '}
                        {bill.amount_mode === 'interest' || bill.amount_mode === 'schedule'
                          ? `${(bill.interest_rate ?? 0) > 0 ? `${bill.interest_rate}% ao mês · ` : ''}evolução de obra`
                          : repeatSummary(bill.frequency || 'monthly', bill.start_month, bill.due_day)}{' '}
                        · {cat?.name ?? 'Categoria'} ·{' '}
                        {bill.recurrence === 'ongoing'
                          ? 'sem término'
                          : `até ${bill.end_month}`}
                        {payers ? ` · ${payers}` : ' · sem responsáveis'}
                      </p>
                    </div>
                    <div className="row-side">
                      <div className="row-actions">
                        <button
                          type="button"
                          className="icon-btn"
                          aria-label="Editar conta"
                          title="Editar"
                          onClick={() => {
                            setEditingBillId(bill.id)
                            setBillForm({
                              name: bill.name,
                              amount: String(bill.amount),
                              amount_mode:
                                bill.amount_mode === 'schedule' ? 'interest' : bill.amount_mode || 'fixed',
                              interest_rate:
                                bill.interest_rate > 0 ? String(bill.interest_rate) : '',
                              category_id: bill.category_id,
                              member_ids: bill.member_ids ?? [],
                              due_day: String(bill.due_day),
                              frequency: bill.frequency || 'monthly',
                              recurrence: bill.recurrence,
                              start_month: bill.start_month,
                              end_month: bill.end_month ?? '',
                              notes: bill.notes ?? '',
                            })
                            setSheet('bill')
                          }}
                        >
                          <PencilIcon />
                        </button>
                        <button
                          type="button"
                          className="icon-btn danger"
                          aria-label="Excluir conta"
                          title="Excluir"
                          onClick={() =>
                            askDelete(
                              'Excluir conta?',
                              `Tem certeza que deseja excluir “${bill.name}”? Essa ação não pode ser desfeita.`,
                              () => api.deleteBill(bill.id),
                            )
                          }
                        >
                          <TrashIcon />
                        </button>
                      </div>
                      <div className="amount expense">{currency.format(monthAmount)}</div>
                    </div>
                  </article>
                )
              })}
            </div>
          )}
          <button type="button" className="primary" onClick={() => openSheet('bill')}>
            Nova conta
          </button>
        </section>
      )}

      {view === 'family' && (
        <section className="card">
          <h2>Família</h2>
          <p className="meta" style={{ marginBottom: '0.85rem' }}>
            Salário mensal de cada pessoa entra na previsão.
          </p>
          {members.length === 0 ? (
            <p className="empty">Adicione quem mora / contribui em casa.</p>
          ) : (
            <div className="list">
              {members.map((member) => (
                <article key={member.id} className="row">
                  <div className="icon-wrap">
                    <CategoryGlyph icon="salary" />
                  </div>
                  <div className="row-main">
                    <h3>{member.name}</h3>
                    <p>Salário mensal</p>
                  </div>
                  <div className="row-side">
                    <div className="row-actions">
                      <button
                        type="button"
                        className="icon-btn"
                        aria-label="Editar pessoa"
                        title="Editar"
                        onClick={() => {
                          setEditingMemberId(member.id)
                          setMemberForm({
                            name: member.name,
                            monthly_salary: String(member.monthly_salary),
                          })
                          setSheet('member')
                        }}
                      >
                        <PencilIcon />
                      </button>
                      <button
                        type="button"
                        className="icon-btn danger"
                        aria-label="Excluir pessoa"
                        title="Excluir"
                        onClick={() =>
                          askDelete(
                            'Excluir pessoa?',
                            `Tem certeza que deseja excluir “${member.name}” da família? Essa ação não pode ser desfeita.`,
                            () => api.deleteMember(member.id),
                          )
                        }
                      >
                        <TrashIcon />
                      </button>
                    </div>
                    <div className="amount income">{currency.format(member.monthly_salary)}</div>
                  </div>
                </article>
              ))}
            </div>
          )}
          <button type="button" className="primary" onClick={() => openSheet('member')}>
            Adicionar pessoa
          </button>
        </section>
      )}
        {view === 'statistics' && (
            <section className="card">
              <h2>Estatisticas</h2>
              <p className="meta" style={{ marginBottom: '0.85rem' }}>
                Gráficos
              </p>
            </section>
        )}
      {view === 'categories' && (
        <section className="card">
          <h2>Categorias</h2>
          <p className="meta" style={{ marginBottom: '0.85rem' }}>
            Organize gastos, entradas e contas.
          </p>
          {categories.length === 0 ? (
            <p className="empty">Nenhuma categoria cadastrada.</p>
          ) : (
            <div className="list">
              {categories.map((cat) => (
                <article key={cat.id} className="row">
                  <div className="icon-wrap">
                    <CategoryGlyph icon={cat.icon} />
                  </div>
                  <div className="row-main">
                    <h3>{cat.name}</h3>
                    <p>
                      {iconLabel(cat.icon)}
                      {cat.description ? ` · ${cat.description}` : ''}
                    </p>
                  </div>
                  <div className="row-actions">
                    <button
                      type="button"
                      className="icon-btn"
                      aria-label="Editar categoria"
                      title="Editar"
                      onClick={() => {
                        setEditingCategoryId(cat.id)
                        setCategoryForm({
                          name: cat.name,
                          description: cat.description ?? '',
                          icon: (cat.icon as CategoryIcon) || 'other',
                        })
                        setSheet('category')
                      }}
                    >
                      <PencilIcon />
                    </button>
                    <button
                      type="button"
                      className="icon-btn danger"
                      aria-label="Excluir categoria"
                      title="Excluir"
                      onClick={() =>
                        askDelete(
                          'Excluir categoria?',
                          `Tem certeza que deseja excluir “${cat.name}”? Essa ação não pode ser desfeita.`,
                          () => api.deleteCategory(cat.id),
                        )
                      }
                    >
                      <TrashIcon />
                    </button>
                  </div>
                </article>
              ))}
            </div>
          )}
          <button type="button" className="primary" onClick={() => openSheet('category')}>
            Nova categoria
          </button>
        </section>
      )}
      </div>

      <nav className="bottom-nav" aria-label="Navegação">
        <button type="button" className={view === 'home' ? 'active' : ''} onClick={() => setView('home')}>
          Início
        </button>
        <button type="button" className={view === 'balance' ? 'active' : ''} onClick={() => setView('balance')}>
          Saldo
        </button>
        <button type="button" className={view === 'ledger' ? 'active' : ''} onClick={() => setView('ledger')}>
          Gastos
        </button>
        <button type="button" className={view === 'bills' ? 'active' : ''} onClick={() => setView('bills')}>
          Contas
        </button>
        <button type="button" className={view === 'family' ? 'active' : ''} onClick={() => setView('family')}>
          Família
        </button>
        <button type="button" className={view === 'statistics' ? 'active' : ''} onClick={() => setView('statistics')}>
          Estatísticas
        </button>
        <button
          type="button"
          className={view === 'categories' ? 'active' : ''}
          onClick={() => setView('categories')}
        >
          Categorias
        </button>
      </nav>

      {sheet && (
        <div className="sheet" role="dialog" aria-modal="true">
          <div className="sheet-panel">
            <header>
              <h2>
                {sheet === 'member'
                  ? editingMemberId
                    ? 'Editar pessoa'
                    : 'Nova pessoa'
                  : sheet === 'category'
                    ? editingCategoryId
                      ? 'Editar categoria'
                      : 'Nova categoria'
                  : sheet === 'bill'
                    ? editingBillId
                      ? 'Editar conta'
                      : 'Nova conta'
                  : sheet === 'expense'
                    ? editingTxId
                      ? 'Editar saída'
                      : 'Nova saída'
                    : sheet === 'freelance'
                      ? 'Entrada freelancer'
                      : editingTxId
                        ? 'Editar entrada'
                        : 'Nova entrada'}
              </h2>
              <button type="button" className="ghost" onClick={() => setSheet(null)}>
                Fechar
              </button>
            </header>

            {sheet === 'member' ? (
              <form className="form" onSubmit={submitMember}>
                <label>
                  Nome
                  <input
                    required
                    value={memberForm.name}
                    onChange={(e) => setMemberForm((p) => ({ ...p, name: e.target.value }))}
                    placeholder="Ex.: Ana"
                  />
                </label>
                <label>
                  Salário mensal
                  <input
                    required
                    type="number"
                    min="0"
                    step="0.01"
                    value={memberForm.monthly_salary}
                    onChange={(e) => setMemberForm((p) => ({ ...p, monthly_salary: e.target.value }))}
                    placeholder="0,00"
                  />
                </label>
                <button className="primary" type="submit">
                  Salvar
                </button>
              </form>
            ) : sheet === 'category' ? (
              <form className="form" onSubmit={submitCategory}>
                <label>
                  Nome
                  <input
                    required
                    value={categoryForm.name}
                    onChange={(e) => setCategoryForm((p) => ({ ...p, name: e.target.value }))}
                    placeholder="Ex.: Mercado, Pet…"
                  />
                </label>
                <label>
                  Descrição
                  <input
                    value={categoryForm.description}
                    onChange={(e) => setCategoryForm((p) => ({ ...p, description: e.target.value }))}
                    placeholder="Opcional"
                  />
                </label>
                <div>
                  <span className="field-label">Ícone</span>
                  <div className="icon-picker">
                    {CATEGORY_ICONS.map((icon) => (
                      <button
                        key={icon}
                        type="button"
                        className={categoryForm.icon === icon ? 'active' : ''}
                        aria-label={iconLabel(icon)}
                        title={iconLabel(icon)}
                        onClick={() => setCategoryForm((p) => ({ ...p, icon }))}
                      >
                        <CategoryGlyph icon={icon} />
                      </button>
                    ))}
                  </div>
                </div>
                <button className="primary" type="submit">
                  Salvar
                </button>
              </form>
            ) : sheet === 'bill' ? (
              <form className="form bill-form" onSubmit={submitBill} noValidate>
                <div className="bill-form-layout">
                  <BillSchedulePicker
                    startMonth={billForm.start_month}
                    dueDay={Number(billForm.due_day) || 1}
                    frequency={billForm.frequency}
                    onDateChange={(startMonth, dueDay) =>
                      setBillForm((p) => ({
                        ...p,
                        start_month: startMonth,
                        due_day: String(dueDay),
                      }))
                    }
                    onFrequencyChange={(frequency) =>
                      setBillForm((p) => ({ ...p, frequency }))
                    }
                  />
                  <div className="bill-form-fields">
                    <label>
                      Nome
                      <input
                        required
                        value={billForm.name}
                        onChange={(e) => setBillForm((p) => ({ ...p, name: e.target.value }))}
                        placeholder="Internet, Luz…"
                      />
                    </label>
                    <label>
                      Tipo de valor
                      <select
                        value={billForm.amount_mode}
                        onChange={(e) => {
                          const amount_mode = e.target.value as BillAmountMode
                          setBillForm((p) => ({ ...p, amount_mode }))
                        }}
                      >
                        <option value="fixed">Valor fixo</option>
                        <option value="interest">Evolução de obra (% juros ao mês)</option>
                      </select>
                    </label>
                    {billForm.amount_mode === 'fixed' ? (
                      <label>
                        Valor
                        <input
                          required
                          type="text"
                          inputMode="decimal"
                          value={billForm.amount}
                          onChange={(e) => setBillForm((p) => ({ ...p, amount: e.target.value }))}
                        />
                      </label>
                    ) : (
                      <>
                        <label>
                          Encargo inicial (1º mês)
                          <input
                            required
                            type="text"
                            inputMode="decimal"
                            value={billForm.amount}
                            onChange={(e) => setBillForm((p) => ({ ...p, amount: e.target.value }))}
                            placeholder="316,86"
                          />
                        </label>
                        <label>
                          Juros ao mês (%)
                          <input
                            type="text"
                            inputMode="decimal"
                            value={billForm.interest_rate}
                            onChange={(e) =>
                              setBillForm((p) => ({ ...p, interest_rate: e.target.value }))
                            }
                            placeholder="0,27"
                          />
                        </label>
                      </>
                    )}
                    <label>
                      Duração
                      <select
                        value={billForm.recurrence}
                        onChange={(e) =>
                          setBillForm((p) => ({
                            ...p,
                            recurrence: e.target.value as BillRecurrence,
                            end_month:
                              e.target.value === 'until' && !p.end_month ? p.start_month : p.end_month,
                          }))
                        }
                      >
                        <option value="ongoing">Sem duração de término</option>
                        <option value="until">Termina em…</option>
                      </select>
                    </label>
                    {billForm.recurrence === 'until' ? (
                      <BillEndMonthPicker
                        value={billForm.end_month}
                        minMonth={billForm.start_month}
                        onChange={(end_month) => setBillForm((p) => ({ ...p, end_month }))}
                      />
                    ) : null}
                    <label>
                      Categoria
                      <select
                        required
                        value={billForm.category_id || ''}
                        onChange={(e) =>
                          setBillForm((p) => ({ ...p, category_id: Number(e.target.value) }))
                        }
                      >
                        {categories.map((cat) => (
                          <option key={cat.id} value={cat.id}>
                            {cat.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label>
                      Quem paga
                      <div className="check-list check-list-compact">
                        {members.length === 0 ? (
                          <span className="meta">Cadastre a família primeiro.</span>
                        ) : (
                          members.map((member) => {
                            const checked = billForm.member_ids.includes(member.id)
                            return (
                              <label key={member.id} className="check-item">
                                <input
                                  type="checkbox"
                                  checked={checked}
                                  onChange={() => {
                                    setBillForm((prev) => ({
                                      ...prev,
                                      member_ids: checked
                                        ? prev.member_ids.filter((id) => id !== member.id)
                                        : [...prev.member_ids, member.id],
                                    }))
                                  }}
                                />
                                <span>{member.name}</span>
                              </label>
                            )
                          })
                        )}
                      </div>
                    </label>
                    <label>
                      Obs.
                      <input
                        value={billForm.notes}
                        onChange={(e) => setBillForm((p) => ({ ...p, notes: e.target.value }))}
                        placeholder="Opcional"
                      />
                    </label>
                  </div>
                </div>
                <button className="primary" type="submit">
                  Salvar
                </button>
              </form>
            ) : (
              <form className="form bill-form" onSubmit={submitTransaction}>
                <div className="bill-form-layout">
                  <DayDatePicker
                    value={txForm.date}
                    onChange={(date) => setTxForm((p) => ({ ...p, date }))}
                  />
                  <div className="bill-form-fields">
                    {sheet === 'expense' ? (
                      <label>
                        Pagamento
                        <select
                          value={txForm.credit_card}
                          onChange={(e) => {
                            const credit_card = e.target.value
                            const label = creditCardLabel(credit_card)
                            setTxForm((p) => {
                              const previousLabel = creditCardLabel(p.credit_card)
                              const descriptionWasAuto =
                                !p.description.trim() ||
                                p.description.trim() === previousLabel
                              return {
                                ...p,
                                credit_card,
                                description:
                                  credit_card && descriptionWasAuto
                                    ? label
                                    : !credit_card && p.description.trim() === previousLabel
                                      ? ''
                                      : p.description,
                              }
                            })
                          }}
                        >
                          {CREDIT_CARDS.map((card) => (
                            <option key={card.id || 'none'} value={card.id}>
                              {card.label}
                            </option>
                          ))}
                        </select>
                      </label>
                    ) : null}
                    <label>
                      Descrição
                      <input
                        required
                        value={txForm.description}
                        onChange={(e) => setTxForm((p) => ({ ...p, description: e.target.value }))}
                        placeholder={
                          sheet === 'expense'
                            ? 'Mercado, Pix, almoço…'
                            : 'Pagamento, freelance…'
                        }
                      />
                    </label>
                    <label>
                      Valor
                      <input
                        required
                        type="text"
                        inputMode="decimal"
                        value={txForm.amount}
                        onChange={(e) => setTxForm((p) => ({ ...p, amount: e.target.value }))}
                      />
                    </label>
                    <label>
                      Categoria
                      <select
                        required
                        value={txForm.category_id || ''}
                        onChange={(e) =>
                          setTxForm((p) => ({ ...p, category_id: Number(e.target.value) }))
                        }
                      >
                        {categories.map((cat) => (
                          <option key={cat.id} value={cat.id}>
                            {cat.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label>
                      Pessoa
                      <select
                        value={txForm.member_id || ''}
                        onChange={(e) =>
                          setTxForm((p) => ({ ...p, member_id: Number(e.target.value) }))
                        }
                      >
                        <option value="">Sem vínculo</option>
                        {members.map((member) => (
                          <option key={member.id} value={member.id}>
                            {member.name}
                          </option>
                        ))}
                      </select>
                    </label>
                  </div>
                </div>
                <button className="primary" type="submit">
                  Salvar
                </button>
              </form>
            )}
          </div>
        </div>
      )}
      {quickLaunch ? (
        <QuickLaunch
          saving={quickLaunchSaving}
          error={error}
          onSave={submitQuickLaunch}
          onClose={closeQuickLaunch}
        />
      ) : null}
      {confirmDelete ? (
        <div
          className="sheet"
          role="dialog"
          aria-modal="true"
          aria-labelledby="confirm-delete-title"
          onClick={() => setConfirmDelete(null)}
        >
          <div className="sheet-panel confirm-panel" onClick={(e) => e.stopPropagation()}>
            <header>
              <h2 id="confirm-delete-title">{confirmDelete.title}</h2>
              <button type="button" className="ghost" onClick={() => setConfirmDelete(null)}>
                Fechar
              </button>
            </header>
            <p className="confirm-message">{confirmDelete.message}</p>
            <div className="confirm-actions">
              <button type="button" className="ghost confirm-cancel" onClick={() => setConfirmDelete(null)}>
                Cancelar
              </button>
              <button
                type="button"
                className="primary confirm-danger"
                onClick={() => {
                  void confirmDelete.onConfirm()
                }}
              >
                Excluir
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}
