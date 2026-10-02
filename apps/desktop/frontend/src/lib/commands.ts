import { app } from './stores/app.svelte';
import type { ThemeMode } from './wire';

export const REFRESH_EVENT = 'plugboard:refresh';
export const FILTER_TABLES_EVENT = 'plugboard:filter-tables';
export const COMMIT_EVENT = 'plugboard:commit';
export const ADD_ROW_EVENT = 'plugboard:add-row';
export const FILTER_ROWS_EVENT = 'plugboard:filter-rows';

export function runCommand(command: string) {
  const ws = app.active;
  if (command.startsWith('theme:')) {
    app.updateSettings({ theme: command.slice(6) as ThemeMode });
    return;
  }
  switch (command) {
    case 'about':
      app.settingsOpen = 'about';
      break;
    case 'settings':
      app.settingsOpen = 'general';
      break;
    case 'new-connection':
      app.editing = null;
      break;
    case 'open-sqlite':
      app.openSQLiteFile();
      break;
    case 'switch-connection':
      app.switcherOpen = true;
      break;
    case 'home':
      if (app.showHome) app.leaveHome();
      else app.goHome();
      break;
    case 'new-query':
      ws?.newQuery();
      break;
    case 'close-tab':
      if (ws?.activeTabId) app.requestCloseTab(ws, ws.activeTabId);
      break;
    case 'close-connection':
      if (ws) app.requestClose(ws);
      break;
    case 'commit':
      window.dispatchEvent(new CustomEvent(COMMIT_EVENT));
      break;
    case 'add-row':
      window.dispatchEvent(new CustomEvent(ADD_ROW_EVENT));
      break;
    case 'next-tab':
      ws?.selectTabByOffset(1);
      break;
    case 'prev-tab':
      ws?.selectTabByOffset(-1);
      break;
    case 'refresh':
      window.dispatchEvent(new CustomEvent(REFRESH_EVENT));
      break;
    case 'filter':
      window.dispatchEvent(new CustomEvent(ws?.activeTab?.kind === 'table' ? FILTER_ROWS_EVENT : FILTER_TABLES_EVENT));
      break;
    case 'filter-tables':
      window.dispatchEvent(new CustomEvent(FILTER_TABLES_EVENT));
      break;
  }
}

const SHORTCUTS: Record<string, string> = {
  ',': 'settings',
  n: 'new-connection',
  o: 'open-sqlite',
  k: 'switch-connection',
  'shift+h': 'home',
  t: 'new-query',
  w: 'close-tab',
  'shift+w': 'close-connection',
  'shift+]': 'next-tab',
  'shift+}': 'next-tab',
  'shift+[': 'prev-tab',
  'shift+{': 'prev-tab',
  r: 'refresh',
  f: 'filter',
  'shift+f': 'filter-tables',
  s: 'commit',
  i: 'add-row',
};

function overlayOpen() {
  return app.editing !== undefined || app.passwordFor !== null || app.switcherOpen || app.settingsOpen !== null || app.pendingClose !== null;
}

export function handleShortcut(e: KeyboardEvent) {
  if (!(e.metaKey || e.ctrlKey) || e.altKey) return;
  const key = e.key.toLowerCase();

  if (!e.shiftKey && /^[1-9]$/.test(key)) {
    const ws = app.active;
    if (!ws || overlayOpen()) return;
    e.preventDefault();
    const i = key === '9' ? ws.tabs.length - 1 : Number(key) - 1;
    if (ws.tabs[i]) ws.activeTabId = ws.tabs[i].id;
    return;
  }

  const command = SHORTCUTS[(e.shiftKey ? 'shift+' : '') + key];
  if (!command) return;
  if ((command === 'filter' || command === 'filter-tables') && (e.target as HTMLElement | null)?.closest?.('.cm-editor')) return;
  e.preventDefault();
  if (overlayOpen() && command !== 'settings') return;
  runCommand(command);
}
