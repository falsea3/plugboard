const NAMES: Record<string, string> = { '↵': 'Enter', '⌫': 'Backspace', '⇥': 'Tab' };
const MODS: [string, string][] = [['⌃', 'Ctrl'], ['⌘', 'Ctrl'], ['⌥', 'Alt'], ['⇧', 'Shift']];

export const isMac = typeof navigator !== 'undefined' && /Mac/.test(navigator.userAgent);

export function pcKeys(label: string): string {
  if (NAMES[label]) return NAMES[label];
  return label.replace(/([⌃⌥⇧⌘]+)(Tab|[^\s⌃⌥⇧⌘)·—,]|,)/g, (_, mods: string, key: string) => {
    const names: string[] = [];
    for (const [glyph, name] of MODS) if (mods.includes(glyph) && !names.includes(name)) names.push(name);
    return [...names, NAMES[key] ?? key].join('+');
  });
}

export function keys(label: string): string {
  return isMac ? label : pcKeys(label);
}
