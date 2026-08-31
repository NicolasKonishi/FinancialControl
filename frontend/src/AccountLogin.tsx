import { useState } from 'react'
import type { FormEvent } from 'react'
import type { DeviceAccount } from './deviceAccounts'
import type { Member } from './types'
import { CategoryGlyph } from './icons'

export function AccountLogin({
  members,
  savedAccounts,
  onLogin,
  onCreateFirst,
}: {
  members: Member[]
  savedAccounts: DeviceAccount[]
  onLogin: (member: Member, remember: boolean) => void
  onCreateFirst: (name: string) => Promise<void>
}) {
  const [remember, setRemember] = useState(true)
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const savedIds = new Set(savedAccounts.map((item) => item.memberId))
  const otherMembers = members.filter((member) => !savedIds.has(member.id))

  async function submitCreate(event: FormEvent) {
    event.preventDefault()
    const name = newName.trim()
    if (!name) {
      setError('Informe o nome.')
      return
    }
    setBusy(true)
    setError('')
    try {
      await onCreateFirst(name)
      setNewName('')
      setCreating(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao criar pessoa')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="account-login">
      <div className="account-login-panel">
        <div className="account-login-brand">
          <CategoryGlyph icon="home" />
          <h1>Fluxo</h1>
          <p>Entre como um membro da família neste dispositivo.</p>
        </div>

        {error ? <div className="error">{error}</div> : null}

        {savedAccounts.length > 0 ? (
          <section className="account-login-section">
            <h2>Contas neste dispositivo</h2>
            <div className="account-login-list">
              {savedAccounts.map((account) => {
                const member = members.find((item) => item.id === account.memberId)
                if (!member) return null
                return (
                  <button
                    key={account.memberId}
                    type="button"
                    className="account-login-row"
                    onClick={() => onLogin(member, true)}
                  >
                    <span className="account-login-avatar" aria-hidden>
                      {member.name.slice(0, 1).toUpperCase()}
                    </span>
                    <span className="account-login-row-main">
                      <strong>{member.name}</strong>
                      <span>conta salva</span>
                    </span>
                  </button>
                )
              })}
            </div>
          </section>
        ) : null}

        {otherMembers.length > 0 ? (
          <section className="account-login-section">
            <h2>{savedAccounts.length > 0 ? 'Outras pessoas' : 'Quem está usando?'}</h2>
            <div className="account-login-list">
              {otherMembers.map((member) => (
                <button
                  key={member.id}
                  type="button"
                  className="account-login-row"
                  onClick={() => onLogin(member, remember)}
                >
                  <span className="account-login-avatar" aria-hidden>
                    {member.name.slice(0, 1).toUpperCase()}
                  </span>
                  <span className="account-login-row-main">
                    <strong>{member.name}</strong>
                    <span>membro da família</span>
                  </span>
                </button>
              ))}
            </div>
            <label className="account-login-remember">
              <input
                type="checkbox"
                checked={remember}
                onChange={(e) => setRemember(e.target.checked)}
              />
              <span>Salvar neste dispositivo</span>
            </label>
          </section>
        ) : null}

        {members.length === 0 || creating ? (
          <section className="account-login-section">
            <h2>{members.length === 0 ? 'Crie a primeira pessoa' : 'Nova pessoa'}</h2>
            <form className="form" onSubmit={(event) => void submitCreate(event)}>
              <label>
                Nome
                <input
                  required
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="Seu nome"
                  autoComplete="name"
                />
              </label>
              <button className="primary" type="submit" disabled={busy}>
                {busy ? 'Salvando…' : 'Criar e entrar'}
              </button>
              {members.length > 0 ? (
                <button type="button" className="ghost" onClick={() => setCreating(false)}>
                  Cancelar
                </button>
              ) : null}
            </form>
          </section>
        ) : (
          <button type="button" className="ghost account-login-add" onClick={() => setCreating(true)}>
            Cadastrar outra pessoa
          </button>
        )}
      </div>
    </div>
  )
}
