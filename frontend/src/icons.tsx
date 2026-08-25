import type { CategoryIcon } from './types'

export const CATEGORY_ICONS = [
  'food',
  'market',
  'cafe',
  'transport',
  'car',
  'home',
  'utilities',
  'health',
  'leisure',
  'travel',
  'shopping',
  'clothing',
  'pets',
  'kids',
  'education',
  'phone',
  'subscriptions',
  'gift',
  'salary',
  'freelance',
  'investment',
  'other',
] as const satisfies readonly CategoryIcon[]

const labels: Record<string, string> = {
  food: 'Comida',
  market: 'Mercado',
  cafe: 'Café / lanche',
  transport: 'Transporte',
  car: 'Carro',
  home: 'Casa',
  utilities: 'Contas casa',
  health: 'Saúde',
  leisure: 'Lazer',
  travel: 'Viagem',
  shopping: 'Compras',
  clothing: 'Roupas',
  pets: 'Pets',
  kids: 'Filhos',
  education: 'Educação',
  phone: 'Celular',
  subscriptions: 'Assinaturas',
  gift: 'Presente',
  salary: 'Salário',
  freelance: 'Freelancer',
  investment: 'Investimento',
  other: 'Outro',
}

export function iconLabel(icon: string) {
  return labels[icon] ?? icon
}

/** Minimal inline SVG icons for categories (mobile-friendly). */
export function CategoryGlyph({ icon, className = '' }: { icon: CategoryIcon | string; className?: string }) {
  const common = {
    viewBox: '0 0 24 24',
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 1.8,
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
    className: `glyph ${className}`,
    'aria-hidden': true,
  }

  switch (icon) {
    case 'food':
      return (
        <svg {...common}>
          <path d="M4 10h16" />
          <path d="M6 10v8a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-8" />
          <path d="M8 10V7a4 4 0 0 1 8 0v3" />
        </svg>
      )
    case 'market':
      return (
        <svg {...common}>
          <path d="M3 7h18l-1.5 12a2 2 0 0 1-2 1.8H6.5a2 2 0 0 1-2-1.8L3 7Z" />
          <path d="M8 7V5a4 4 0 0 1 8 0v2" />
        </svg>
      )
    case 'cafe':
      return (
        <svg {...common}>
          <path d="M5 9h11v6a3 3 0 0 1-3 3H8a3 3 0 0 1-3-3V9Z" />
          <path d="M16 10h2a2.5 2.5 0 0 1 0 5h-2" />
          <path d="M6 20h10" />
        </svg>
      )
    case 'transport':
      return (
        <svg {...common}>
          <rect x="4" y="6" width="16" height="10" rx="2" />
          <path d="M4 12h16" />
          <circle cx="8" cy="18" r="1.5" />
          <circle cx="16" cy="18" r="1.5" />
        </svg>
      )
    case 'car':
      return (
        <svg {...common}>
          <path d="M4 14 6.5 8.5A2 2 0 0 1 8.3 7.5h7.4a2 2 0 0 1 1.8 1L20 14" />
          <path d="M3 14h18v3.5a1.5 1.5 0 0 1-1.5 1.5H4.5A1.5 1.5 0 0 1 3 17.5V14Z" />
          <circle cx="7.5" cy="17" r="1.2" />
          <circle cx="16.5" cy="17" r="1.2" />
        </svg>
      )
    case 'home':
      return (
        <svg {...common}>
          <path d="M4 11.5 12 4l8 7.5" />
          <path d="M6.5 10.5V20h11V10.5" />
        </svg>
      )
    case 'utilities':
      return (
        <svg {...common}>
          <path d="M13 2 5 13h6l-1 9 9-12h-6l0-8Z" />
        </svg>
      )
    case 'health':
      return (
        <svg {...common}>
          <path d="M12 21s-7-4.4-7-10a4 4 0 0 1 7-2.5A4 4 0 0 1 19 11c0 5.6-7 10-7 10Z" />
        </svg>
      )
    case 'leisure':
      return (
        <svg {...common}>
          <circle cx="12" cy="12" r="8" />
          <path d="M12 8v4l2.5 2.5" />
        </svg>
      )
    case 'travel':
      return (
        <svg {...common}>
          <path d="M10 20 12 4l2 16" />
          <path d="M5 11h14" />
          <path d="M7 16h10" />
          <path d="M8.5 7.5 12 4l3.5 3.5" />
        </svg>
      )
    case 'shopping':
      return (
        <svg {...common}>
          <path d="M7 8h14l-1.5 10H8.5L7 8Z" />
          <path d="M7 8 5.5 3H3" />
          <circle cx="10" cy="20" r="1.2" />
          <circle cx="17" cy="20" r="1.2" />
        </svg>
      )
    case 'clothing':
      return (
        <svg {...common}>
          <path d="M9 4 12 6.5 15 4l3 2.5-2 3V20H8V9.5l-2-3L9 4Z" />
        </svg>
      )
    case 'pets':
      return (
        <svg {...common}>
          <ellipse cx="12" cy="15" rx="5" ry="4" />
          <circle cx="7" cy="9" r="1.6" />
          <circle cx="17" cy="9" r="1.6" />
          <circle cx="9.5" cy="6.5" r="1.4" />
          <circle cx="14.5" cy="6.5" r="1.4" />
        </svg>
      )
    case 'kids':
      return (
        <svg {...common}>
          <circle cx="12" cy="7" r="3" />
          <path d="M6 20v-1a5 5 0 0 1 10 0v1" />
          <path d="M8 12.5c1.2-1 2.5-1.5 4-1.5s2.8.5 4 1.5" />
        </svg>
      )
    case 'education':
      return (
        <svg {...common}>
          <path d="M3 9l9-5 9 5-9 5-9-5Z" />
          <path d="M7 11.5v4.2c0 .8 2.2 2.3 5 2.3s5-1.5 5-2.3v-4.2" />
        </svg>
      )
    case 'phone':
      return (
        <svg {...common}>
          <rect x="7" y="3" width="10" height="18" rx="2" />
          <path d="M11 18h2" />
        </svg>
      )
    case 'subscriptions':
      return (
        <svg {...common}>
          <rect x="3" y="6" width="18" height="13" rx="2" />
          <path d="M8 6V4.5A1.5 1.5 0 0 1 9.5 3h5A1.5 1.5 0 0 1 16 4.5V6" />
          <path d="M8 12h8" />
          <path d="M8 15h5" />
        </svg>
      )
    case 'gift':
      return (
        <svg {...common}>
          <rect x="4" y="10" width="16" height="10" rx="1.5" />
          <path d="M12 10v10" />
          <path d="M4 14h16" />
          <path d="M12 10c-2.5 0-4-1.6-4-3.2S9.5 4 12 6c2.5-2 4-.6 4 .8S14.5 10 12 10Z" />
        </svg>
      )
    case 'salary':
      return (
        <svg {...common}>
          <rect x="3" y="6" width="18" height="12" rx="2" />
          <path d="M3 10h18" />
          <path d="M12 14h.01" />
        </svg>
      )
    case 'freelance':
      return (
        <svg {...common}>
          <path d="M4 17 17 4l3 3L7 20H4v-3Z" />
          <path d="M14 7l3 3" />
        </svg>
      )
    case 'investment':
      return (
        <svg {...common}>
          <path d="M4 18V6" />
          <path d="M4 18h16" />
          <path d="M8 14l3-3 3 2 5-6" />
        </svg>
      )
    default:
      return (
        <svg {...common}>
          <circle cx="12" cy="12" r="8" />
          <path d="M12 8v5" />
          <path d="M12 16h.01" />
        </svg>
      )
  }
}

const actionIconProps = {
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.8,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  className: 'action-glyph',
  'aria-hidden': true,
}

export function PencilIcon() {
  return (
    <svg {...actionIconProps}>
      <path d="M12 20h9" />
      <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5Z" />
    </svg>
  )
}

export function TrashIcon() {
  return (
    <svg {...actionIconProps}>
      <path d="M3 6h18" />
      <path d="M8 6V4.5A1.5 1.5 0 0 1 9.5 3h5A1.5 1.5 0 0 1 16 4.5V6" />
      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
      <path d="M10 11v6" />
      <path d="M14 11v6" />
    </svg>
  )
}
