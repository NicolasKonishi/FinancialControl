/** Credit cards / wallets used to tag expenses. */
export const CREDIT_CARDS = [
  { id: '', label: 'Sem cartão (à vista)' },
  { id: 'pix', label: 'Pix' },
  { id: 'debito', label: 'Débito' },
  { id: 'nubank', label: 'Nubank' },
  { id: 'bb', label: 'Banco do Brasil' },
  { id: 'mercado-livre', label: 'Mercado Livre' },
  { id: 'itau', label: 'Itaú' },
  { id: 'bradesco', label: 'Bradesco' },
  { id: 'santander', label: 'Santander' },
  { id: 'inter', label: 'Inter' },
  { id: 'c6', label: 'C6 Bank' },
  { id: 'picpay', label: 'PicPay' },
  { id: 'caixa', label: 'Caixa' },
  { id: 'will', label: 'Will Bank' },
  { id: 'neon', label: 'Neon' },
] as const

export type CreditCardId = (typeof CREDIT_CARDS)[number]['id']

export function creditCardLabel(id: string) {
  return CREDIT_CARDS.find((card) => card.id === id)?.label ?? ''
}

/** Infer card from a saved description (exact match on label). */
export function creditCardFromDescription(description: string): CreditCardId {
  const trimmed = description.trim().toLowerCase()
  const match = CREDIT_CARDS.find(
    (card) => card.id && card.label.toLowerCase() === trimmed,
  )
  return (match?.id ?? '') as CreditCardId
}
