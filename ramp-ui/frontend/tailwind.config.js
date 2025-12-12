/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Custom colors for the app
        primary: {
          50: '#f2f4f8',
          100: '#dce1eb',
          200: '#b8c2d4',
          300: '#94a3bd',
          400: '#7085a6',
          500: '#6272a4',
          600: '#515f88',
          700: '#434e70',
          800: '#363e59',
          900: '#2a3045',
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
