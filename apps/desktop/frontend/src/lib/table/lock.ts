type Lock = { readOnly: boolean; view: boolean; editable: boolean; structureEditable: boolean };

export function rowsLockReason(l: Lock & { hasColumns: boolean; hasKey: boolean }): string {
  if (l.readOnly) return 'Read-only session';
  if (l.view) return 'Views are read-only';
  if (!l.editable) return 'Rows can’t be edited here — change them in the SQL editor';
  if (l.hasColumns && !l.hasKey) return 'No primary key — rows can’t be identified safely';
  return '';
}

export function structureLockReason(l: Lock): string {
  if (l.readOnly) return 'Read-only session';
  if (l.view) return 'Views can’t be altered here';
  if (!l.structureEditable) return 'Change the structure in the SQL editor';
  return '';
}
