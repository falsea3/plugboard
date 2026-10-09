import { MySQL } from '@codemirror/lang-sql';
import logo from '../assets/mysql-dolphin.svg?raw';
import type { Engine } from './engine';

export const mysql: Engine = {
  driver: 'mysql',
  name: 'MySQL',
  logo,
  background: 'var(--engine-mysql)',
  file: false,
  fileExtensions: [],
  defaultPort: 3306,
  defaultUser: 'root',
  defaultDatabase: '',
  schemes: ['mysql', 'mariadb'],
  sslModeFromUrl(q) {
    const mode = (q.get('ssl-mode') ?? q.get('sslmode') ?? '').toLowerCase();
    const tls = (q.get('tls') ?? '').toLowerCase();
    if (mode === 'disabled' || tls === 'false') return 'disable';
    if (mode === 'required' || tls === 'skip-verify') return 'require';
    if (mode === 'verify_identity' || mode === 'verify_ca' || tls === 'true') return 'verify-full';
    return undefined;
  },
  canPreferSsl: true,
  databaseHint: 'Optional',
  userHint: '',
  sqlDialect: MySQL,
};
