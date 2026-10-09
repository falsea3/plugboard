import type { KeyInfo } from '../api/wire';

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

export function formatTTL(seconds: number): string {
  if (seconds < 0) return 'No expiry';
  if (seconds < 60) return `${seconds}s`;
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (d > 0) return h ? `${d}d ${h}h` : `${d}d`;
  if (h > 0) return m ? `${h}h ${m}m` : `${h}h`;
  return s ? `${m}m ${s}s` : `${m}m`;
}

export const TYPE_LABEL: Record<string, string> = {
  string: 'STR',
  hash: 'HASH',
  list: 'LIST',
  set: 'SET',
  zset: 'ZSET',
  stream: 'STREAM',
};
