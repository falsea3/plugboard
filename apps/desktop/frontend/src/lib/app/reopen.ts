const KEY = 'plugboard.openConnections';

export function rememberOpen(ids: string[], active: string) {
  try {
    localStorage.setItem(KEY, JSON.stringify({ ids, active }));
  } catch {
    return;
  }
}

export function lastOpen(): { ids: string[]; active: string } {
  try {
    const v = JSON.parse(localStorage.getItem(KEY) ?? 'null');
    if (v && Array.isArray(v.ids)) return { ids: v.ids.filter((x: unknown) => typeof x === 'string'), active: typeof v.active === 'string' ? v.active : '' };
  } catch {
    return { ids: [], active: '' };
  }
  return { ids: [], active: '' };
}
