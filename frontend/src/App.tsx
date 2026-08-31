import { useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from './api'
import {
  BillsNavIcon,
  CATEGORY_ICONS,
  CategoryGlyph,
  ChevronLeftIcon,
  HomeNavIcon,
  iconLabel,
  LedgerNavIcon,
  PencilIcon,
  SavingsNavIcon,
  SettingsNavIcon,
  StatsNavIcon,
  TrashIcon,
} from './icons'
import { applyTheme, getPreferredTheme, persistTheme, type Theme } from './theme'
import { BillEndMonthPicker } from './BillEndMonthPicker'
import { BillSchedulePicker } from './BillSchedulePicker'
import { DayDatePicker } from './DayDatePicker'
import {
  billActiveInMonth,
  billChargeInMonth,
  billShareForMember,
  parseLocaleNumber,
} from './billUtils'
import { creditCardLabel } from './creditCards'
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
  SavingsEndKind,
  SavingsGoal,
  SavingsPlan,
  Transaction,
  Wallet,
  WalletKind,
} from './types'
import './index.css'

type View = 'home' | 'ledger' | 'bills' | 'savings' | 'family' | 'categories' | 'statistics' | 'settings'
type SheetMode = 'expense' | 'income' | 'freelance' | 'member' | 'bill' | 'category' | 'goal' | 'wallet' | 'pay-bill' | null
type WalletOwner = number | 'joint'
type MemberBenefitKey = 'checking' | 'credit' | 'meal' | 'food' | 'fuel' | 'company'

const WALLET_KINDS: { kind: WalletKind; label: string }[] = [
  { kind: 'checking', label: 'Pix / conta' },
  { kind: 'credit', label: 'Cartão' },
  { kind: 'benefit', label: 'Benefício' },
  { kind: 'company', label: 'Empresa' },
  { kind: 'savings', label: 'Caixinha' },
]

const MEMBER_BENEFITS: {
  key: MemberBenefitKey
  name: string
  kind: WalletKind
  aliases: string[]
}[] = [
  { key: 'checking', name: 'Conta / Pix', kind: 'checking', aliases: ['conta', 'pix'] },
  { key: 'credit', name: 'Cartão de crédito', kind: 'credit', aliases: ['cartão', 'credito', 'crédito'] },
  { key: 'meal', name: 'Vale-alimentação', kind: 'benefit', aliases: ['alimentação', 'alimentacao'] },
  { key: 'food', name: 'Vale-refeição', kind: 'benefit', aliases: ['refeição', 'refeicao'] },
  { key: 'fuel', name: 'Vale-combustível', kind: 'benefit', aliases: ['combustível', 'combustivel', 'mobilidade'] },
  { key: 'company', name: 'Empresa', kind: 'company', aliases: ['empresa'] },
]

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

function monthsAheadKey(delta: number) {
  const now = new Date()
  const date = new Date(Date.UTC(now.getFullYear(), now.getMonth() + delta, 1))
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`
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

function goalMonthlyPlan(goal: SavingsGoal) {
  if (goal.end_kind === 'none') return 0
  if (goal.saved_amount >= goal.target_amount) return 0
  const remaining = Math.round((goal.target_amount - goal.saved_amount) * 100) / 100
  if (remaining <= 0 || goal.monthly_amount <= 0) return 0
  return Math.min(goal.monthly_amount, remaining)
}

function goalYieldLabel(goal: SavingsGoal) {
  if (!(goal.cdi_annual > 0)) return ''
  return `103% CDI (${goal.yield_annual.toLocaleString('pt-BR', { maximumFractionDigits: 2 })}% a.a.)`
}

function goalProjectedAmount(goal: SavingsGoal, year: number, month: number) {
  const monthly = goalMonthlyPlan(goal)
  let months = 12
  if (goal.end_kind === 'date' && goal.end_month) {
    const [endYear, endMonth] = goal.end_month.split('-').map(Number)
    months = (endYear - year) * 12 + (endMonth - month) + 1
    if (months < 1) months = 1
  } else if (goal.end_kind === 'amount' && monthly > 0 && goal.target_amount > goal.saved_amount) {
    months = Math.max(1, Math.ceil((goal.target_amount - goal.saved_amount) / monthly))
  }
  const annual = (goal.cdi_annual / 100) * 1.03
  const rate = annual > 0 ? Math.pow(1 + annual, 1 / 12) - 1 : 0
  let value = goal.saved_amount
  for (let i = 0; i < months; i += 1) {
    value = value * (1 + rate) + monthly
  }
  return { months, amount: Math.round(value * 100) / 100 }
}

function isCompanyMemberName(name: string) {
  return name.trim().toLowerCase() === 'empresa'
}

function payWalletsForMember(wallets: Wallet[], memberId: number) {
  const own = wallets.filter((wallet) => wallet.member_id === memberId && wallet.kind !== 'savings')
  const cash = own.filter((wallet) => wallet.kind !== 'benefit')
  return cash.length > 0 ? cash : own
}

function spendWalletsForMember(wallets: Wallet[], memberId: number) {
  return wallets
    .filter((wallet) => wallet.member_id === memberId && wallet.kind !== 'savings')
    .sort((a, b) => walletKindOrder(a.kind) - walletKindOrder(b.kind) || a.name.localeCompare(b.name))
}

function preferredSpendWallet(wallets: Wallet[], memberId: number) {
  const options = spendWalletsForMember(wallets, memberId)
  const checking = options.find((wallet) => wallet.kind === 'checking')
  return checking ?? options[0] ?? null
}

function goalCheckingWallets(wallets: Wallet[], memberIds: number[]) {
  const allowed = new Set(memberIds)
  return wallets
    .filter(
      (wallet) =>
        wallet.kind === 'checking' && wallet.member_id != null && allowed.has(wallet.member_id),
    )
    .sort(
      (a, b) =>
        (a.member_id ?? 0) - (b.member_id ?? 0) || a.name.localeCompare(b.name, 'pt-BR'),
    )
}

function walletMatchesBenefit(wallet: Wallet, benefit: (typeof MEMBER_BENEFITS)[number]) {
  if (wallet.kind !== benefit.kind) return false
  if (benefit.kind === 'checking' || benefit.kind === 'credit' || benefit.kind === 'company') return true
  const name = wallet.name.toLowerCase()
  return benefit.aliases.some((alias) => name.includes(alias))
}

function memberBenefitKeys(wallets: Wallet[], memberId: number) {
  const own = wallets.filter((wallet) => wallet.member_id === memberId)
  return MEMBER_BENEFITS.filter((benefit) => own.some((wallet) => walletMatchesBenefit(wallet, benefit))).map(
    (benefit) => benefit.key,
  )
}

function walletKindLabel(kind: string) {
  return WALLET_KINDS.find((item) => item.kind === kind)?.label ?? 'Saldo'
}

function walletKindIcon(kind: string): CategoryIcon {
  switch (kind) {
    case 'checking':
      return 'salary'
    case 'credit':
      return 'shopping'
    case 'savings':
      return 'investment'
    case 'benefit':
      return 'food'
    case 'company':
      return 'freelance'
    default:
      return 'other'
  }
}

function walletKindOrder(kind: string) {
  const index = WALLET_KINDS.findIndex((item) => item.kind === kind)
  return index < 0 ? 99 : index
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
  const [savingsGoals, setSavingsGoals] = useState<SavingsGoal[]>([])
  const [wallets, setWallets] = useState<Wallet[]>([])
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
  const [editingGoalId, setEditingGoalId] = useState<number | null>(null)
  const [editingWalletId, setEditingWalletId] = useState<number | null>(null)
  const [payingBill, setPayingBill] = useState<Bill | null>(null)
  const [selectedGoalId, setSelectedGoalId] = useState<number | null>(null)
  const [goalAdjustAmount, setGoalAdjustAmount] = useState('')
  const [goalAdjustWalletId, setGoalAdjustWalletId] = useState(0)

  const [txForm, setTxForm] = useState({
    category_id: 0,
    member_id: 0,
    wallet_id: 0,
    description: '',
    amount: '',
    date: todayISO(),
    credit_card: '',
  })
  const [memberForm, setMemberForm] = useState({
    name: '',
    monthly_salary: '',
    benefits: ['checking'] as MemberBenefitKey[],
  })
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
  const [goalForm, setGoalForm] = useState({
    name: '',
    target_amount: '',
    end_kind: 'amount' as SavingsEndKind,
    end_month: '',
    member_ids: [] as number[],
    notes: '',
  })
  const [goalPlan, setGoalPlan] = useState<SavingsPlan | null>(null)
  const [walletForm, setWalletForm] = useState({
    name: '',
    kind: 'checking' as WalletKind,
    member_id: 'joint' as WalletOwner,
    balance: '',
  })
  const [payForm, setPayForm] = useState({ member_id: 0, wallet_id: 0 })

  const categoryById = useMemo(() => new Map(categories.map((c) => [c.id, c])), [categories])
  const memberById = useMemo(() => new Map(members.map((m) => [m.id, m])), [members])
  const personMembers = useMemo(
    () => members.filter((member) => !isCompanyMemberName(member.name)),
    [members],
  )

  function walletOwnerLabel(wallet: Wallet) {
    if (wallet.member_id == null) return 'Família'
    return memberById.get(wallet.member_id)?.name ?? 'Pessoa'
  }

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
  const paymentByBillId = useMemo(
    () => new Map(billPayments.map((payment) => [payment.bill_id, payment])),
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

  const listedGoals = useMemo(() => {
    return [...savingsGoals]
      .filter((goal) => {
        if (memberFilter === 'all') return true
        return (goal.member_ids ?? []).includes(memberFilter)
      })
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [savingsGoals, memberFilter])

  const listedWallets = useMemo(() => {
    const filtered = wallets.filter((wallet) => {
      if (memberFilter === 'all') return true
      return wallet.member_id == null || wallet.member_id === memberFilter
    })
    return [...filtered].sort((a, b) => {
      const aJoint = a.member_id == null ? 0 : 1
      const bJoint = b.member_id == null ? 0 : 1
      if (aJoint !== bJoint) return aJoint - bJoint
      const aMember = a.member_id ?? 0
      const bMember = b.member_id ?? 0
      if (aMember !== bMember) return aMember - bMember
      const kindDiff = walletKindOrder(a.kind) - walletKindOrder(b.kind)
      if (kindDiff !== 0) return kindDiff
      return a.name.localeCompare(b.name)
    })
  }, [wallets, memberFilter])

  const cashWallets = useMemo(
    () => listedWallets.filter((wallet) => wallet.kind !== 'savings'),
    [listedWallets],
  )
  const cashTotal = useMemo(
    () => cashWallets.reduce((sum, wallet) => sum + wallet.balance, 0),
    [cashWallets],
  )

  const billsByGroup = useMemo(() => {
    const joint: Bill[] = []
    const byMember = new Map<number, Bill[]>()
    for (const member of personMembers) byMember.set(member.id, [])
    for (const bill of listedBills) {
      const ids = bill.member_ids ?? []
      if (ids.length === 1) {
        const list = byMember.get(ids[0])
        if (list) list.push(bill)
        else joint.push(bill)
      } else {
        joint.push(bill)
      }
    }
    return { joint, byMember }
  }, [listedBills, personMembers])

  const selectedMemberForecast = useMemo(() => {
    const list = forecast?.by_member ?? []
    if (memberFilter === 'all') return null
    return list.find((item) => item.member_id === memberFilter) ?? null
  }, [forecast, memberFilter])

  async function refresh() {
    setError('')
    setLoading(true)
    try {
      const [cats, mems, txs, billList, payments, goals, fc, walletList] = await Promise.all([
        api.listCategories(),
        api.listMembers(),
        api.listTransactions(),
        api.listBills(),
        api.listBillPayments(year, month),
        api.listSavingsGoals(),
        api.monthlyForecast(year, month),
        api.listWallets(),
      ])
      setCategories(cats)
      setMembers(mems)
      setTransactions(txs)
      setBills(billList)
      setBillPayments(payments)
      setSavingsGoals(goals)
      setForecast(fc)
      setWallets(walletList)

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
    if (sheet !== 'goal') return
    if (goalForm.end_kind === 'none') {
      setGoalPlan(null)
      return
    }
    const target = parseLocaleNumber(goalForm.target_amount)
    if (!(target > 0)) {
      setGoalPlan(null)
      return
    }
    if (goalForm.end_kind === 'date' && !goalForm.end_month) {
      setGoalPlan(null)
      return
    }
    const saved = editingGoalId
      ? (savingsGoals.find((goal) => goal.id === editingGoalId)?.saved_amount ?? 0)
      : 0
    const timer = window.setTimeout(() => {
      void api
        .planSavings({
          end_kind: goalForm.end_kind,
          target,
          end_month: goalForm.end_month || undefined,
          members: goalForm.member_ids.length,
          saved,
        })
        .then(setGoalPlan)
        .catch(() => setGoalPlan(null))
    }, 250)
    return () => window.clearTimeout(timer)
  }, [
    sheet,
    goalForm.end_kind,
    goalForm.target_amount,
    goalForm.end_month,
    goalForm.member_ids,
    editingGoalId,
    savingsGoals,
  ])

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
      const memberId = members[0]?.id || 0
      await api.createTransaction({
        category_id: category.id,
        member_id: memberId || null,
        wallet_id: preferredSpendWallet(wallets, memberId)?.id ?? null,
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
    setEditingGoalId(null)
    setEditingWalletId(null)
    setPayingBill(null)
    if (mode === 'member') {
      setMemberForm({ name: '', monthly_salary: '', benefits: ['checking'] })
    } else if (mode === 'category') {
      setCategoryForm({ name: '', description: '', icon: 'other' })
    } else if (mode === 'wallet') {
      setWalletForm({
        name: '',
        kind: 'checking',
        member_id: memberFilter === 'all' ? 'joint' : memberFilter,
        balance: '',
      })
    } else if (mode === 'goal') {
      setGoalPlan(null)
      setGoalForm({
        name: '',
        target_amount: '',
        end_kind: 'amount',
        end_month: '',
        member_ids: personMembers.map((m) => m.id),
        notes: '',
      })
    } else if (mode === 'bill') {
      const home = categories.find((c) => c.icon === 'home') ?? categories[0]
      const due = new Date().getDate()
      setBillForm({
        name: '',
        amount: '',
        amount_mode: 'fixed',
        interest_rate: '',
        category_id: home?.id || 0,
        member_ids: personMembers.map((m) => m.id),
        due_day: String(due),
        frequency: 'monthly',
        recurrence: 'ongoing',
        start_month: currentMonthKey(),
        end_month: '',
        notes: '',
      })
    } else if (mode === 'freelance') {
      const freelance = categories.find((c) => c.icon === 'freelance') ?? categories[0]
      const memberId = prefill?.member_id || members[0]?.id || 0
      setTxForm({
        category_id: prefill?.category_id || freelance?.id || 0,
        member_id: memberId,
        wallet_id: preferredSpendWallet(wallets, memberId)?.id ?? 0,
        description: prefill?.description || 'Freelancer',
        amount: prefill?.amount ?? '',
        date: prefill?.date || todayISO(),
        credit_card: prefill?.credit_card ?? '',
      })
    } else if (mode === 'income') {
      const salary = categories.find((c) => c.icon === 'salary') ?? categories[0]
      const memberId = prefill?.member_id || members[0]?.id || 0
      setTxForm({
        category_id: prefill?.category_id || salary?.id || 0,
        member_id: memberId,
        wallet_id: preferredSpendWallet(wallets, memberId)?.id ?? 0,
        description: prefill?.description || 'Salário recebido',
        amount: prefill?.amount ?? '',
        date: prefill?.date || todayISO(),
        credit_card: prefill?.credit_card ?? '',
      })
    } else if (mode === 'expense') {
      const market = categories.find((c) => c.icon === 'market' || c.icon === 'food') ?? categories[0]
      const credit_card = prefill?.credit_card ?? ''
      const memberId = prefill?.member_id || members[0]?.id || 0
      setTxForm({
        category_id: prefill?.category_id || market?.id || 0,
        member_id: memberId,
        wallet_id: preferredSpendWallet(wallets, memberId)?.id ?? 0,
        description: prefill?.description || (credit_card ? creditCardLabel(credit_card) : ''),
        amount: prefill?.amount ?? '',
        date: prefill?.date || todayISO(),
        credit_card,
      })
    }
    setSheet(mode)
  }

  function openWalletSheet(wallet: Wallet) {
    setEditingTxId(null)
    setEditingMemberId(null)
    setEditingBillId(null)
    setEditingCategoryId(null)
    setEditingGoalId(null)
    setEditingWalletId(wallet.id)
    setWalletForm({
      name: wallet.name,
      kind: (WALLET_KINDS.some((item) => item.kind === wallet.kind)
        ? wallet.kind
        : 'checking') as WalletKind,
      member_id: wallet.member_id ?? 'joint',
      balance: String(wallet.balance).replace('.', ','),
    })
    setSheet('wallet')
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
    if (!txForm.wallet_id) {
      setError('Escolha o método de pagamento / conta.')
      return
    }
    const payload = {
      category_id: txForm.category_id,
      member_id: txForm.member_id || null,
      wallet_id: txForm.wallet_id || null,
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
      const saved = editingMemberId
        ? await api.updateMember(editingMemberId, payload)
        : await api.createMember(payload)
      const current = wallets.filter((wallet) => wallet.member_id === saved.id)
      for (const benefit of MEMBER_BENEFITS) {
        const existing = current.find((wallet) => walletMatchesBenefit(wallet, benefit))
        const wanted = memberForm.benefits.includes(benefit.key)
        if (wanted && !existing) {
          await api.createWallet({
            name: benefit.name,
            kind: benefit.kind,
            member_id: saved.id,
            balance: 0,
          })
        } else if (!wanted && existing) {
          await api.deleteWallet(existing.id)
        }
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

  async function submitGoal(event: FormEvent) {
    event.preventDefault()
    if (!goalForm.name.trim()) {
      setError('Informe o nome da meta.')
      return
    }
    if (goalForm.member_ids.length === 0) {
      setError('Escolha quem participa da meta.')
      return
    }
    const target =
      goalForm.end_kind === 'none' ? 0 : parseLocaleNumber(goalForm.target_amount)
    if (goalForm.end_kind !== 'none' && !(target > 0)) {
      setError('Informe o valor da meta.')
      return
    }
    if (goalForm.end_kind === 'date' && !goalForm.end_month) {
      setError('Escolha até quando juntar.')
      return
    }
    const payload = {
      name: goalForm.name.trim(),
      target_amount: target,
      member_ids: goalForm.member_ids,
      notes: goalForm.notes.trim(),
      end_kind: goalForm.end_kind,
      end_month: goalForm.end_kind === 'none' ? null : goalForm.end_month || null,
    }
    try {
      if (editingGoalId) {
        await api.updateSavingsGoal(editingGoalId, payload)
      } else {
        await api.createSavingsGoal(payload)
      }
      setSheet(null)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao salvar meta')
    }
  }

  async function submitWallet(event: FormEvent) {
    event.preventDefault()
    if (!walletForm.name.trim()) {
      setError('Informe o nome da conta.')
      return
    }
    const balance = parseLocaleNumber(walletForm.balance || '0')
    if (Number.isNaN(balance)) {
      setError('Informe um saldo válido.')
      return
    }
    const payload = {
      name: walletForm.name.trim(),
      kind: walletForm.kind,
      member_id: walletForm.member_id === 'joint' ? null : walletForm.member_id,
      balance,
    }
    try {
      if (editingWalletId) {
        await api.updateWallet(editingWalletId, payload)
      } else {
        await api.createWallet(payload)
      }
      setSheet(null)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao salvar saldo')
    }
  }

  function openGoalDetail(goal: SavingsGoal) {
    const options = goalCheckingWallets(wallets, goal.member_ids ?? [])
    setSelectedGoalId(goal.id)
    setGoalAdjustAmount('')
    setGoalAdjustWalletId(options[0]?.id ?? 0)
    setView('savings')
  }

  async function adjustSelectedGoal(direction: 'add' | 'subtract') {
    if (!selectedGoalId) return
    const amount = parseLocaleNumber(goalAdjustAmount)
    if (!(amount > 0)) {
      setError('Informe um valor para mover.')
      return
    }
    const options = goalCheckingWallets(
      wallets,
      listedGoals.find((goal) => goal.id === selectedGoalId)?.member_ids ?? [],
    )
    const walletId = options.some((wallet) => wallet.id === goalAdjustWalletId)
      ? goalAdjustWalletId
      : (options[0]?.id ?? 0)
    if (!walletId) {
      setError('Escolha a conta de alguém desta caixinha.')
      return
    }
    try {
      await api.adjustSavings(selectedGoalId, {
        amount: direction === 'add' ? amount : -amount,
        wallet_id: walletId,
      })
      setGoalAdjustAmount('')
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao atualizar a caixinha')
    }
  }

  function openPayBill(bill: Bill) {
    const assigned = bill.member_ids ?? []
    const preferred =
      memberFilter !== 'all' && (assigned.length === 0 || assigned.includes(memberFilter))
        ? memberFilter
        : assigned[0] ?? members[0]?.id ?? 0
    const options = payWalletsForMember(wallets, preferred)
    setPayingBill(bill)
    setPayForm({
      member_id: preferred,
      wallet_id: options[0]?.id ?? 0,
    })
    setSheet('pay-bill')
  }

  async function submitPayBill(event: FormEvent) {
    event.preventDefault()
    if (!payingBill) return
    if (!payForm.member_id) {
      setError('Escolha quem pagou.')
      return
    }
    try {
      await api.setBillPaid(payingBill.id, {
        year,
        month,
        paid: true,
        paid_by_member_id: payForm.member_id,
        wallet_id: payForm.wallet_id || null,
      })
      setSheet(null)
      setPayingBill(null)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao pagar conta')
    }
  }

  async function markBillUnpaid(bill: Bill) {
    try {
      await api.setBillPaid(bill.id, { year, month, paid: false })
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao desfazer pagamento')
    }
  }

  const usedRatio = forecast
    ? Math.min(1, forecast.total_available > 0 ? forecast.projected_expense / forecast.total_available : 0)
    : 0
  const paceClass = usedRatio >= 1 ? 'danger' : usedRatio >= 0.85 ? 'warn' : ''
  const isConfigView = view === 'settings' || view === 'family' || view === 'categories'
  const homeForecastRemaining =
    memberFilter === 'all' ? (forecast?.remaining ?? 0) : (selectedMemberForecast?.remaining ?? 0)
  const homeForecastAvailable =
    memberFilter === 'all' ? (forecast?.total_available ?? 0) : (selectedMemberForecast?.total_available ?? 0)
  const selectedGoal = listedGoals.find((goal) => goal.id === selectedGoalId) ?? null
  const goalMoveWallets = selectedGoal
    ? goalCheckingWallets(wallets, selectedGoal.member_ids ?? [])
    : []
  const goalFundingWalletId = goalMoveWallets.some((wallet) => wallet.id === goalAdjustWalletId)
    ? goalAdjustWalletId
    : (goalMoveWallets[0]?.id ?? 0)

  function openBillEditor(bill: Bill) {
    setEditingBillId(bill.id)
    setBillForm({
      name: bill.name,
      amount: String(bill.amount),
      amount_mode: bill.amount_mode === 'schedule' ? 'interest' : bill.amount_mode || 'fixed',
      interest_rate: bill.interest_rate > 0 ? String(bill.interest_rate) : '',
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
  }

  function openGoalEditor(goal: SavingsGoal) {
    setEditingGoalId(goal.id)
    setGoalForm({
      name: goal.name,
      target_amount: goal.target_amount ? String(goal.target_amount) : '',
      end_kind: goal.end_kind || 'amount',
      end_month: goal.end_month ?? '',
      member_ids: goal.member_ids ?? [],
      notes: goal.notes ?? '',
    })
    setSheet('goal')
  }

  function renderBillRow(bill: Bill) {
    const cat = categoryById.get(bill.category_id)
    const monthAmount = billChargeInMonth(bill, year, month)
    const paid = paidBillIds.has(bill.id)
    const payment = paymentByBillId.get(bill.id)
    const paidByName = payment?.paid_by_member_id
      ? memberById.get(payment.paid_by_member_id)?.name
      : null
    const payers = (bill.member_ids ?? [])
      .map((id) => memberById.get(id)?.name)
      .filter(Boolean)
      .join(', ')
    const overdue = !paid && isBillOverdue(bill.due_day, year, month)
    const ids = bill.member_ids ?? []
    const shareLabel =
      ids.length > 1
        ? ` · ${ids.length} pessoas · ${currency.format(monthAmount / ids.length)} cada`
        : ''
    return (
      <article key={bill.id} className={`row bill-manage-row${paid ? ' paid' : ''}${overdue ? ' overdue' : ''}`}>
        <button
          type="button"
          className={`check-pay${paid ? ' on' : ''}`}
          aria-pressed={paid}
          aria-label={paid ? 'Desmarcar paga' : 'Marcar como paga'}
          onClick={() => (paid ? void markBillUnpaid(bill) : openPayBill(bill))}
        >
          {paid ? (
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4">
              <path d="M5 12.5 9.5 17 19 7" />
            </svg>
          ) : null}
        </button>
        <div className="row-main">
          <h3>{bill.name}</h3>
          <p>
            vence dia {bill.due_day}
            {overdue ? ' · atrasada' : ''}
            {paid && paidByName ? ` · pago por ${paidByName}` : ''}
            {payers && !paid ? ` · ${payers}` : ''}
            {cat ? ` · ${cat.name}` : ''}
            {shareLabel}
          </p>
        </div>
        <div className="row-side">
          <div className="row-actions">
            <button type="button" className="icon-btn" aria-label="Editar conta" onClick={() => openBillEditor(bill)}>
              <PencilIcon />
            </button>
            <button
              type="button"
              className="icon-btn danger"
              aria-label="Excluir conta"
              onClick={() =>
                askDelete('Excluir conta?', `Tem certeza que deseja excluir “${bill.name}”? Essa ação não pode ser desfeita.`, () =>
                  api.deleteBill(bill.id),
                )
              }
            >
              <TrashIcon />
            </button>
          </div>
          <div className={`amount ${paid ? 'income' : 'expense'}`}>{currency.format(monthAmount)}</div>
        </div>
      </article>
    )
  }

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
          {!isConfigView && (
            <div className="month-switch">
              <button type="button" aria-label="Mês anterior" onClick={() => shiftMonth(-1)}>
                ‹
              </button>
              <span>{monthLabel.format(new Date(year, month - 1, 1))}</span>
              <button type="button" aria-label="Próximo mês" onClick={() => shiftMonth(1)}>
                ›
              </button>
            </div>
          )}
        </div>
      </header>

      {error && !quickLaunch ? <div className="error">{error}</div> : null}
      <InstallBanner
        hint={installHint}
        onInstall={() => void installCtrl.current?.prompt()}
        onDismiss={() => installCtrl.current?.dismiss()}
      />
      {loading ? <div className="empty">Carregando…</div> : null}

      {!isConfigView && view !== 'statistics' && (
        <div className="filter-bar" aria-label="Filtro por pessoa">
          <button
            type="button"
            className={memberFilter === 'all' ? 'chip active' : 'chip'}
            onClick={() => setMemberFilter('all')}
          >
            Toda a família
          </button>
          {personMembers.map((member) => (
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
      )}

      {view === 'home' && (
        <>
          <div className="home-dash">
            <section className="card saldo-card">
              <h2>Saldo</h2>
              <p className={`hero-value ${cashTotal < 0 ? 'neg' : 'pos'}`}>
                {currency.format(cashTotal)}
              </p>
              <p className="meta">
                {memberFilter === 'all'
                  ? 'conjunto da família — escolha alguém acima para ver só o individual'
                  : 'desta pessoa, incluindo o que é conjunto'}
              </p>
              {memberFilter === 'all' ? (
                <div className="list" style={{ marginTop: '0.85rem' }}>
                  {personMembers.map((member) => {
                    const total = wallets
                      .filter((wallet) => wallet.member_id === member.id && wallet.kind !== 'savings')
                      .reduce((sum, wallet) => sum + wallet.balance, 0)
                    return (
                      <button
                        key={member.id}
                        type="button"
                        className="row"
                        onClick={() => setMemberFilter(member.id)}
                      >
                        <div className="icon-wrap">
                          <CategoryGlyph icon="salary" />
                        </div>
                        <div className="row-main">
                          <h3>{member.name}</h3>
                          <p>saldo individual</p>
                        </div>
                        <div className={`amount ${total < 0 ? 'expense' : 'income'}`}>
                          {currency.format(total)}
                        </div>
                      </button>
                    )
                  })}
                </div>
              ) : cashWallets.length === 0 ? (
                <p className="empty" style={{ marginTop: '0.85rem' }}>
                  Nenhum saldo nesta pessoa. Cadastre os benefícios em Configuração.
                </p>
              ) : (
                <div className="list" style={{ marginTop: '0.85rem' }}>
                  {cashWallets.map((wallet) => (
                    <button
                      key={wallet.id}
                      type="button"
                      className="row"
                      onClick={() => openWalletSheet(wallet)}
                    >
                      <div className="icon-wrap">
                        <CategoryGlyph icon={walletKindIcon(wallet.kind)} />
                      </div>
                      <div className="row-main">
                        <h3>{wallet.name}</h3>
                        <p>
                          {walletKindLabel(wallet.kind)} · {walletOwnerLabel(wallet)}
                        </p>
                      </div>
                      <div className={`amount ${wallet.balance < 0 ? 'expense' : 'income'}`}>
                        {currency.format(wallet.balance)}
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </section>
            <section className="card">
              <h2>Previsão do mês</h2>
              <p className={`forecast-value ${homeForecastRemaining < 0 ? 'neg' : 'pos'}`}>
                {currency.format(homeForecastRemaining)}
              </p>
              <p className="meta">
                sobrando de {currency.format(homeForecastAvailable)} disponíveis
                {memberFilter === 'all' ? ' na família' : ''}
              </p>
              <div className="metrics">
                {memberFilter === 'all' ? (
                  <>
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
                      <span>Gastos variáveis</span>
                      <strong>{currency.format(forecast?.total_expense ? forecast.total_expense - forecast.planned_bills : 0)}</strong>
                    </div>
                  </>
                ) : (
                  <>
                    <div className="metric">
                      <span>Disponível</span>
                      <strong>{currency.format(selectedMemberForecast?.total_available ?? 0)}</strong>
                    </div>
                    <div className="metric">
                      <span>A pagar (contas)</span>
                      <strong>{currency.format(selectedMemberForecast?.bill_share ?? 0)}</strong>
                    </div>
                    <div className="metric">
                      <span>Gastos variáveis</span>
                      <strong>{currency.format(selectedMemberForecast?.variable_expense ?? 0)}</strong>
                    </div>
                    <div className="metric">
                      <span>Total a pagar</span>
                      <strong>{currency.format(selectedMemberForecast?.total_to_pay ?? 0)}</strong>
                    </div>
                  </>
                )}
              </div>
              {memberFilter === 'all' ? (
                <>
                  <div className={`progress ${paceClass}`} title="Ritmo de gasto projetado">
                    <i style={{ width: `${usedRatio * 100}%` }} />
                  </div>
                  <p className="meta">
                    Pode gastar cerca de {currency.format(forecast?.safe_daily_spend ?? 0)} por dia nos
                    próximos {forecast?.days_remaining ?? 0} dias.
                  </p>
                </>
              ) : null}
            </section>
            <section className="card">
              <h2>Previsão para guardar</h2>
              <p className="meta" style={{ marginBottom: '0.85rem' }}>
                {memberFilter === 'all'
                  ? 'Quanto cada um deve guardar neste mês nas caixinhas.'
                  : `Quanto ${selectedMemberForecast?.member_name ?? 'esta pessoa'} deve guardar neste mês.`}
              </p>
              {(forecast?.by_member ?? []).filter((item) =>
                memberFilter === 'all' ? true : item.member_id === memberFilter,
              ).filter((item) => !isCompanyMemberName(item.member_name)).length === 0 ? (
                <p className="empty">Nenhuma meta cadastrada ainda.</p>
              ) : (
                <div className="list">
                  {(forecast?.by_member ?? [])
                    .filter((item) => (memberFilter === 'all' ? true : item.member_id === memberFilter))
                    .filter((item) => !isCompanyMemberName(item.member_name))
                    .map((item) => (
                      <article key={item.member_id} className="row">
                        <div className="icon-wrap">
                          <CategoryGlyph icon="investment" />
                        </div>
                        <div className="row-main">
                          <h3>{item.member_name}</h3>
                          <p>reservar nas metas do mês</p>
                        </div>
                        <div className={`amount ${item.savings_share > 0 ? 'expense' : 'income'}`}>
                          {currency.format(item.savings_share)}
                        </div>
                      </article>
                    ))}
                </div>
              )}
            </section>
          </div>
        </>
      )}

      {view === 'ledger' && (
        <section className="card">
          <h2>Gastos do mês</h2>
          <p className="meta" style={{ marginBottom: '0.85rem' }}>
            Compras e imprevistos — o valor sai na hora da conta de quem pagou. Salário e freelancer entram no saldo.
          </p>
          {monthTransactions.length === 0 ? (
            <p className="empty">Sem lançamentos neste mês.</p>
          ) : (
            <div className="list">
              {monthTransactions.map((tx) => {
                const cat = categoryById.get(tx.category_id)
                const member = tx.member_id ? memberById.get(tx.member_id) : null
                const wallet = tx.wallet_id ? wallets.find((item) => item.id === tx.wallet_id) : null
                return (
                  <article key={tx.id} className="row">
                    <div className="icon-wrap">
                      <CategoryGlyph icon={cat?.icon ?? 'other'} />
                    </div>
                    <div className="row-main">
                      <h3>{tx.description}</h3>
                      <p>
                        {cat?.name ?? 'Categoria'}
                        {member ? ` · ${member.name}` : ''}
                        {wallet ? ` · ${wallet.name}` : ''} · {formatWhen(tx.date, tx.created_at)}
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
                              wallet_id:
                                tx.wallet_id ??
                                preferredSpendWallet(wallets, tx.member_id ?? 0)?.id ??
                                0,
                              description: tx.description,
                              amount: String(tx.amount),
                              date: formatDate(tx.date),
                              credit_card: '',
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
          <div className="actions">
            <button type="button" className="launch-action" onClick={openQuickLaunch}>
              <strong>Rápido</strong>
              <span>lugar e valor</span>
            </button>
            <button type="button" onClick={() => openSheet('expense')}>
              <strong>Gasto</strong>
              <span>mercado, imprevisto…</span>
            </button>
            <button type="button" onClick={() => openSheet('income')}>
              <strong>Salário</strong>
              <span>recebido</span>
            </button>
            <button type="button" onClick={() => openSheet('freelance')}>
              <strong>Freelancer</strong>
              <span>entrada extra</span>
            </button>
          </div>
        </section>
      )}

      {view === 'bills' && (
        <>
          <section className="card">
            <h2>Contas do mês</h2>
            <p className="meta" style={{ marginBottom: '0.85rem' }}>
              {listedBills.length === 0
                ? 'Nenhuma conta ativa neste mês.'
                : `${paidBills.length} de ${listedBills.length} pagas · falta ${currency.format(unpaidTotal)}. Marque para subtrair de quem pagou.`}
            </p>
            {listedBills.length > 0 && (
              <div
                className={`progress checklist-progress ${
                  unpaidBills.length === 0
                    ? ''
                    : unpaidBills.some((bill) => isBillOverdue(bill.due_day, year, month))
                      ? 'warn'
                      : ''
                }`}
              >
                <i
                  style={{
                    width: `${listedBills.length === 0 ? 0 : (paidBills.length / listedBills.length) * 100}%`,
                  }}
                />
              </div>
            )}
            {listedBills.length === 0 ? (
              <p className="empty">Cadastre aluguel, internet, escola…</p>
            ) : memberFilter === 'all' ? (
              <>
                {billsByGroup.joint.length > 0 ? (
                  <>
                    <p className="section-label">Conjuntas (divididas)</p>
                    <div className="list">{billsByGroup.joint.map((bill) => renderBillRow(bill))}</div>
                  </>
                ) : null}
                {personMembers.map((member) => {
                  const items = billsByGroup.byMember.get(member.id) ?? []
                  if (items.length === 0) return null
                  return (
                    <div key={member.id}>
                      <p className="section-label">{member.name}</p>
                      <div className="list">{items.map((bill) => renderBillRow(bill))}</div>
                    </div>
                  )
                })}
              </>
            ) : (
              <div className="list">{listedBills.map((bill) => renderBillRow(bill))}</div>
            )}
            <button type="button" className="primary" onClick={() => openSheet('bill')}>
              Nova conta
            </button>
          </section>
        </>
      )}

      {view === 'savings' && selectedGoal ? (
        <section className="card">
          <div className="section-head">
            <button
              type="button"
              className="icon-btn"
              aria-label="Voltar para caixinhas"
              onClick={() => setSelectedGoalId(null)}
            >
              <ChevronLeftIcon />
            </button>
            <h2>{selectedGoal.name}</h2>
          </div>
          <p className={`hero-value ${selectedGoal.saved_amount < 0 ? 'neg' : 'pos'}`}>
            {currency.format(selectedGoal.saved_amount)}
          </p>
          <p className="meta">
            {selectedGoal.target_amount > 0
              ? `guardados de ${currency.format(selectedGoal.target_amount)}`
              : 'guardados nesta caixinha'}
            {goalYieldLabel(selectedGoal) ? ` · ${goalYieldLabel(selectedGoal)}` : ''}
          </p>
          {selectedGoal.target_amount > 0 ? (
            <div className="progress checklist-progress" style={{ margin: '0.85rem 0' }}>
              <i
                style={{
                  width: `${Math.min(100, (selectedGoal.saved_amount / selectedGoal.target_amount) * 100)}%`,
                }}
              />
            </div>
          ) : null}
          {(() => {
            const projection = goalProjectedAmount(selectedGoal, year, month)
            return (
              <div className="metrics" style={{ marginTop: '0.85rem' }}>
                <div className="metric">
                  <span>Previsão CDI</span>
                  <strong>{currency.format(projection.amount)}</strong>
                </div>
                <div className="metric">
                  <span>Em</span>
                  <strong>
                    {projection.months} {projection.months === 1 ? 'mês' : 'meses'}
                  </strong>
                </div>
                <div className="metric">
                  <span>Guardar / mês</span>
                  <strong>{currency.format(goalMonthlyPlan(selectedGoal))}</strong>
                </div>
              </div>
            )
          })()}
          <p className="meta" style={{ margin: '1rem 0 0.55rem' }}>
            Adicionar ou tirar dinheiro. Sai ou volta da conta de quem está nesta caixinha — vales não entram.
          </p>
          <form
            className="goal-save"
            onSubmit={(event) => {
              event.preventDefault()
              void adjustSelectedGoal('add')
            }}
          >
            <label>
              Valor
              <input
                type="text"
                inputMode="decimal"
                value={goalAdjustAmount}
                onChange={(e) => setGoalAdjustAmount(e.target.value)}
                placeholder="0,00"
              />
            </label>
            <div className="goal-move-actions">
              <button className="primary" type="submit" disabled={goalMoveWallets.length === 0}>
                Adicionar
              </button>
              <button
                type="button"
                className="ghost"
                disabled={goalMoveWallets.length === 0}
                onClick={() => void adjustSelectedGoal('subtract')}
              >
                Subtrair
              </button>
            </div>
          </form>
          {goalMoveWallets.length > 0 ? (
            <label style={{ marginTop: '0.75rem' }}>
              De qual conta
              <select
                value={goalFundingWalletId || ''}
                onChange={(e) => setGoalAdjustWalletId(Number(e.target.value))}
              >
                {goalMoveWallets.map((wallet) => {
                  const owner = wallet.member_id ? memberById.get(wallet.member_id)?.name : null
                  return (
                    <option key={wallet.id} value={wallet.id}>
                      {owner ? `${owner} · ` : ''}
                      {wallet.name} · {currency.format(wallet.balance)}
                    </option>
                  )
                })}
              </select>
            </label>
          ) : (
            <p className="empty">Cadastre a conta (Pix) de quem está nesta caixinha em Configuração.</p>
          )}
          <div className="row-actions" style={{ marginTop: '1rem' }}>
            <button type="button" className="ghost" onClick={() => openGoalEditor(selectedGoal)}>
              Editar
            </button>
            <button
              type="button"
              className="ghost danger"
              onClick={() =>
                askDelete(
                  'Remover caixinha?',
                  `Tem certeza que deseja excluir “${selectedGoal.name}”?`,
                  async () => {
                    await api.deleteSavingsGoal(selectedGoal.id)
                    setSelectedGoalId(null)
                  },
                )
              }
            >
              Remover caixinha
            </button>
          </div>
        </section>
      ) : view === 'savings' ? (
        <section className="card">
          <h2>Caixinhas</h2>
          <p className="meta" style={{ marginBottom: '0.85rem' }}>
            Toque para ver o guardado e a previsão a 103% do CDI.
          </p>
          {listedGoals.length === 0 ? (
            <p className="empty">Crie uma caixinha para guardar dinheiro.</p>
          ) : (
            <div className="goal-grid">
              {listedGoals.map((goal) => {
                const ratio = Math.min(1, goal.target_amount > 0 ? goal.saved_amount / goal.target_amount : 0)
                return (
                  <button
                    key={goal.id}
                    type="button"
                    className="piggy"
                    onClick={() => openGoalDetail(goal)}
                  >
                    <div className="icon-wrap">
                      <CategoryGlyph icon="investment" />
                    </div>
                    <h3>{goal.name}</h3>
                    <p className="piggy-value">{currency.format(goal.saved_amount)}</p>
                    <p>
                      {goal.target_amount > 0
                        ? `de ${currency.format(goal.target_amount)}`
                        : goalYieldLabel(goal) || 'sem prazo'}
                    </p>
                    {goal.target_amount > 0 ? (
                      <div className="progress checklist-progress">
                        <i style={{ width: `${ratio * 100}%` }} />
                      </div>
                    ) : null}
                  </button>
                )
              })}
            </div>
          )}
          <button type="button" className="primary" onClick={() => openSheet('goal')}>
            Nova caixinha
          </button>
        </section>
      ) : null}


      {view === 'settings' && (
        <section className="card">
          <h2>Configuração</h2>
          <p className="meta" style={{ marginBottom: '0.85rem' }}>
            Membros da família, benefícios e categorias.
          </p>
          <div className="list">
            <button type="button" className="row settings-item" onClick={() => setView('family')}>
              <div className="icon-wrap">
                <CategoryGlyph icon="salary" />
              </div>
              <div className="row-main">
                <h3>Família</h3>
                <p>
                  {members.length === 0
                    ? 'Adicione quem mora / contribui e os benefícios de cada um'
                    : members.length === 1
                      ? '1 pessoa'
                      : `${members.length} pessoas`}
                </p>
              </div>
              <span className="settings-chevron" aria-hidden>
                ›
              </span>
            </button>
            <button type="button" className="row settings-item" onClick={() => setView('categories')}>
              <div className="icon-wrap">
                <CategoryGlyph icon="other" />
              </div>
              <div className="row-main">
                <h3>Categorias</h3>
                <p>
                  {categories.length === 0
                    ? 'Organize gastos, entradas e contas'
                    : categories.length === 1
                      ? '1 categoria'
                      : `${categories.length} categorias`}
                </p>
              </div>
              <span className="settings-chevron" aria-hidden>
                ›
              </span>
            </button>
          </div>
        </section>
      )}

      {view === 'family' && (
        <section className="card">
          <div className="section-head">
            <button
              type="button"
              className="icon-btn"
              aria-label="Voltar para configuração"
              title="Voltar"
              onClick={() => setView('settings')}
            >
              <ChevronLeftIcon />
            </button>
            <h2>Família</h2>
          </div>
          <p className="meta" style={{ marginBottom: '0.85rem' }}>
            Salário mensal entra na previsão. Benefícios (Pix, cartão, vales, empresa) viram formas de pagar.
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
                    <p>
                      {memberBenefitKeys(wallets, member.id)
                        .map((key) => MEMBER_BENEFITS.find((item) => item.key === key)?.name)
                        .filter(Boolean)
                        .join(' · ') || 'Sem benefícios cadastrados'}
                    </p>
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
                            benefits: memberBenefitKeys(wallets, member.id),
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
          <div className="section-head">
            <button
              type="button"
              className="icon-btn"
              aria-label="Voltar para configuração"
              title="Voltar"
              onClick={() => setView('settings')}
            >
              <ChevronLeftIcon />
            </button>
            <h2>Categorias</h2>
          </div>
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
          <HomeNavIcon />
          Início
        </button>
        <button type="button" className={view === 'bills' ? 'active' : ''} onClick={() => setView('bills')}>
          <BillsNavIcon />
          Contas
        </button>
        <button
          type="button"
          className={view === 'savings' ? 'active' : ''}
          onClick={() => {
            setSelectedGoalId(null)
            setView('savings')
          }}
        >
          <SavingsNavIcon />
          Metas
        </button>
        <button type="button" className={view === 'ledger' ? 'active' : ''} onClick={() => setView('ledger')}>
          <LedgerNavIcon />
          Gastos
        </button>
        <button type="button" className={view === 'statistics' ? 'active' : ''} onClick={() => setView('statistics')}>
          <StatsNavIcon />
          Estatísticas
        </button>
        <button
          type="button"
          className={isConfigView ? 'active' : ''}
          onClick={() => setView('settings')}
        >
          <SettingsNavIcon />
          Configuração
        </button>
      </nav>

      {sheet && (
        <div className="sheet" role="dialog" aria-modal="true">
          <div className="sheet-panel">
            <header>
              <h2>
                {sheet === 'pay-bill'
                  ? 'Quem pagou?'
                  : sheet === 'wallet'
                  ? editingWalletId
                    ? 'Editar saldo'
                    : 'Novo saldo'
                  : sheet === 'member'
                  ? editingMemberId
                    ? 'Editar pessoa'
                    : 'Nova pessoa'
                  : sheet === 'category'
                    ? editingCategoryId
                      ? 'Editar categoria'
                      : 'Nova categoria'
                    : sheet === 'goal'
                      ? editingGoalId
                        ? 'Editar meta'
                        : 'Nova meta'
                  : sheet === 'bill'
                    ? editingBillId
                      ? 'Editar conta'
                      : 'Nova conta'
                  : sheet === 'expense'
                    ? editingTxId
                      ? 'Editar gasto'
                      : 'Novo gasto'
                    : sheet === 'freelance'
                      ? 'Freelancer feito'
                      : editingTxId
                        ? 'Editar entrada'
                        : 'Salário recebido'}
              </h2>
              <button type="button" className="ghost" onClick={() => setSheet(null)}>
                Fechar
              </button>
            </header>

            {sheet === 'pay-bill' && payingBill ? (
              <form className="form" onSubmit={submitPayBill}>
                <p className="meta">
                  {payingBill.name} · {currency.format(billChargeInMonth(payingBill, year, month))}
                </p>
                <div>
                  <span className="field-label">Quem pagou</span>
                  <div className="filter-bar" style={{ marginBottom: 0 }}>
                    {members.map((member) => (
                      <button
                        key={member.id}
                        type="button"
                        className={payForm.member_id === member.id ? 'chip active' : 'chip'}
                        onClick={() => {
                          const options = payWalletsForMember(wallets, member.id)
                          setPayForm({
                            member_id: member.id,
                            wallet_id: options[0]?.id ?? 0,
                          })
                        }}
                      >
                        {member.name}
                      </button>
                    ))}
                  </div>
                </div>
                {payWalletsForMember(wallets, payForm.member_id).length > 1 ? (
                  <label>
                    De qual conta
                    <select
                      value={payForm.wallet_id || ''}
                      onChange={(e) =>
                        setPayForm((p) => ({ ...p, wallet_id: Number(e.target.value) }))
                      }
                    >
                      {payWalletsForMember(wallets, payForm.member_id).map((wallet) => (
                        <option key={wallet.id} value={wallet.id}>
                          {wallet.name} · {currency.format(wallet.balance)}
                        </option>
                      ))}
                    </select>
                  </label>
                ) : payWalletsForMember(wallets, payForm.member_id).length === 1 ? (
                  <p className="meta">
                    Sai de {payWalletsForMember(wallets, payForm.member_id)[0].name} (
                    {currency.format(payWalletsForMember(wallets, payForm.member_id)[0].balance)})
                  </p>
                ) : (
                  <p className="empty">Essa pessoa ainda não tem uma conta cadastrada.</p>
                )}
                <button
                  className="primary"
                  type="submit"
                  disabled={!payForm.member_id || payWalletsForMember(wallets, payForm.member_id).length === 0}
                >
                  Pagar
                </button>
              </form>
            ) : sheet === 'wallet' ? (
              <form className="form" onSubmit={submitWallet}>
                <label>
                  Nome
                  <input
                    required
                    value={walletForm.name}
                    onChange={(e) => setWalletForm((p) => ({ ...p, name: e.target.value }))}
                    placeholder="Conta, caixinha, vale…"
                  />
                </label>
                <div>
                  <span className="field-label">Tipo</span>
                  <div className="filter-bar" style={{ marginBottom: 0 }}>
                    {WALLET_KINDS.map((item) => (
                      <button
                        key={item.kind}
                        type="button"
                        className={walletForm.kind === item.kind ? 'chip active' : 'chip'}
                        onClick={() => setWalletForm((p) => ({ ...p, kind: item.kind }))}
                      >
                        {item.label}
                      </button>
                    ))}
                  </div>
                </div>
                <label>
                  De quem é
                  <select
                    value={walletForm.member_id === 'joint' ? 'joint' : String(walletForm.member_id)}
                    onChange={(e) =>
                      setWalletForm((p) => ({
                        ...p,
                        member_id: e.target.value === 'joint' ? 'joint' : Number(e.target.value),
                      }))
                    }
                  >
                    <option value="joint">Família (conjunto)</option>
                    {members.map((member) => (
                      <option key={member.id} value={member.id}>
                        {member.name}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  Saldo
                  <input
                    required
                    type="text"
                    inputMode="decimal"
                    value={walletForm.balance}
                    onChange={(e) => setWalletForm((p) => ({ ...p, balance: e.target.value }))}
                    placeholder="0,00"
                  />
                </label>
                <button className="primary" type="submit">
                  Salvar
                </button>
                {editingWalletId ? (
                  <button
                    type="button"
                    className="ghost danger"
                    onClick={() => {
                      const name = walletForm.name.trim() || 'este saldo'
                      setSheet(null)
                      askDelete(
                        'Excluir saldo?',
                        `Tem certeza que deseja excluir “${name}”?`,
                        () => api.deleteWallet(editingWalletId),
                      )
                    }}
                  >
                    Excluir
                  </button>
                ) : null}
              </form>
            ) : sheet === 'member' ? (
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
                <div>
                  <span className="field-label">Benefícios</span>
                  <div className="check-list check-list-compact">
                    {MEMBER_BENEFITS.map((benefit) => {
                      const checked = memberForm.benefits.includes(benefit.key)
                      return (
                        <label key={benefit.key} className="check-item">
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() => {
                              setMemberForm((prev) => ({
                                ...prev,
                                benefits: checked
                                  ? prev.benefits.filter((key) => key !== benefit.key)
                                  : [...prev.benefits, benefit.key],
                              }))
                            }}
                          />
                          <span>{benefit.name}</span>
                        </label>
                      )
                    })}
                  </div>
                </div>
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
            ) : sheet === 'goal' ? (
              <form className="form" onSubmit={submitGoal}>
                <label>
                  Qual é a meta?
                  <input
                    required
                    value={goalForm.name}
                    onChange={(e) => setGoalForm((p) => ({ ...p, name: e.target.value }))}
                    placeholder="Viagem, reserva, carro…"
                  />
                </label>
                <label>
                  Quem participa
                  <div className="check-list check-list-compact">
                    {members.length === 0 ? (
                      <span className="meta">Cadastre a família primeiro.</span>
                    ) : (
                      members.map((member) => {
                        const checked = goalForm.member_ids.includes(member.id)
                        return (
                          <label key={member.id} className="check-item">
                            <input
                              type="checkbox"
                              checked={checked}
                              onChange={() => {
                                setGoalForm((prev) => ({
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
                <div>
                  <span className="field-label">Até quando? (opcional)</span>
                  <div className="filter-bar" style={{ marginBottom: 0 }}>
                    {(
                      [
                        ['none', 'Sem prazo'],
                        ['date', 'Até uma data'],
                        ['amount', 'Até um valor'],
                      ] as const
                    ).map(([kind, label]) => (
                      <button
                        key={kind}
                        type="button"
                        className={goalForm.end_kind === kind ? 'chip active' : 'chip'}
                        onClick={() =>
                          setGoalForm((p) => ({
                            ...p,
                            end_kind: kind,
                            end_month: kind === 'date' && !p.end_month ? monthsAheadKey(12) : p.end_month,
                          }))
                        }
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                </div>
                {goalForm.end_kind === 'date' ? (
                  <>
                    <BillEndMonthPicker
                      value={goalForm.end_month}
                      minMonth={currentMonthKey()}
                      onChange={(end_month) => setGoalForm((p) => ({ ...p, end_month }))}
                    />
                    <label>
                      Quanto juntar até lá
                      <input
                        required
                        type="text"
                        inputMode="decimal"
                        value={goalForm.target_amount}
                        onChange={(e) => setGoalForm((p) => ({ ...p, target_amount: e.target.value }))}
                        placeholder="8000"
                      />
                    </label>
                  </>
                ) : null}
                {goalForm.end_kind === 'amount' ? (
                  <>
                    <label>
                      Quanto juntar
                      <input
                        required
                        type="text"
                        inputMode="decimal"
                        value={goalForm.target_amount}
                        onChange={(e) => setGoalForm((p) => ({ ...p, target_amount: e.target.value }))}
                        placeholder="8000"
                      />
                    </label>
                    <p className="meta">
                      Sem data, o plano usa 12 meses. Você pode escolher um mês final para ajustar.
                    </p>
                    <BillEndMonthPicker
                      value={goalForm.end_month}
                      minMonth={currentMonthKey()}
                      onChange={(end_month) => setGoalForm((p) => ({ ...p, end_month }))}
                    />
                  </>
                ) : null}
                {goalPlan && goalForm.end_kind !== 'none' ? (
                  <p className="goal-plan">
                    Guardar <strong>{currency.format(goalPlan.monthly_amount)}</strong> por mês
                    {goalForm.member_ids.length > 1
                      ? ` (${currency.format(goalPlan.per_member)} por pessoa)`
                      : ''}
                    , em {goalPlan.months} {goalPlan.months === 1 ? 'mês' : 'meses'}, a 103% do CDI
                    {` (${goalPlan.cdi_annual.toLocaleString('pt-BR', { maximumFractionDigits: 2 })}% a.a.)`}.
                  </p>
                ) : null}
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
                    <label>
                      Pessoa
                      <select
                        value={txForm.member_id || ''}
                        onChange={(e) => {
                          const member_id = Number(e.target.value)
                          setTxForm((p) => ({
                            ...p,
                            member_id,
                            wallet_id: preferredSpendWallet(wallets, member_id)?.id ?? 0,
                          }))
                        }}
                      >
                        <option value="">Sem vínculo</option>
                        {members.map((member) => (
                          <option key={member.id} value={member.id}>
                            {member.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label>
                      {sheet === 'expense' ? 'Como pagou' : 'Onde entra o saldo'}
                      <select
                        required
                        value={txForm.wallet_id || ''}
                        onChange={(e) =>
                          setTxForm((p) => ({ ...p, wallet_id: Number(e.target.value) }))
                        }
                      >
                        <option value="">Escolha a conta</option>
                        {spendWalletsForMember(wallets, txForm.member_id).map((wallet) => (
                          <option key={wallet.id} value={wallet.id}>
                            {wallet.name} · {walletKindLabel(wallet.kind)} · {currency.format(wallet.balance)}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label>
                      Descrição
                      <input
                        required
                        value={txForm.description}
                        onChange={(e) => setTxForm((p) => ({ ...p, description: e.target.value }))}
                        placeholder={
                          sheet === 'expense'
                            ? 'Mercado, gasolina, imprevisto…'
                            : sheet === 'freelance'
                              ? 'Freelancer feito'
                              : 'Salário recebido'
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
