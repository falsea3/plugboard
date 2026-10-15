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
