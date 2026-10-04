import type { ThemeMode } from '../api/wire';

const STORAGE_KEY = 'plugboard.theme';
const media = matchMedia('(prefers-color-scheme: dark)');
let mode: ThemeMode = 'system';

function resolved(): 'light' | 'dark' {
  if (mode === 'system') return media.matches ? 'dark' : 'light';
  return mode;
}

function apply() {
  document.documentElement.dataset.theme = resolved();
}

export function setThemeMode(next: ThemeMode) {
  mode = next;
  try {
    localStorage.setItem(STORAGE_KEY, next);
  } catch {
  }
  apply();
}

media.addEventListener('change', () => {
  if (mode === 'system') apply();
});
