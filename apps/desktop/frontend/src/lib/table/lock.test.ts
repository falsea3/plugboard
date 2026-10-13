import { describe, expect, it } from 'vitest';
import { rowsLockReason, structureLockReason } from './lock';

const base = { readOnly: false, view: false, editable: true, structureEditable: true, hasColumns: true, hasKey: true };

describe('lock reasons', () => {
  it('lets an ordinary table be edited', () => {
    expect(rowsLockReason(base)).toBe('');
    expect(structureLockReason(base)).toBe('');
  });
  it('names the first thing that stops editing', () => {
    expect(rowsLockReason({ ...base, readOnly: true, view: true })).toBe('Read-only session');
    expect(rowsLockReason({ ...base, view: true })).toBe('Views are read-only');
    expect(rowsLockReason({ ...base, hasKey: false })).toMatch(/No primary key/);
    expect(rowsLockReason({ ...base, hasKey: false, hasColumns: false })).toBe('');
    expect(structureLockReason({ ...base, structureEditable: false })).toMatch(/SQL editor/);
  });
});
