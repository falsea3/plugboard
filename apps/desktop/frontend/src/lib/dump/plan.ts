import type { DumpOptions, DumpProgress, ImportProgress } from '../api/wire';
import type { MenuItem } from '../grid/grid';
import { formatBytes, formatCount } from '../ui/format';

export type DumpWhat = 'all' | 'structure' | 'data';

export function dumpName(database: string, name: string, now = new Date()): string {
  const base = (database || name || 'dump').replace(/[^\p{L}\p{N}._-]+/gu, '_').replace(/^_+|_+$/g, '') || 'dump';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${base}-${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}.sql`;
}

export function dumpOptions(what: DumpWhat, schemas: string[], all: string[], clean: boolean, inserts: boolean, owners = false): DumpOptions {
  return {
    schemas: schemas.length === all.length ? [] : schemas,
    structure: what !== 'data',
    data: what !== 'structure',
    clean: clean && what !== 'data',
    inserts: inserts && what !== 'structure',
    owners: owners && what !== 'data',
  };
}

const PHASES: Record<string, string> = {
  structure: 'Writing structure',
  data: 'Writing rows',
  constraints: 'Writing keys, indexes and the rest',
  done: 'Finishing',
};

export function dumpStep(p: DumpProgress | null): string {
  if (!p) return 'Reading the catalog…';
  const phase = PHASES[p.phase] ?? 'Working';
  return p.phase === 'data' && p.object ? `${phase} · ${p.object}` : phase;
}

export function dumpDetail(p: DumpProgress | null): string {
  if (!p) return '';
  const parts = [formatBytes(p.bytes)];
  if (p.rows > 0) parts.unshift(formatCount(p.rows, 'row'));
  if (p.phase === 'data' && p.tables > 0) parts.push(`${p.done} of ${formatCount(p.tables, 'table')}`);
  return parts.join(' · ');
}

export function dumpShare(p: DumpProgress | null): number | null {
  if (!p || p.phase !== 'data' || p.tables === 0) return null;
  return Math.min(1, p.done / p.tables);
}

export function importShare(p: ImportProgress | null): number | null {
  if (!p || p.total <= 0) return null;
  return Math.min(1, p.bytes / p.total);
}

export function dumpMenu(): MenuItem[] {
  return [
    { id: 'dump-export', label: 'Export dump…' },
    { id: 'dump-import', label: 'Import dump…' },
  ];
}
