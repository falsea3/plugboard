import type { KeyInfo } from '../api/wire';
import type { MenuItem } from '../grid/grid';
import type { SelectOption } from '../ui/Select.svelte';

export type KeyRow =
  | { kind: 'folder'; path: string; name: string; depth: number; count: number; open: boolean }
  | { kind: 'key'; key: KeyInfo; name: string; depth: number };

type Folder = { name: string; path: string; folders: Map<string, Folder>; keys: KeyInfo[]; count: number };

const folder = (name: string, path: string): Folder => ({ name, path, folders: new Map(), keys: [], count: 0 });

export function keyRows(keys: KeyInfo[], open: ReadonlySet<string>, sep = ':'): KeyRow[] {
  const root = folder('', '');
  for (const k of keys) {
    const parts = sep ? k.name.split(sep) : [k.name];
    let at = root;
    at.count++;
    for (let i = 0; i < parts.length - 1; i++) {
      const path = parts.slice(0, i + 1).join(sep) + sep;
      let next = at.folders.get(parts[i]);
      if (!next) at.folders.set(parts[i], (next = folder(parts[i], path)));
      next.count++;
      at = next;
    }
    at.keys.push(k);
  }
  const rows: KeyRow[] = [];
  const walk = (f: Folder, depth: number) => {
    for (const sub of [...f.folders.values()].sort((a, b) => a.name.localeCompare(b.name))) {
      if (sub.count === 1) {
        const only = onlyKey(sub);
        rows.push({ kind: 'key', key: only, name: only.name.slice(f.path.length), depth });
        continue;
      }
      const isOpen = open.has(sub.path);
      rows.push({ kind: 'folder', path: sub.path, name: sub.name, depth, count: sub.count, open: isOpen });
      if (isOpen) walk(sub, depth + 1);
    }
    for (const k of [...f.keys].sort((a, b) => a.name.localeCompare(b.name))) {
      rows.push({ kind: 'key', key: k, name: k.name.slice(f.path.length), depth });
    }
  };
  walk(root, 0);
  return rows;
}

function onlyKey(f: Folder): KeyInfo {
  return f.keys[0] ?? onlyKey([...f.folders.values()][0]);
}

export function scanPattern(filter: string): string {
  const f = filter.trim();
  if (!f) return '*';
  if (/[*?[]/.test(f)) return f;
  return `*${f.replace(/\\/g, '\\\\')}*`;
}

export function keyMenu(readOnly: boolean): MenuItem[] {
  const items: MenuItem[] = [
    { id: 'key-open', label: 'Open' },
    { id: 'key-copy-name', label: 'Copy name' },
  ];
  return readOnly ? items : [...items, 'sep', { id: 'key-delete', label: 'Delete…', danger: true }];
}

export function databaseOptions(databases: string[], counts: Record<string, number>): SelectOption<string>[] {
  const keys = (db: string) => counts[db] ?? 0;
  const label = (n: number) => (n === 1 ? '1 key' : `${n.toLocaleString('en-US')} keys`);
  return [...databases]
    .sort((a, b) => Number(keys(b) > 0) - Number(keys(a) > 0) || Number(a) - Number(b))
    .map(db => ({ value: db, label: `Database ${db}`, hint: keys(db) > 0 ? label(keys(db)) : 'empty', muted: keys(db) === 0 }));
}
