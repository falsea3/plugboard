import { PostgreSQL } from '@codemirror/lang-sql';
import logo from 'simple-icons/icons/postgresql.svg?raw';
import type { Engine } from './engine';

export const postgres: Engine = {
  driver: 'postgres',
  name: 'PostgreSQL',
  logo,
  background: 'var(--engine-postgres)',
  file: false,
  fileExtensions: [],
  defaultPort: 5432,
  defaultUser: 'postgres',
  defaultDatabase: 'postgres',
  schemes: ['postgres', 'postgresql', 'pgsql'],
  sslModeFromUrl(q) {
    const mode = q.get('sslmode');
    if (mode) return mode === 'verify-ca' ? 'verify-full' : mode === 'allow' ? '' : mode;
    if (q.get('ssl') === 'true' || q.get('ssl') === '1') return 'require';
    return undefined;
  },
  canPreferSsl: true,
  databaseHint: 'Optional — defaults to postgres',
  userHint: '',
  sqlDialect: PostgreSQL,
};
