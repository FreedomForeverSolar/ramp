// Theme definitions for the Ramp desktop app

export interface ThemeColors {
  bg: string;
  bgSecondary: string;
  text: string;
  textSecondary: string;
  border: string;
  accent: string;
  blue: string;
  green: string;
  purple: string;
  red: string;
  orange: string;
  yellow: string;
  pink: string;
  coral: string;
}

export interface Theme {
  id: string;
  name: string;
  colors: ThemeColors;
}

export const themes: Theme[] = [
  {
    id: 'github-dark',
    name: 'GitHub Dark',
    colors: {
      bg: '#0d1117',
      bgSecondary: '#161b22',
      text: '#c9d1d9',
      textSecondary: '#8b949e',
      border: '#30363d',
      accent: '#58a6ff',
      blue: '#58a6ff',
      green: '#3fb950',
      purple: '#a371f7',
      red: '#f85149',
      orange: '#db6d28',
      yellow: '#d29922',
      pink: '#db61a2',
      coral: '#ea6045',
    },
  },
  {
    id: 'dracula',
    name: 'Dracula',
    colors: {
      bg: '#282a36',
      bgSecondary: '#44475a',
      text: '#f8f8f2',
      textSecondary: '#6272a4',
      border: '#44475a',
      accent: '#bd93f9',
      blue: '#8be9fd',
      green: '#50fa7b',
      purple: '#bd93f9',
      red: '#ff5555',
      orange: '#ffb86c',
      yellow: '#f1fa8c',
      pink: '#ff79c6',
      coral: '#ff6e6e',
    },
  },
  {
    id: 'nord',
    name: 'Nord',
    colors: {
      bg: '#2e3440',
      bgSecondary: '#3b4252',
      text: '#eceff4',
      textSecondary: '#d8dee9',
      border: '#4c566a',
      accent: '#88c0d0',
      blue: '#81a1c1',
      green: '#a3be8c',
      purple: '#b48ead',
      red: '#bf616a',
      orange: '#d08770',
      yellow: '#ebcb8b',
      pink: '#b48ead',
      coral: '#d08770',
    },
  },
  {
    id: 'tokyo-night',
    name: 'Tokyo Night',
    colors: {
      bg: '#1a1b26',
      bgSecondary: '#24283b',
      text: '#c0caf5',
      textSecondary: '#565f89',
      border: '#414868',
      accent: '#7aa2f7',
      blue: '#7aa2f7',
      green: '#9ece6a',
      purple: '#bb9af7',
      red: '#f7768e',
      orange: '#ff9e64',
      yellow: '#e0af68',
      pink: '#ff007c',
      coral: '#f7768e',
    },
  },
  {
    id: 'catppuccin-mocha',
    name: 'Catppuccin Mocha',
    colors: {
      bg: '#1e1e2e',
      bgSecondary: '#313244',
      text: '#cdd6f4',
      textSecondary: '#a6adc8',
      border: '#45475a',
      accent: '#89b4fa',
      blue: '#89b4fa',
      green: '#a6e3a1',
      purple: '#cba6f7',
      red: '#f38ba8',
      orange: '#fab387',
      yellow: '#f9e2af',
      pink: '#f5c2e7',
      coral: '#eba0ac',
    },
  },
  {
    id: 'one-dark',
    name: 'One Dark',
    colors: {
      bg: '#282c34',
      bgSecondary: '#21252b',
      text: '#abb2bf',
      textSecondary: '#5c6370',
      border: '#3e4451',
      accent: '#61afef',
      blue: '#61afef',
      green: '#98c379',
      purple: '#c678dd',
      red: '#e06c75',
      orange: '#d19a66',
      yellow: '#e5c07b',
      pink: '#c678dd',
      coral: '#e06c75',
    },
  },
];

export function getThemeById(id: string): Theme {
  return themes.find((t) => t.id === id) ?? themes[0]; // Default to GitHub Dark
}

export function applyTheme(theme: Theme): void {
  const root = document.documentElement;
  root.style.setProperty('--color-bg', theme.colors.bg);
  root.style.setProperty('--color-bg-secondary', theme.colors.bgSecondary);
  root.style.setProperty('--color-text', theme.colors.text);
  root.style.setProperty('--color-text-secondary', theme.colors.textSecondary);
  root.style.setProperty('--color-border', theme.colors.border);
  root.style.setProperty('--color-accent', theme.colors.accent);
  root.style.setProperty('--color-blue', theme.colors.blue);
  root.style.setProperty('--color-green', theme.colors.green);
  root.style.setProperty('--color-purple', theme.colors.purple);
  root.style.setProperty('--color-red', theme.colors.red);
  root.style.setProperty('--color-orange', theme.colors.orange);
  root.style.setProperty('--color-yellow', theme.colors.yellow);
  root.style.setProperty('--color-pink', theme.colors.pink);
  root.style.setProperty('--color-coral', theme.colors.coral);
}
