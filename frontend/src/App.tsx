import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from './api'
import { CategoryGlyph } from './icons'
import { applyTheme, getPreferredTheme, persistTheme, type Theme } from './theme'
import type { Bill, Category, Member, MonthlyForecast, Transaction } from './types'
import './index.css'

type View = 'home' | 'ledger' | 'bills' | 'family'
type SheetMode = 'expense' | 'income' | 'freelance' | 'member' | 'bill' | null

const currency = new Intl.NumberFormat('pt-BR', {
  style: 'currency',
  currency: 'BRL',
})

const monthLabel = new Intl.DateTimeFormat('pt-BR', {
  month: 'long',
  year: 'numeric',
})

function todayISO() {
  return new Date().toISOString().slice(0, 10)
}

function currentMonthKey() {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

function formatDate(value: string) {
  return value.slice(0, 10)
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
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [forecast, setForecast] = useState<MonthlyForecast | null>(null)
  const [memberFilter, setMemberFilter] = useState<number | 'all'>('all')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  const [sheet, setSheet] = useState<SheetMode>(null)
  const [editingTxId, setEditingTxId] = useState<number | null>(null)
  const [editingMemberId, setEditingMemberId] = useState<number | null>(null)
  const [editingBillId, setEditingBillId] = useState<number | null>(null)

  const [txForm, setTxForm] = useState({
    category_id: 0,
    member_id: 0,
    description: '',
    amount: '',
    date: todayISO(),
  })
  const [memberForm, setMemberForm] = useState({ name: '', monthly_salary: '' })
  const [billForm, setBillForm] = useState({
    name: '',
    amount: '',
    category_id: 0,
    member_ids: [] as number[],
    due_day: '10',
    recurrence: 'ongoing' as 'ongoing' | 'until',
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

  const monthBills = useMemo(() => {
    return bills.filter((bill) => {
      const target = `${year}-${String(month).padStart(2, '0')}`
      if (target < bill.start_month) return false
      if (bill.recurrence !== 'ongoing' && bill.end_month && target > bill.end_month) return false
      if (memberFilter === 'all') return true
      return (bill.member_ids ?? []).includes(memberFilter)
    })
  }, [bills, year, month, memberFilter])

  const selectedMemberForecast = useMemo(() => {
    const list = forecast?.by_member ?? []
    if (memberFilter === 'all') return null
    return list.find((item) => item.member_id === memberFilter) ?? null
  }, [forecast, memberFilter])

  async function refresh() {
    setError('')
    setLoading(true)
    try {
      const [cats, mems, txs, billList, fc] = await Promise.all([
        api.listCategories(),
        api.listMembers(),
        api.listTransactions(),
        api.listBills(),
        api.monthlyForecast(year, month),
      ])
      setCategories(cats)
      setMembers(mems)
      setTransactions(txs)
      setBills(billList)
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

  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  useEffect(() => {
    void refresh()
  }, [year, month])

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

  function openSheet(mode: SheetMode) {
    setEditingTxId(null)
    setEditingMemberId(null)
    setEditingBillId(null)
    if (mode === 'member') {
      setMemberForm({ name: '', monthly_salary: '' })
    } else if (mode === 'bill') {
      const home = categories.find((c) => c.icon === 'home') ?? categories[0]
      setBillForm({
        name: '',
        amount: '',
        category_id: home?.id || 0,
        member_ids: members.map((m) => m.id),
        due_day: '10',
        recurrence: 'ongoing',
        start_month: currentMonthKey(),
        end_month: '',
        notes: '',
      })
    } else if (mode === 'freelance') {
      const freelance = categories.find((c) => c.icon === 'freelance') ?? categories[0]
      setTxForm({
        category_id: freelance?.id || 0,
        member_id: members[0]?.id || 0,
        description: 'Freelancer',
        amount: '',
        date: todayISO(),
      })
    } else if (mode === 'income') {
      const salary = categories.find((c) => c.icon === 'salary') ?? categories[0]
      setTxForm({
        category_id: salary?.id || 0,
        member_id: members[0]?.id || 0,
        description: 'Entrada extra',
        amount: '',
        date: todayISO(),
      })
    } else if (mode === 'expense') {
      const market = categories.find((c) => c.icon === 'market' || c.icon === 'food') ?? categories[0]
      setTxForm({
        category_id: market?.id || 0,
        member_id: members[0]?.id || 0,
        description: '',
        amount: '',
        date: todayISO(),
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
    const type = sheet === 'expense' ? 'expense' : 'income'
    const payload = {
      category_id: txForm.category_id,
      member_id: txForm.member_id || null,
      type: type as 'income' | 'expense',
      description: txForm.description,
      amount: Number(txForm.amount),
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

  async function submitBill(event: FormEvent) {
    event.preventDefault()
    if (!billForm.category_id) {
      setError('Selecione uma categoria para a conta.')
      return
    }
    const payload = {
      name: billForm.name,
      amount: Number(billForm.amount),
      category_id: billForm.category_id,
      member_ids: billForm.member_ids,
      due_day: Number(billForm.due_day),
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

  const usedRatio = forecast
    ? Math.min(1, forecast.total_available > 0 ? forecast.projected_expense / forecast.total_available : 0)
    : 0
  const paceClass = usedRatio >= 1 ? 'danger' : usedRatio >= 0.85 ? 'warn' : ''

  return (
    <div className="app">
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

      {error ? <div className="error">{error}</div> : null}
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

      {memberFilter !== 'all' && selectedMemberForecast && (
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

      {memberFilter === 'all' && (forecast?.by_member?.length ?? 0) > 0 && view === 'home' && (
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

      {view === 'home' && (
        <>
          {memberFilter === 'all' && (
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
          )}

          <div className="actions">
            <button type="button" onClick={() => openSheet('expense')}>
              <strong>Saída</strong>
              <span>gasto</span>
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
                          {cat?.name ?? 'Categoria'} · {formatDate(tx.date)}
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
                    <div>
                      <h3>{tx.description}</h3>
                      <p>
                        {cat?.name ?? 'Categoria'}
                        {member ? ` · ${member.name}` : ''} · {formatDate(tx.date)}
                      </p>
                      <div className="row-actions">
                        <button
                          type="button"
                          className="ghost"
                          onClick={() => {
                            setEditingTxId(tx.id)
                            setTxForm({
                              category_id: tx.category_id,
                              member_id: tx.member_id ?? 0,
                              description: tx.description,
                              amount: String(tx.amount),
                              date: formatDate(tx.date),
                            })
                            setSheet(tx.type === 'expense' ? 'expense' : 'income')
                          }}
                        >
                          Editar
                        </button>
                        <button
                          type="button"
                          className="danger"
                          onClick={() => {
                            void (async () => {
                              try {
                                await api.deleteTransaction(tx.id)
                                await refresh()
                              } catch (err) {
                                setError(err instanceof Error ? err.message : 'Erro ao excluir')
                              }
                            })()
                          }}
                        >
                          Excluir
                        </button>
                      </div>
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
          <button type="button" className="primary" onClick={() => openSheet('expense')}>
            Novo lançamento
          </button>
        </section>
      )}

      {view === 'bills' && (
        <section className="card">
          <h2>Contas mensais</h2>
          <p className="meta" style={{ marginBottom: '0.85rem' }}>
            {memberFilter === 'all'
              ? 'Fixas (luz, internet) ou com término (assinatura, crédito).'
              : `Contas em que ${selectedMemberForecast?.member_name ?? 'esta pessoa'} participa.`}
          </p>
          {monthBills.length === 0 ? (
            <p className="empty">Nenhuma conta ativa neste mês.</p>
          ) : (
            <div className="list">
              {monthBills.map((bill) => {
                const cat = categoryById.get(bill.category_id)
                const payers = (bill.member_ids ?? [])
                  .map((id) => memberById.get(id)?.name)
                  .filter(Boolean)
                  .join(', ')
                return (
                  <article key={bill.id} className="row">
                    <div className="icon-wrap">
                      <CategoryGlyph icon={cat?.icon ?? 'home'} />
                    </div>
                    <div>
                      <h3>{bill.name}</h3>
                      <p>
                        dia {bill.due_day} · {cat?.name ?? 'Categoria'} ·{' '}
                        {bill.recurrence === 'ongoing'
                          ? 'sempre'
                          : `até ${bill.end_month}`}
                        {payers ? ` · ${payers}` : ' · sem responsáveis'}
                      </p>
                      <div className="row-actions">
                        <button
                          type="button"
                          className="ghost"
                          onClick={() => {
                            setEditingBillId(bill.id)
                            setBillForm({
                              name: bill.name,
                              amount: String(bill.amount),
                              category_id: bill.category_id,
                              member_ids: bill.member_ids ?? [],
                              due_day: String(bill.due_day),
                              recurrence: bill.recurrence,
                              start_month: bill.start_month,
                              end_month: bill.end_month ?? '',
                              notes: bill.notes ?? '',
                            })
                            setSheet('bill')
                          }}
                        >
                          Editar
                        </button>
                        <button
                          type="button"
                          className="danger"
                          onClick={() => {
                            void (async () => {
                              try {
                                await api.deleteBill(bill.id)
                                await refresh()
                              } catch (err) {
                                setError(err instanceof Error ? err.message : 'Erro ao excluir')
                              }
                            })()
                          }}
                        >
                          Excluir
                        </button>
                      </div>
                    </div>
                    <div className="amount expense">{currency.format(bill.amount)}</div>
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
                  <div>
                    <h3>{member.name}</h3>
                    <p>Salário mensal</p>
                    <div className="row-actions">
                      <button
                        type="button"
                        className="ghost"
                        onClick={() => {
                          setEditingMemberId(member.id)
                          setMemberForm({
                            name: member.name,
                            monthly_salary: String(member.monthly_salary),
                          })
                          setSheet('member')
                        }}
                      >
                        Editar
                      </button>
                      <button
                        type="button"
                        className="danger"
                        onClick={() => {
                          void (async () => {
                            try {
                              await api.deleteMember(member.id)
                              await refresh()
                            } catch (err) {
                              setError(err instanceof Error ? err.message : 'Erro ao excluir')
                            }
                          })()
                        }}
                      >
                        Excluir
                      </button>
                    </div>
                  </div>
                  <div className="amount income">{currency.format(member.monthly_salary)}</div>
                </article>
              ))}
            </div>
          )}
          <button type="button" className="primary" onClick={() => openSheet('member')}>
            Adicionar pessoa
          </button>
        </section>
      )}

      <nav className="bottom-nav" aria-label="Navegação">
        <button type="button" className={view === 'home' ? 'active' : ''} onClick={() => setView('home')}>
          Início
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
            ) : sheet === 'bill' ? (
              <form className="form" onSubmit={submitBill}>
                <label>
                  Nome da conta
                  <input
                    required
                    value={billForm.name}
                    onChange={(e) => setBillForm((p) => ({ ...p, name: e.target.value }))}
                    placeholder="Internet, Luz, Netflix…"
                  />
                </label>
                <label>
                  Valor mensal
                  <input
                    required
                    type="number"
                    min="0.01"
                    step="0.01"
                    value={billForm.amount}
                    onChange={(e) => setBillForm((p) => ({ ...p, amount: e.target.value }))}
                  />
                </label>
                <label>
                  Tipo
                  <select
                    value={billForm.recurrence}
                    onChange={(e) =>
                      setBillForm((p) => ({
                        ...p,
                        recurrence: e.target.value as 'ongoing' | 'until',
                      }))
                    }
                  >
                    <option value="ongoing">Sempre (luz, internet…)</option>
                    <option value="until">Com término (assinatura, crédito…)</option>
                  </select>
                </label>
                <label>
                  Dia do vencimento
                  <input
                    required
                    type="number"
                    min="1"
                    max="31"
                    value={billForm.due_day}
                    onChange={(e) => setBillForm((p) => ({ ...p, due_day: e.target.value }))}
                  />
                </label>
                <label>
                  Começa em
                  <input
                    required
                    type="month"
                    value={billForm.start_month}
                    onChange={(e) => setBillForm((p) => ({ ...p, start_month: e.target.value }))}
                  />
                </label>
                {billForm.recurrence === 'until' ? (
                  <label>
                    Termina em
                    <input
                      required
                      type="month"
                      value={billForm.end_month}
                      onChange={(e) => setBillForm((p) => ({ ...p, end_month: e.target.value }))}
                    />
                  </label>
                ) : null}
                <label>
                  Categoria
                  <select
                    required
                    value={billForm.category_id || ''}
                    onChange={(e) => setBillForm((p) => ({ ...p, category_id: Number(e.target.value) }))}
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
                  <div className="check-list">
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
                  Observação
                  <input
                    value={billForm.notes}
                    onChange={(e) => setBillForm((p) => ({ ...p, notes: e.target.value }))}
                    placeholder="Opcional"
                  />
                </label>
                <button className="primary" type="submit">
                  Salvar
                </button>
              </form>
            ) : (
              <form className="form" onSubmit={submitTransaction}>
                <label>
                  Descrição
                  <input
                    required
                    value={txForm.description}
                    onChange={(e) => setTxForm((p) => ({ ...p, description: e.target.value }))}
                    placeholder={sheet === 'expense' ? 'Mercado, almoço…' : 'Pagamento, freelance…'}
                  />
                </label>
                <label>
                  Valor
                  <input
                    required
                    type="number"
                    min="0.01"
                    step="0.01"
                    value={txForm.amount}
                    onChange={(e) => setTxForm((p) => ({ ...p, amount: e.target.value }))}
                  />
                </label>
                <label>
                  Categoria
                  <select
                    required
                    value={txForm.category_id || ''}
                    onChange={(e) => setTxForm((p) => ({ ...p, category_id: Number(e.target.value) }))}
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
                    onChange={(e) => setTxForm((p) => ({ ...p, member_id: Number(e.target.value) }))}
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
                  Data
                  <input
                    required
                    type="date"
                    value={txForm.date}
                    onChange={(e) => setTxForm((p) => ({ ...p, date: e.target.value }))}
                  />
                </label>
                <button className="primary" type="submit">
                  Salvar
                </button>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
