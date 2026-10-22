import { defineConfig, presetUno, presetIcons } from 'unocss'

export default defineConfig({
  presets: [
    presetUno({ preflight: false }),
    presetIcons({
      scale: 1.2,
      warn: true,
      extraProperties: {
        'display': 'inline-block',
        'vertical-align': 'middle',
      },
    }),
  ],
  theme: {
    colors: {
      brand: {
        50: '#f0fdf4',
        100: '#dcfce7',
        500: '#22c55e',
        600: '#16a34a',
        700: '#15803d',
      },
      dark: {
        header: '#0f172a',
        sidebar: '#1e293b',
        sidebarHover: '#334155',
      }
    }
  },
  shortcuts: {
    'btn-primary': 'bg-brand-600 hover:bg-brand-700 text-white font-medium py-1.5 px-3 rounded shadow-sm transition',
    'badge-pill': 'px-2 py-0.5 text-xs rounded-full font-medium',
    'card-panel': 'bg-white rounded-lg border border-slate-200/80 shadow-sm p-4',
  }
})
