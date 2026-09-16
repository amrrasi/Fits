/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50:  '#f0f4ff',
          100: '#e0e9ff',
          200: '#c7d7fe',
          300: '#a5bcfd',
          400: '#8098fa',
          500: '#6374f5',
          600: '#4f52ea',
          700: '#4240d0',
          800: '#3636a8',
          900: '#313285',
          950: '#1e1c50',
        },
        aurora: {
          400: '#5eead4',
          500: '#2dd4bf',
          600: '#0d9488',
        },
        space: {
          850: '#141726',
          900: '#0e1120',
          950: '#090b15',
        },
      },
      fontFamily: {
        sans: ['Vazirmatn', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      keyframes: {
        'fade-up': {
          '0%':   { opacity: 0, transform: 'translateY(6px)' },
          '100%': { opacity: 1, transform: 'translateY(0)' },
        },
        'pop-in': {
          '0%':   { opacity: 0, transform: 'scale(.97)' },
          '100%': { opacity: 1, transform: 'scale(1)' },
        },
        shimmer: {
          '0%':   { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
      },
      animation: {
        'fade-up': 'fade-up .35s cubic-bezier(.16,1,.3,1) both',
        'pop-in':  'pop-in .2s cubic-bezier(.16,1,.3,1) both',
        shimmer:   'shimmer 2.2s linear infinite',
      },
      boxShadow: {
        glass: '0 1px 2px rgba(15,15,35,.04), 0 8px 24px -8px rgba(15,15,35,.10)',
        'glass-dark': '0 1px 2px rgba(0,0,0,.3), 0 8px 28px -8px rgba(0,0,0,.55)',
        glow: '0 0 0 1px rgba(99,116,245,.16), 0 4px 20px -4px rgba(99,116,245,.35)',
      },
    },
  },
  plugins: [],
}
