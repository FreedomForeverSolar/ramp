/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Primary color uses CSS variable from theme
        primary: {
          50: 'color-mix(in srgb, var(--color-accent) 10%, white)',
          100: 'color-mix(in srgb, var(--color-accent) 20%, white)',
          200: 'color-mix(in srgb, var(--color-accent) 40%, white)',
          300: 'color-mix(in srgb, var(--color-accent) 60%, white)',
          400: 'color-mix(in srgb, var(--color-accent) 80%, white)',
          500: 'var(--color-accent)',
          600: 'color-mix(in srgb, var(--color-accent) 90%, black)',
          700: 'color-mix(in srgb, var(--color-accent) 70%, black)',
          800: 'color-mix(in srgb, var(--color-accent) 50%, black)',
          900: 'color-mix(in srgb, var(--color-accent) 30%, black)',
        },
        // GitHub Dark background scale
        gray: {
          50: '#f0f6fc',   // Lightest
          100: '#c9d1d9',  // Default text
          200: '#b1bac4',
          300: '#8b949e',  // Muted
          400: '#6e7681',
          500: '#484f58',  // Border
          600: '#30363d',  // Border muted
          700: '#21262d',  // Canvas subtle
          800: '#161b22',  // Canvas default
          900: '#0d1117',  // Canvas inset
        },
        // GitHub Dark accent colors
        gh: {
          // Canvas (backgrounds)
          canvasDefault: '#0d1117',
          canvasOverlay: '#161b22',
          canvasInset: '#010409',
          canvasSubtle: '#161b22',
          // Borders
          borderDefault: '#30363d',
          borderMuted: '#21262d',
          borderSubtle: '#6e7681',
          // Text
          fgDefault: '#c9d1d9',
          fgMuted: '#8b949e',
          fgSubtle: '#6e7681',
          // Accent colors
          blue: '#58a6ff',
          green: '#3fb950',
          purple: '#a371f7',
          red: '#f85149',
          orange: '#db6d28',
          yellow: '#d29922',
          pink: '#db61a2',
          coral: '#ea6045',
          // State colors
          successFg: '#3fb950',
          dangerFg: '#f85149',
          warningFg: '#d29922',
          // Diff colors
          diffAdd: '#238636',
          diffDelete: '#da3633',
          diffChange: '#9e6a03',
        },
      },
      animation: {
        'dropdown-in': 'dropdown-in 0.15s ease-out',
        'dropdown-in-up': 'dropdown-in-up 0.15s ease-out',
      },
      keyframes: {
        'dropdown-in': {
          '0%': { opacity: '0', transform: 'scale(0.95) translateY(-4px)' },
          '100%': { opacity: '1', transform: 'scale(1) translateY(0)' },
        },
        'dropdown-in-up': {
          '0%': { opacity: '0', transform: 'scale(0.95) translateY(4px)' },
          '100%': { opacity: '1', transform: 'scale(1) translateY(0)' },
        },
      },
    },
  },
  plugins: [],
}
