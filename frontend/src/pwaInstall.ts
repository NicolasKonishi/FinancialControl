const DISMISS_KEY = 'fluxo-install-dismissed'

export type InstallHint = {
  visible: boolean
  isIOS: boolean
  canPrompt: boolean
}

function isStandalone() {
  const nav = window.navigator as Navigator & { standalone?: boolean }
  return window.matchMedia('(display-mode: standalone)').matches || nav.standalone === true
}

function isIOS() {
  const ua = window.navigator.userAgent
  const iOSDevice = /iphone|ipad|ipod/i.test(ua)
  const iPadOs = window.navigator.platform === 'MacIntel' && window.navigator.maxTouchPoints > 1
  return iOSDevice || iPadOs
}

export function markStandalone() {
  if (isStandalone()) {
    document.documentElement.classList.add('standalone')
  }
}

export function createInstallHint(onChange: (hint: InstallHint) => void) {
  let deferred: { prompt: () => Promise<void> } | null = null

  const emit = () => {
    const dismissed = localStorage.getItem(DISMISS_KEY) === '1'
    onChange({
      visible: !isStandalone() && !dismissed,
      isIOS: isIOS(),
      canPrompt: deferred != null,
    })
  }

  const onPrompt = (event: Event) => {
    event.preventDefault()
    const e = event as Event & { prompt: () => Promise<void> }
    deferred = e
    emit()
  }

  window.addEventListener('beforeinstallprompt', onPrompt)
  window.addEventListener('appinstalled', () => {
    localStorage.setItem(DISMISS_KEY, '1')
    deferred = null
    emit()
  })
  emit()

  return {
    prompt: async () => {
      if (!deferred) return
      await deferred.prompt()
      deferred = null
      emit()
    },
    dismiss: () => {
      localStorage.setItem(DISMISS_KEY, '1')
      emit()
    },
    stop: () => window.removeEventListener('beforeinstallprompt', onPrompt),
  }
}
