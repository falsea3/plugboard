import { ENGINES, emptyConnection, engineForScheme, nameFromFile } from './engines';
import type { Connection } from './wire';

export function parseConnectionUrl(input: string): Connection {
  let raw = input.trim().replace(/^jdbc:/i, '');
  raw = raw.replace(/^[A-Z0-9_]+\s*=\s*/i, '').replace(/^["']|["']$/g, '');
  const m = /^([a-z0-9+]+):/i.exec(raw);
  if (!m) throw new Error('Expected a URL like postgres://user:password@host:5432/database');
  const scheme = m[1].toLowerCase().split('+')[0];
  const e = engineForScheme(scheme);
  if (!e) throw new Error(`Unsupported scheme “${m[1]}:” — use ${ENGINES.map(e => e.schemes[0] + '://').join(', ')}`);

  const c = emptyConnection(e);

  if (e.file) {
    const path = decodeURIComponent(raw.slice(m[0].length).replace(/^\/\/(?=\/)/, '').replace(/^\/\/\.?/, '').split('?')[0]);
    if (!path) throw new Error(`The ${e.name} URL has no file path`);
    c.file = path;
    c.name = nameFromFile(e, path);
    return c;
  }

  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    throw new Error('This doesn’t look like a valid connection URL');
  }
  c.host = url.hostname.replace(/^\[|\]$/g, '') || '127.0.0.1';
  c.port = url.port ? Number(url.port) : e.defaultPort;
  if (url.username) c.user = decodeURIComponent(url.username);
  if (url.password) c.password = decodeURIComponent(url.password);
  c.database = decodeURIComponent(url.pathname.replace(/^\//, ''));

  const sslMode = e.sslModeFromUrl(url.searchParams);
  if (sslMode !== undefined) c.sslMode = sslMode;

  c.name = c.database ? `${c.database} @ ${c.host}` : c.host;
  return c;
}
