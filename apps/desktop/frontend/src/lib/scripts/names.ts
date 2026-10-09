import type { QueryTabState } from '../api/wire';

export type ScriptTab = { title: string; sql: string; saved: string; script?: string };

export const isDirty = (tab: ScriptTab) => tab.sql !== tab.saved;

export function nameProblem(name: string, taken: string[], current = ''): string {
  if (!name.trim()) return 'Give the script a name.';
  if (name !== name.trim()) return 'A name can’t start or end with a space.';
  if (name.length > 120) return 'A name is at most 120 characters.';
  if (name.startsWith('.')) return 'A name can’t start with a dot.';
  if (/[/\\:\p{Cc}]/u.test(name)) return 'A name can’t contain / \\ : or control characters.';
  const lower = name.toLowerCase();
  if (lower !== current.toLowerCase() && taken.some(n => n.toLowerCase() === lower)) return 'A script with this name already exists.';
  return '';
}

export function suggestName(title: string, taken: string[]): string {
  const base = title.replace(/[/\\:\p{Cc}]/gu, ' ').trim().replace(/^\.+/, '') || 'Query';
  if (!nameProblem(base, taken)) return base;
  for (let n = 2; ; n++) {
    const name = `${base} ${n}`;
    if (!nameProblem(name, taken)) return name;
  }
}

export function untitled(titles: string[], prefix = 'Query'): string {
  for (let n = 1; ; n++) {
    const title = `${prefix} ${n}`;
    if (!titles.includes(title)) return title;
  }
}

export const toState = (tab: ScriptTab, active: boolean): QueryTabState => ({
  title: tab.title,
  sql: tab.sql,
  script: tab.script ?? '',
  saved: tab.saved,
  active,
});

export const sortNames = (names: string[]) => [...names].sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()));
