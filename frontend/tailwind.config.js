/** @type {import('tailwindcss').Config} */
function withOpacity(varName) {
  return ({ opacityValue }) =>
    opacityValue !== undefined
      ? `rgb(var(${varName}) / ${opacityValue})`
      : `rgb(var(${varName}))`
}

export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        bg:       withOpacity('--c-bg'),
        surface:  withOpacity('--c-surface'),
        surface2: withOpacity('--c-surface-2'),
        sidebar:  withOpacity('--c-sidebar'),
        border:   withOpacity('--c-border'),
        'border-strong': withOpacity('--c-border-strong'),
        text:            withOpacity('--c-text'),
        'text-secondary':withOpacity('--c-text-secondary'),
        'text-muted':    withOpacity('--c-text-muted'),

        accent: {
          50:  withOpacity('--c-accent-50'),
          100: withOpacity('--c-accent-100'),
          200: withOpacity('--c-accent-200'),
          300: withOpacity('--c-accent-300'),
          400: withOpacity('--c-accent-400'),
          500: withOpacity('--c-accent-500'),
          600: withOpacity('--c-accent-600'),
          700: withOpacity('--c-accent-700'),
          800: withOpacity('--c-accent-800'),
          900: withOpacity('--c-accent-900'),
        },
        success: { DEFAULT: withOpacity('--c-success'), bg: withOpacity('--c-success-bg') },
        warning: { DEFAULT: withOpacity('--c-warning'), bg: withOpacity('--c-warning-bg') },
        danger:  { DEFAULT: withOpacity('--c-danger'),  bg: withOpacity('--c-danger-bg') },
        info:    { DEFAULT: withOpacity('--c-info'),    bg: withOpacity('--c-info-bg') },
      },
      fontFamily: {
        sans: ['Iransansfanum', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      fontSize: {
        xs:  ['0.75rem',  { lineHeight: '1.1rem' }],
        sm:  ['0.8125rem',{ lineHeight: '1.25rem' }],
        base:['0.875rem', { lineHeight: '1.4rem' }],
        lg:  ['1rem',     { lineHeight: '1.5rem' }],
        xl:  ['1.125rem', { lineHeight: '1.6rem' }],
      },
      boxShadow: {
        subtle: '0 1px 3px 0 rgb(30 35 80 / 0.06), 0 1px 2px -1px rgb(30 35 80 / 0.05)',
        popover: '0 4px 20px -6px rgb(0 0 0 / 0.18), 0 2px 6px -2px rgb(0 0 0 / 0.08)',
      },
      keyframes: {
        'fade-up': {
          '0%':   { opacity: 0, transform: 'translateY(6px)' },
          '100%': { opacity: 1, transform: 'translateY(0)' },
        },
      },
      animation: {
        'fade-up': 'fade-up .25s cubic-bezier(.16,1,.3,1) both',
      },
    },
  },
  plugins: [],
}
