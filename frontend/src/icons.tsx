import type { CategoryIcon } from './types'

const labels: Record<string, string> = {
  food: 'Comida',
  market: 'Mercado',
  transport: 'Transporte',
  home: 'Casa',
  health: 'Saúde',
  leisure: 'Lazer',
  salary: 'Salário',
  freelance: 'Freelancer',
  education: 'Educação',
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
    case 'transport':
      return (
        <svg {...common}>
          <rect x="4" y="6" width="16" height="10" rx="2" />
          <path d="M4 12h16" />
          <circle cx="8" cy="18" r="1.5" />
          <circle cx="16" cy="18" r="1.5" />
        </svg>
      )
    case 'home':
      return (
        <svg {...common}>
          <path d="M4 11.5 12 4l8 7.5" />
          <path d="M6.5 10.5V20h11V10.5" />
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
    case 'education':
      return (
        <svg {...common}>
          <path d="M3 9l9-5 9 5-9 5-9-5Z" />
          <path d="M7 11.5v4.2c0 .8 2.2 2.3 5 2.3s5-1.5 5-2.3v-4.2" />
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
