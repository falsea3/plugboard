import { describe, expect, it } from 'vitest';
import { dumpDetail, dumpName, dumpOptions, dumpShare, dumpStep, importShare } from './plan';

describe('dump plan', () => {
  it('names the file after the database and the time', () => {
    const at = new Date(2026, 9, 6, 9, 5);
    expect(dumpName('shop', 'Local', at)).toBe('shop-2026-10-06-0905.sql');
    expect(dumpName('', 'My DB / prod', at)).toBe('My_DB_prod-2026-10-06-0905.sql');
    expect(dumpName('/var/data/app.db', '', at)).toBe('var_data_app.db-2026-10-06-0905.sql');
    expect(dumpName('', '', at)).toBe('dump-2026-10-06-0905.sql');
  });

  it('sends no schema list when all are picked and drops options that do not apply', () => {
    expect(dumpOptions('all', ['a', 'b'], ['a', 'b'], true, true, true)).toEqual({ schemas: [], structure: true, data: true, clean: true, inserts: true, owners: true });
    expect(dumpOptions('structure', ['a'], ['a', 'b'], true, true)).toEqual({ schemas: ['a'], structure: true, data: false, clean: true, inserts: false, owners: false });
    expect(dumpOptions('data', ['a', 'b'], ['a', 'b'], true, false, true)).toEqual({ schemas: [], structure: false, data: true, clean: false, inserts: false, owners: false });
  });

  it('describes progress', () => {
    expect(dumpStep(null)).toBe('Reading the catalog…');
    const p = { phase: 'data', object: 'public.orders', tables: 4, done: 1, rows: 10, bytes: 100 };
    expect(dumpStep(p)).toBe('Writing rows · public.orders');
    expect(dumpShare(p)).toBe(0.25);
    expect(dumpDetail(p)).toBe('10 rows · 100 B · 1 of 4 tables');
    expect(dumpDetail({ ...p, phase: 'structure', rows: 0, bytes: 2048 })).toBe('2.0 KB');
    expect(dumpShare({ ...p, phase: 'structure' })).toBeNull();
    expect(importShare({ bytes: 50, total: 200, statements: 1, errors: 0, current: '' })).toBe(0.25);
    expect(importShare({ bytes: 50, total: 0, statements: 1, errors: 0, current: '' })).toBeNull();
  });
});
