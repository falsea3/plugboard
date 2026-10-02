import { DEFAULT_PORTS, emptyConnection, type Connection, type Driver } from './wire';
import { sqliteName } from './format';

const SCHEMES: Record<string, Driver> = {
  postgres: 'postgres',
  postgresql: 'postgres',
  pgsql: 'postgres',
  mysql: 'mysql',
  mariadb: 'mysql',
  sqlite: 'sqlite',
  sqlite3: 'sqlite',
  file: 'sqlite',
};

export function parseConnectionUrl(input: string): Connection {
  let raw = input.trim().replace(/^jdbc:/i, '');
  raw = raw.replace(/^[A-Z0-9_]+\s*=\s*/i, '').replace(/^["']|["']$/g, '');
  const m = /^([a-z0-9+]+):/i.exec(raw);
  if (!m) throw new Error('Expected a URL like postgres://user:password@host:5432/database');
  const scheme = m[1].toLowerCase().split('+')[0];
  const driver = SCHEMES[scheme];
  if (!driver) throw new Error(`Unsupported scheme “${m[1]}:” — use postgres://, mysql:// or sqlite://`);

  const c = emptyConnection(driver);

  if (driver === 'sqlite') {
    const path = decodeURIComponent(raw.slice(m[0].length).replace(/^\/\/(?=\/)/, '').replace(/^\/\/\.?/, '').split('?')[0]);
    if (!path) throw new Error('The SQLite URL has no file path');
    c.file = path;
    c.name = sqliteName(path);
    return c;
  }

  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    throw new Error('This doesn’t look like a valid connection URL');
  }
  c.host = url.hostname.replace(/^\[|\]$/g, '') || '127.0.0.1';
  c.port = url.port ? Number(url.port) : DEFAULT_PORTS[driver];
  if (url.username) c.user = decodeURIComponent(url.username);
  if (url.password) c.password = decodeURIComponent(url.password);
  c.database = decodeURIComponent(url.pathname.replace(/^\//, ''));

  const q = url.searchParams;
  if (driver === 'postgres') {
    const mode = q.get('sslmode');
    if (mode) c.sslMode = mode === 'verify-ca' ? 'verify-full' : mode === 'allow' ? '' : mode;
    else if (q.get('ssl') === 'true' || q.get('ssl') === '1') c.sslMode = 'require';
  } else {
    const mode = (q.get('ssl-mode') ?? q.get('sslmode') ?? '').toLowerCase();
    const tls = (q.get('tls') ?? '').toLowerCase();
    if (mode === 'disabled' || tls === 'false') c.sslMode = 'disable';
    else if (mode === 'required' || tls === 'skip-verify') c.sslMode = 'require';
    else if (mode === 'verify_identity' || mode === 'verify_ca' || tls === 'true') c.sslMode = 'verify-full';
  }

  c.name = c.database ? `${c.database} @ ${c.host}` : c.host;
  return c;
}
