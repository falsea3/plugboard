import { format } from 'sql-formatter';
import type { Engine } from '../engines';

export type FormatLanguage = NonNullable<Engine['formatAs']>;

export function formatSql(text: string, language: FormatLanguage): string {
  const out = format(text, { language, keywordCase: 'preserve', tabWidth: 2, linesBetweenQueries: 1 });
  return text.endsWith('\n') ? `${out}\n` : out;
}
