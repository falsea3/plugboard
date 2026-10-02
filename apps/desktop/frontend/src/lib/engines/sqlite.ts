import { SQLite } from '@codemirror/lang-sql';
import logo from 'simple-icons/icons/sqlite.svg?raw';
import type { Engine } from './engine';

export const sqlite: Engine = {
  driver: 'sqlite',
  name: 'SQLite',
  logo,
  background: 'var(--engine-sqlite)',
  file: true,
  fileExtensions: ['db', 'sqlite', 'sqlite3', 'db3'],
  defaultPort: 0,
  defaultUser: '',
  defaultDatabase: '',
  schemes: ['sqlite', 'sqlite3', 'file'],
  sslModeFromUrl: () => undefined,
  sqlDialect: SQLite,
};
