import type { Connection, Driver } from '../wire';
import type { Engine } from './engine';
import { mysql } from './mysql';
import { postgres } from './postgres';
import { sqlite } from './sqlite';

export type { Engine } from './engine';
export { mysql, postgres, sqlite };

export const ENGINES: Engine[] = [postgres, mysql, sqlite];

export function engine(driver: Driver): Engine {
  const e = ENGINES.find(e => e.driver === driver);
  if (!e) throw new Error(`Unknown database engine “${driver}”`);
  return e;
}

export function engineForScheme(scheme: string): Engine | undefined {
  return ENGINES.find(e => e.schemes.includes(scheme));
}

export function nameFromFile(e: Engine, path: string): string {
  const file = path.split(/[\\/]/).pop() ?? '';
  const ext = e.fileExtensions.find(x => file.toLowerCase().endsWith('.' + x));
  return (ext ? file.slice(0, -ext.length - 1) : file) || e.name;
}

export function emptyConnection(e: Engine = ENGINES[0]): Connection {
  return {
    id: '',
    name: '',
    driver: e.driver,
    host: e.file ? '' : '127.0.0.1',
    port: e.defaultPort,
    user: e.defaultUser,
    password: '',
    savePassword: true,
    database: '',
    file: '',
    sslMode: '',
    env: '',
    color: '',
    readOnly: false,
    ssh: { enabled: false, host: '', port: 22, user: '', auth: 'password', keyFile: '', password: '', passphrase: '' },
  };
}
