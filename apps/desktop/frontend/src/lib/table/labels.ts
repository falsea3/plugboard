import type { Column, RowCount } from '../api/wire';

const fmt = (n: number) => n.toLocaleString('en-US');

export function rowsLabel(loaded: number | null, hasMore: boolean, count: RowCount | null): string {
  if (count?.known) return `${count.exact ? '' : '~'}${fmt(count.count)} ${count.count === 1 ? 'row' : 'rows'}`;
  if (loaded === null) return '';
  return `${fmt(loaded)}${hasMore ? '+' : ''} ${loaded === 1 && !hasMore ? 'row' : 'rows'}`;
}

export function wherePlaceholder(columns: Column[]): string {
  const text = columns.find(c => !c.primaryKey && c.kind === 'text');
  if (text) return `${text.name} = '…'`;
  const any = columns[0];
  return any ? `${any.name} = …` : 'condition';
}
