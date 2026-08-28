import type { InstallHint } from './pwaInstall'

export function InstallBanner({
  hint,
  onInstall,
  onDismiss,
}: {
  hint: InstallHint
  onInstall: () => void
  onDismiss: () => void
}) {
  if (!hint.visible) return null

  return (
    <aside className="install-banner" aria-label="Instalar o Fluxo">
      <div>
        <strong>Use o Fluxo no celular</strong>
        {hint.isIOS ? (
          <p>
            No Safari, toque em <em>Compartilhar</em> e depois em <em>Adicionar à Tela de Início</em>.
            Abra sempre pelo Safari — o Chrome do iPhone não instala PWA.
          </p>
        ) : hint.canPrompt ? (
          <p>Instale o app para lançar Pix, débito e crédito na hora da compra.</p>
        ) : (
          <p>No menu do navegador, escolha a opção de instalar ou adicionar à tela inicial.</p>
        )}
      </div>
      <div className="install-banner-actions">
        {hint.canPrompt ? (
          <button type="button" className="primary" onClick={onInstall}>
            Instalar
          </button>
        ) : null}
        <button type="button" className="ghost" onClick={onDismiss}>
          Agora não
        </button>
      </div>
    </aside>
  )
}
