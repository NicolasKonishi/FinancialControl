import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { launchShortcutUrl, nowLocal } from './quickAdd'

export function QuickLaunch({
  onSave,
  onClose,
  saving,
  error,
}: {
  onSave: (place: string, amount: string) => Promise<boolean>
  onClose: () => void
  saving: boolean
  error: string
}) {
  const [place, setPlace] = useState('')
  const [amount, setAmount] = useState('')
  const [clock, setClock] = useState(() => nowLocal())
  const [copied, setCopied] = useState(false)
  const [justSaved, setJustSaved] = useState('')
  const placeRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    placeRef.current?.focus()
    const tick = window.setInterval(() => setClock(nowLocal()), 15_000)
    return () => window.clearInterval(tick)
  }, [])

  async function submit(event: FormEvent) {
    event.preventDefault()
    setClock(nowLocal())
    const ok = await onSave(place, amount)
    if (!ok) return
    setJustSaved(`${place.trim()} · ${amount.trim()}`)
    setPlace('')
    setAmount('')
    placeRef.current?.focus()
  }

  async function copyShortcut() {
    try {
      await navigator.clipboard.writeText(launchShortcutUrl())
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      setCopied(false)
    }
  }

  return (
    <div className="quick-launch" role="dialog" aria-modal="true" aria-labelledby="quick-launch-title">
      <div className="quick-launch-panel">
        <header>
          <div>
            <h2 id="quick-launch-title">Lançar no Fluxo</h2>
            <p className="quick-launch-when">{clock.label}</p>
          </div>
          <button type="button" className="ghost" onClick={onClose}>
            Fechar
          </button>
        </header>

        <form className="quick-launch-form" onSubmit={(event) => void submit(event)}>
          <label>
            Lugar
            <input
              ref={placeRef}
              required
              value={place}
              onChange={(e) => setPlace(e.target.value)}
              placeholder="Padaria, mercado, Uber…"
              autoComplete="off"
              enterKeyHint="next"
            />
          </label>
          <label>
            Valor
            <input
              required
              className="quick-launch-amount"
              type="text"
              inputMode="decimal"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="0,00"
              enterKeyHint="done"
            />
          </label>
          <p className="meta">
            A data, a hora e o mês entram sozinhos no momento em que você salvar.
          </p>
          {error ? <div className="error">{error}</div> : null}
          {justSaved ? <p className="quick-launch-ok">Salvo: {justSaved}</p> : null}
          <button className="primary" type="submit" disabled={saving}>
            {saving ? 'Salvando…' : 'Salvar saída'}
          </button>
        </form>

        <div className="quick-launch-hint">
          <p>
            No iPhone, abra <em>Atalhos</em> → novo atalho → <em>Abrir URLs</em> e cole o link.
            Ou, com esta tela aberta, use <em>Compartilhar → Adicionar à Tela de Início</em>.
          </p>
          <button type="button" className="ghost" onClick={() => void copyShortcut()}>
            {copied ? 'Link copiado' : 'Copiar link do atalho'}
          </button>
        </div>
      </div>
    </div>
  )
}
