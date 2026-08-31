import type { Member } from './types'

export type DeviceAccount = {
  memberId: number
  name: string
  savedAt: string
}

const SAVED_KEY = 'fluxo-saved-accounts'
const ACTIVE_KEY = 'fluxo-active-member-id'

export function loadSavedAccounts(): DeviceAccount[] {
  try {
    const raw = localStorage.getItem(SAVED_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as DeviceAccount[]
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (item) =>
        item &&
        typeof item.memberId === 'number' &&
        typeof item.name === 'string' &&
        item.name.trim().length > 0,
    )
  } catch {
    return []
  }
}

function persistSaved(accounts: DeviceAccount[]) {
  localStorage.setItem(SAVED_KEY, JSON.stringify(accounts))
  return accounts
}

export function saveAccount(member: Member): DeviceAccount[] {
  const current = loadSavedAccounts().filter((item) => item.memberId !== member.id)
  current.unshift({
    memberId: member.id,
    name: member.name,
    savedAt: new Date().toISOString(),
  })
  return persistSaved(current)
}

export function removeSavedAccount(memberId: number): DeviceAccount[] {
  return persistSaved(loadSavedAccounts().filter((item) => item.memberId !== memberId))
}

export function syncSavedAccountNames(members: Member[]): DeviceAccount[] {
  const byId = new Map(members.map((member) => [member.id, member]))
  const next = loadSavedAccounts()
    .filter((item) => byId.has(item.memberId))
    .map((item) => ({
      ...item,
      name: byId.get(item.memberId)?.name ?? item.name,
    }))
  return persistSaved(next)
}

export function getActiveMemberId(): number | null {
  const raw = localStorage.getItem(ACTIVE_KEY)
  if (!raw) return null
  const id = Number(raw)
  return Number.isFinite(id) && id > 0 ? id : null
}

export function setActiveMemberId(memberId: number | null) {
  if (memberId == null) localStorage.removeItem(ACTIVE_KEY)
  else localStorage.setItem(ACTIVE_KEY, String(memberId))
}
