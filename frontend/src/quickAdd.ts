import { CREDIT_CARDS, type CreditCardId } from './creditCards'
import type { Category, Member } from './types'

export type QuickAddMode = 'expense' | 'income' | 'freelance'

export type QuickAddDraft = {
  mode: QuickAddMode
  amount: string
  description: string
  date: string
  credit_card: CreditCardId
  categoryQuery: string
  memberQuery: string
}

const MODE_ALIASES: Record<string, QuickAddMode> = {
  saida: 'expense',
  saída: 'expense',
  expense: 'expense',
  gasto: 'expense',
  entrada: 'income',
  income: 'income',
  extra: 'freelance',
  freelance: 'freelance',
  freelancer: 'freelance',
}

const QUERY_KEYS = [
  'nova',
  'add',
  'valor',
  'amount',
  'desc',
  'description',
  'data',
  'date',
  'cartao',
  'cartão',
  'card',
  'pagamento',
  'categoria',
  'category',
  'pessoa',
  'member',
]

const STORAGE_KEY = 'fluxo-quick-add'

export function nowLocal(at = new Date()) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return {
    date: `${at.getFullYear()}-${pad(at.getMonth() + 1)}-${pad(at.getDate())}`,
    time: `${pad(at.getHours())}:${pad(at.getMinutes())}`,
    year: at.getFullYear(),
    month: at.getMonth() + 1,
    label: new Intl.DateTimeFormat('pt-BR', {
      weekday: 'long',
      day: 'numeric',
      month: 'long',
      hour: '2-digit',
      minute: '2-digit',
    }).format(at),
  }
}

export function isQuickLaunchUrl(href = window.location.href) {
  const url = new URL(href)
  const path = url.pathname.replace(/\/$/, '')
  const hash = url.hash.replace(/^#/, '')
  return (
    url.searchParams.has('lancar') ||
    url.searchParams.has('lançar') ||
    hash === 'lancar' ||
    path === '/lancar'
  )
}

export function launchShortcutUrl() {
  return `${window.location.origin}/?lancar`
}

export function setQuickLaunchUrl(open: boolean) {
  const url = new URL(window.location.href)
  url.searchParams.delete('lancar')
  url.searchParams.delete('lançar')
  if (url.hash.replace(/^#/, '') === 'lancar') url.hash = ''
  if (url.pathname.replace(/\/$/, '') === '/lancar') url.pathname = '/'
  if (open) url.searchParams.set('lancar', '1')
  const search = url.searchParams.toString()
  const next = `${url.pathname}${search ? `?${search}` : ''}${url.hash}`
  if (open) window.history.pushState(null, '', next)
  else window.history.replaceState(null, '', next)
}

function first(params: URLSearchParams, keys: string[]) {
  for (const key of keys) {
    const value = params.get(key)
    if (value != null && value.trim()) return value.trim()
  }
  return ''
}

export function parseQuickAdd(params: URLSearchParams): QuickAddDraft | null {
  const rawMode = first(params, ['nova', 'add']).toLowerCase()
  const mode = MODE_ALIASES[rawMode]
  if (!mode) return null

  const cardQuery = first(params, ['cartao', 'cartão', 'card', 'pagamento']).toLowerCase()
  const card =
    CREDIT_CARDS.find(
      (item) =>
        item.id &&
        (item.id === cardQuery ||
          item.label.toLowerCase() === cardQuery ||
          item.label.toLowerCase().replace('é', 'e') === cardQuery),
    )?.id ?? ('' as CreditCardId)

  return {
    mode,
    amount: first(params, ['valor', 'amount']).replace('.', ','),
    description: first(params, ['desc', 'description']),
    date: first(params, ['data', 'date']),
    credit_card: card as CreditCardId,
    categoryQuery: first(params, ['categoria', 'category']),
    memberQuery: first(params, ['pessoa', 'member']),
  }
}

export function consumeQuickAdd(): QuickAddDraft | null {
  const stored = sessionStorage.getItem(STORAGE_KEY)
  if (stored) {
    try {
      return JSON.parse(stored) as QuickAddDraft
    } catch {
      sessionStorage.removeItem(STORAGE_KEY)
    }
  }

  const url = new URL(window.location.href)
  const draft = parseQuickAdd(url.searchParams)
  if (!draft) return null

  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(draft))
  for (const key of QUERY_KEYS) url.searchParams.delete(key)
  const search = url.searchParams.toString()
  window.history.replaceState(null, '', `${url.pathname}${search ? `?${search}` : ''}${url.hash}`)
  return draft
}

export function clearQuickAdd() {
  sessionStorage.removeItem(STORAGE_KEY)
}

export function matchCategoryId(categories: Category[], query: string, fallback: number) {
  const q = query.trim().toLowerCase()
  if (!q) return fallback
  const exact = categories.find(
    (cat) => cat.name.toLowerCase() === q || cat.icon.toLowerCase() === q,
  )
  if (exact) return exact.id
  const partial = categories.find(
    (cat) => cat.name.toLowerCase().includes(q) || cat.icon.toLowerCase().includes(q),
  )
  return partial?.id ?? fallback
}

export function matchMemberId(members: Member[], query: string, fallback: number) {
  const q = query.trim().toLowerCase()
  if (!q) return fallback
  const exact = members.find((member) => member.name.toLowerCase() === q)
  if (exact) return exact.id
  const partial = members.find((member) => member.name.toLowerCase().includes(q))
  return partial?.id ?? fallback
}
