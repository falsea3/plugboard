import type { ThemeMode } from './wire';

const STORAGE_KEY = 'relaydb.theme';
const media = matchMedia('(prefers-color-scheme: dark)');
let mode: ThemeMode = 'system';

function resolved(): 'light' | 'dark' {
  if (mode === 'system') return media.matches ? 'dark' : 'light';
  return mode;
}

function apply() {
  document.documentElement.dataset.theme = resolved();
}

/** Applies a theme mode now and remembers it so index.html can paint the right background before Svelte boots. */
export function setThemeMode(next: ThemeMode) {
  mode = next;
  try {
    localStorage.setItem(STORAGE_KEY, next);
  } catch {
    // storage may be unavailable; the setting itself lives in settings.json
  }
  apply();
}

media.addEventListener('change', () => {
  if (mode === 'system') apply();
});
