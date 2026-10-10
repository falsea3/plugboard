import type { ChangeKind } from '../api/wire';

export function failMessage(kind: ChangeKind | undefined, row: number | null, error: string): string {
  if (!kind || row === null) return error;
  const verb = { delete: 'Deleting', update: 'Updating', insert: 'Inserting' }[kind];
  const which = kind === 'insert' ? 'a new row' : `row ${row + 1}`;
  return `${verb} ${which} failed — nothing was saved: ${error}`;
}
