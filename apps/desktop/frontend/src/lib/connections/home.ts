import type { Connection, Driver } from '../api/wire';
import { engine } from '../engines';

export type Scope = { kind: 'all' } | { kind: 'group'; name: string } | { kind: 'engine'; driver: Driver };

export function inScope(scope: Scope, c: Connection): boolean {
  if (scope.kind === 'group') return (c.group ?? '') === scope.name;
  if (scope.kind === 'engine') return c.driver === scope.driver;
  return true;
}

export function matches(query: string, c: Connection): boolean {
  const q = query.trim().toLowerCase();
  return !q || [c.name, c.host, c.database, c.file, c.env, c.group, c.user, engine(c.driver).name].some(v => v?.toLowerCase().includes(q));
}

export function connectionUrl(c: Connection): { scheme: string; address: string; database: string } {
  const e = engine(c.driver);
  if (e.file) return { scheme: '', address: c.file, database: '' };
  const user = c.user ? `${c.user}@` : '';
  const port = c.port ? `:${c.port}` : '';
  return { scheme: `${e.schemes[0]}://`, address: `${user}${c.host}${port}`, database: c.database ? `/${c.database}` : '' };
}

export function sshRoute(c: Connection): string {
  if (!c.ssh?.enabled || engine(c.driver).file) return '';
  const port = c.ssh.port && c.ssh.port !== 22 ? `:${c.ssh.port}` : '';
  return `${c.ssh.user}@${c.ssh.host || c.host}${port}`;
}

export function nextId(ids: string[], current: string, step: number): string {
  if (ids.length === 0) return '';
  const i = ids.indexOf(current);
  if (i < 0) return ids[step > 0 ? 0 : ids.length - 1];
  return ids[Math.max(0, Math.min(ids.length - 1, i + step))];
}

export function usedAgo(iso: string | undefined, now = Date.now()): string {
  const t = iso ? Date.parse(iso) : NaN;
  if (Number.isNaN(t)) return '';
  const min = Math.max(0, Math.floor((now - t) / 60000));
  if (min < 1) return 'just now';
  if (min < 60) return `${min}m`;
  const h = Math.floor(min / 60);
  if (h < 24) return `${h}h`;
  const d = Math.floor(h / 24);
  if (d === 1) return 'yesterday';
  if (d < 7) return `${d}d`;
  if (d < 30) return `${Math.floor(d / 7)}w`;
  if (d < 365) return `${Math.floor(d / 30)}mo`;
  return `${Math.floor(d / 365)}y`;
}

export const GROUP_TONES = ['var(--accent-text)', 'var(--warn)', 'var(--ok)', 'var(--info)', 'var(--danger)'];
