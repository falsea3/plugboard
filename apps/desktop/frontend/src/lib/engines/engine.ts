import type { SQLDialect } from '@codemirror/lang-sql';
import type { Driver } from '../api/wire';

export interface Engine {
  driver: Driver;
  name: string;
  logo: string;
  background: string;
  ink?: string;
  file: boolean;
  fileExtensions: string[];
  defaultPort: number;
  defaultUser: string;
  defaultDatabase: string;
  schemes: string[];
  sslModeFromUrl(q: URLSearchParams, scheme: string): string | undefined;
  canPreferSsl: boolean;
  databaseHint: string;
  userHint: string;
  sqlDialect?: SQLDialect;
}
