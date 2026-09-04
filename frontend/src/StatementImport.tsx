import { useMemo, useRef, useState } from 'react'
import { api } from './api'
import { CREDIT_ISSUERS } from './creditCards'
import type { Category, Member, StatementPreviewItem, Wallet } from './types'

const currency = new Intl.NumberFormat('pt-BR', {
  style: 'currency',
  currency: 'BRL',
})

const dateLabel = new Intl.DateTimeFormat('pt-BR', {
  day: '2-digit',
  month: 'short',
})

export type StatementImportMode = 'credit' | 'debit'

type Props = {
  mode: StatementImportMode
  categories: Category[]
  members: Member[]
  wallets: Wallet[]
  defaultMemberId: number
  defaultWalletId: number
  year: number
  month: number
  onImported: () => Promise<void>
}

function isCredit(wallet: Wallet) {
  return wallet.kind === 'credit'
}

function walletRank(wallet: Wallet) {
  if (wallet.kind === 'checking') return 0
  if (wallet.kind === 'company') return 1
  if (wallet.kind === 'credit') return 2
  return 3
}

function issuerLabel(issuer: string) {
  if (!issuer || issuer === 'unknown') return ''
  return CREDIT_ISSUERS.find((item) => item.id === issuer)?.label ?? issuer
}

function formatDay(value: string) {
  const parsed = new Date(`${value.slice(0, 10)}T12:00:00`)
  if (Number.isNaN(parsed.getTime())) return value.slice(0, 10)
  return dateLabel.format(parsed)
}

function isLocked(item: StatementPreviewItem) {
  return item.kind === 'payment' || item.kind === 'transfer'
}

function isImportable(item: StatementPreviewItem) {
  return item.kind === 'expense' || item.kind === 'income' || item.kind === 'refund'
}

function kindNote(item: StatementPreviewItem) {
  if (item.already_recorded) return 'já lançado'
  if (item.matched_bill_id) return 'já na lista de contas'
  switch (item.kind) {
    case 'payment':
      return 'pagamento da fatura'
    case 'transfer':
      return 'aplicação / transferência interna'
    case 'refund':
      return 'estorno'
    case 'income':
      return 'entrada'
    default:
      return ''
  }
}

function importType(item: StatementPreviewItem): 'income' | 'expense' {
  return item.kind === 'income' || item.kind === 'refund' ? 'income' : 'expense'
}

function showImportAlert(message: string) {
  window.alert(message)
}

export function StatementImport({
  mode,
  categories,
  members,
  wallets,
  defaultMemberId,
  defaultWalletId,
  year,
  month,
  onImported,
}: Props) {
  const fileRef = useRef<HTMLInputElement | null>(null)
  const creditMode = mode === 'credit'
  const expectedType = creditMode ? 'credit_card' : 'account'
  const [memberId, setMemberId] = useState(defaultMemberId)
  const [walletId, setWalletId] = useState(defaultWalletId)
  const [file, setFile] = useState<File | null>(null)
  const [applyToInvoice, setApplyToInvoice] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [items, setItems] = useState<StatementPreviewItem[] | null>(null)
  const [issuer, setIssuer] = useState('')
  const [period, setPeriod] = useState('')
  const [statementType, setStatementType] = useState('')
  const [statementBalance, setStatementBalance] = useState<number | null>(null)
  const [invoiceYear, setInvoiceYear] = useState<number | null>(null)
  const [invoiceMonth, setInvoiceMonth] = useState<number | null>(null)
  const [periodStart, setPeriodStart] = useState<string | null>(null)
  const [periodEnd, setPeriodEnd] = useState<string | null>(null)
  const [dueDate, setDueDate] = useState<string | null>(null)
  const [fileSha, setFileSha] = useState('')

  const memberWallets = useMemo(() => {
    return wallets
      .filter((wallet) => (creditMode ? isCredit(wallet) : !isCredit(wallet) && wallet.kind !== 'savings'))
      .filter((wallet) => wallet.member_id == null || wallet.member_id === memberId || wallet.kind === 'company')
      .sort((a, b) => walletRank(a) - walletRank(b) || a.name.localeCompare(b.name))
  }, [wallets, memberId, creditMode])

  const selectedWallet = memberWallets.find((wallet) => wallet.id === walletId) ?? memberWallets[0] ?? null
  const selectedWalletId = selectedWallet?.id ?? 0
  const selected = (items ?? []).filter((item) => item.selected && isImportable(item))
  const selectedExpenses = selected.filter((item) => importType(item) === 'expense')
  const selectedIncome = selected.filter((item) => importType(item) === 'income')
  const selectedExpenseTotal = selectedExpenses.reduce((sum, item) => sum + item.amount, 0)
  const selectedIncomeTotal = selectedIncome.reduce((sum, item) => sum + item.amount, 0)
  const statementExpenseTotal = (items ?? [])
    .filter((item) => isImportable(item) && importType(item) === 'expense')
    .reduce((sum, item) => sum + item.amount, 0)
  const statementIncomeTotal = (items ?? [])
    .filter((item) => isImportable(item) && importType(item) === 'income')
    .reduce((sum, item) => sum + item.amount, 0)
  const newCount = (items ?? []).filter((item) => isImportable(item) && !item.already_recorded).length
  const matchedCount = (items ?? []).filter((item) => item.already_recorded).length
  const competenceYear = invoiceYear ?? year
  const competenceMonth = invoiceMonth ?? month

  function walletsForMember(nextMember: number) {
    return wallets
      .filter((wallet) => (creditMode ? isCredit(wallet) : !isCredit(wallet) && wallet.kind !== 'savings'))
      .filter(
        (wallet) => wallet.member_id == null || wallet.member_id === nextMember || wallet.kind === 'company',
      )
      .sort((a, b) => walletRank(a) - walletRank(b) || a.name.localeCompare(b.name))
  }

  async function readFile() {
    if (!file) {
      setError('Escolha o CSV, OFX ou PDF do extrato.')
      return
    }
    if (!selectedWalletId) {
      setError(creditMode ? 'Escolha o cartão desta fatura.' : 'Escolha a conta deste extrato.')
      return
    }
    setBusy(true)
    setError('')
    try {
      const preview = await api.previewStatement(file, {
        wallet_id: selectedWalletId,
        member_id: memberId || null,
        year,
        month,
        expected_type: expectedType,
      })
      setItems(preview.items)
      setIssuer(preview.issuer)
      setStatementType(preview.statement_type)
      setStatementBalance(preview.balance ?? null)
      setInvoiceYear(preview.invoice_year ?? preview.import_year ?? null)
      setInvoiceMonth(preview.invoice_month ?? preview.import_month ?? null)
      setPeriodStart(preview.period_start ?? null)
      setPeriodEnd(preview.period_end ?? null)
      setDueDate(preview.due_date ?? null)
      setFileSha(preview.file_sha256 ?? '')
      if (preview.wallet_id) setWalletId(preview.wallet_id)
      const start = preview.period_start ? formatDay(preview.period_start) : ''
      const end = preview.period_end ? formatDay(preview.period_end) : ''
      setPeriod(start && end ? `${start} – ${end}` : start || end)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Não consegui ler o extrato.'
      setItems(null)
      setError(message)
      showImportAlert(message)
    } finally {
      setBusy(false)
    }
  }

  async function importSelected() {
    if (selected.length === 0) {
      setError('Marque pelo menos um lançamento para importar.')
      return
    }
    if (!selectedWalletId) {
      setError(creditMode ? 'Escolha o cartão desta fatura.' : 'Escolha a conta deste extrato.')
      return
    }
    setBusy(true)
    setError('')
    try {
      await api.importStatement({
        wallet_id: selectedWalletId,
        member_id: memberId || null,
        apply_to_invoice: statementType === 'credit_card' || Boolean(selectedWallet && isCredit(selectedWallet) && applyToInvoice),
        statement_type: statementType || expectedType,
        invoice_year: invoiceYear,
        invoice_month: invoiceMonth,
        statement_balance:
          statementType === 'credit_card'
            ? (statementBalance ?? Math.max(0, statementExpenseTotal - statementIncomeTotal))
            : statementBalance,
        period_start: periodStart,
        period_end: periodEnd,
        file_sha256: fileSha,
        file_name: file?.name ?? '',
        items: selected.map((item) => ({
          date: item.date.slice(0, 10),
          description: item.description.trim(),
          amount: item.amount,
          type: importType(item),
          category_id: item.category_id,
        })),
      })
      await onImported()
      setItems(null)
      setFile(null)
      setError('')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Erro ao lançar o extrato.'
      setError(message)
      showImportAlert(message)
    } finally {
      setBusy(false)
    }
  }

  function updateItem(index: number, patch: Partial<StatementPreviewItem>) {
    setItems((prev) => prev?.map((item) => (item.index === index ? { ...item, ...patch } : item)) ?? null)
  }

  function toggleNew(select: boolean) {
    setItems(
      (prev) =>
        prev?.map((item) =>
          isImportable(item) && !item.already_recorded ? { ...item, selected: select } : item,
        ) ?? null,
    )
  }

  return (
    <div className="form statement-import">
      {error ? <div className="error">{error}</div> : null}

      {items == null ? (
        <>
          <p className="meta" style={{ margin: 0 }}>
            {creditMode
              ? 'Manda o PDF ou o OFX da fatura. O Fluxo lê as compras, casa com as contas que você já cadastrou e atualiza a fatura do mês do documento.'
              : 'Manda o CSV, o OFX ou o PDF da conta. O Fluxo lança os gastos, ignora o que já está no extrato e recusa o arquivo se este mês já foi importado.'}
          </p>
          <label>
            Quem movimentou
            <select
              value={memberId || ''}
              onChange={(e) => {
                const next = Number(e.target.value)
                setMemberId(next)
                const options = walletsForMember(next)
                if (!options.some((wallet) => wallet.id === walletId)) {
                  setWalletId(options[0]?.id ?? 0)
                }
              }}
            >
              {members.map((member) => (
                <option key={member.id} value={member.id}>
                  {member.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            {creditMode ? 'Cartão' : 'Conta'}
            <select value={selectedWalletId || ''} onChange={(e) => setWalletId(Number(e.target.value))}>
              {memberWallets.length === 0 ? (
                <option value="">{creditMode ? 'Cadastre um cartão em Configuração' : 'Cadastre uma conta em Configuração'}</option>
              ) : (
                memberWallets.map((wallet) => (
                  <option key={wallet.id} value={wallet.id}>
                    {wallet.name}
                    {isCredit(wallet) ? ' · cartão' : wallet.kind === 'checking' ? ' · Pix' : ''}
                  </option>
                ))
              )}
            </select>
          </label>
          <div>
            <span className="field-label">Arquivo</span>
            <input
              ref={fileRef}
              className="statement-file-input"
              type="file"
              accept="application/pdf,.pdf,text/csv,.csv,application/x-ofx,.ofx,.qfx"
              onChange={(e) => {
                setFile(e.target.files?.[0] ?? null)
                setError('')
              }}
            />
            <button type="button" className="ghost statement-file-btn" onClick={() => fileRef.current?.click()}>
              {file ? file.name : creditMode ? 'Escolher PDF ou OFX da fatura' : 'Escolher CSV, OFX ou PDF'}
            </button>
          </div>
          <button className="primary" type="button" disabled={busy} onClick={() => void readFile()}>
            {busy ? 'Lendo extrato…' : 'Ler extrato'}
          </button>
        </>
      ) : (
        <>
          <p className="meta" style={{ margin: 0 }}>
            {issuerLabel(issuer) ? `${issuerLabel(issuer)} · ` : ''}
            {period || 'movimentações encontradas'}
            {` · ${creditMode ? 'fatura' : 'extrato'} ${String(competenceMonth).padStart(2, '0')}/${competenceYear}`}
            {statementType === 'credit_card' && dueDate ? ` · vence ${formatDay(dueDate)}` : ''}
            {statementType === 'credit_card' && statementBalance != null
              ? ` · ${currency.format(statementBalance)}`
              : ''}
            {`. ${newCount} novas`}
            {matchedCount ? ` · ${matchedCount} já no Fluxo` : ''}
          </p>
          <div className="statement-toolbar">
            <button type="button" className="ghost" onClick={() => toggleNew(true)}>
              Marcar novas
            </button>
            <button type="button" className="ghost" onClick={() => toggleNew(false)}>
              Desmarcar
            </button>
            <button
              type="button"
              className="ghost"
              onClick={() => {
                setItems(null)
                setError('')
              }}
            >
              Outro arquivo
            </button>
          </div>
          <div className="statement-list">
            {items.map((item) => {
              const locked = isLocked(item)
              const note = kindNote(item)
              return (
                <article
                  key={item.index}
                  className={`statement-item${item.already_recorded ? ' is-matched' : ''}${locked ? ' is-skipped' : ''}`}
                >
                  <label className="statement-check">
                    <input
                      type="checkbox"
                      checked={item.selected}
                      disabled={locked || busy}
                      onChange={(e) => updateItem(item.index, { selected: e.target.checked })}
                    />
                    <span className="statement-item-main">
                      <strong>{item.description}</strong>
                      <span>
                        {formatDay(item.date)}
                        {note ? ` · ${note}` : ''}
                      </span>
                    </span>
                  </label>
                  <div className="statement-item-side">
                    <div className={`amount ${importType(item) === 'income' ? 'income' : 'expense'}`}>
                      {importType(item) === 'income' ? '+' : '−'}
                      {currency.format(item.amount)}
                    </div>
                    {locked ? null : (
                      <select
                        value={item.category_id || ''}
                        disabled={busy}
                        onChange={(e) => updateItem(item.index, { category_id: Number(e.target.value) })}
                      >
                        {categories.map((cat) => (
                          <option key={cat.id} value={cat.id}>
                            {cat.name}
                          </option>
                        ))}
                      </select>
                    )}
                  </div>
                </article>
              )
            })}
          </div>
          {selectedWallet && isCredit(selectedWallet) && statementType !== 'credit_card' ? (
            <label className="check-item statement-invoice-opt">
              <input
                type="checkbox"
                checked={applyToInvoice}
                onChange={(e) => setApplyToInvoice(e.target.checked)}
              />
              <span>Somar estes gastos na fatura do {selectedWallet.name}</span>
            </label>
          ) : null}
          {selectedWallet && isCredit(selectedWallet) && statementType === 'credit_card' ? (
            <p className="meta statement-invoice-opt" style={{ margin: 0 }}>
              Esta importação atualiza a fatura de {String(invoiceMonth ?? competenceMonth).padStart(2, '0')}/
              {invoiceYear ?? competenceYear} do {selectedWallet.name}.
            </p>
          ) : null}
          <p className="meta" style={{ margin: 0 }}>
            {selected.length} selecionado{selected.length === 1 ? '' : 's'}
            {selectedExpenses.length ? ` · saídas ${currency.format(selectedExpenseTotal)}` : ''}
            {selectedIncome.length ? ` · entradas ${currency.format(selectedIncomeTotal)}` : ''}
          </p>
          <button className="primary" type="button" disabled={busy || selected.length === 0} onClick={() => void importSelected()}>
            {busy ? 'Lançando…' : `Lançar ${selected.length} item${selected.length === 1 ? '' : 's'}`}
          </button>
        </>
      )}
    </div>
  )
}
