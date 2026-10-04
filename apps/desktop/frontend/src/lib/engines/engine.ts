import type { SQLDialect } from '@codemirror/lang-sql';
import type { Driver } from '../api/wire';

export interface Engine {
  driver: Driver;
  name: string;
  logo: string;
  background: string;
  file: boolean;
  fileExtensions: string[];
  defaultPort: number;
  defaultUser: string;
  defaultDatabase: string;
  schemes: string[];
  sslModeFromUrl(q: URLSearchParams): string | undefined;
  sqlDialect: SQLDialect;
}
