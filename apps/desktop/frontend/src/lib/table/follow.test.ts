import { describe, expect, it } from 'vitest';
import type { Relation } from '../api/wire';
import { jumpTarget, linkedColumns } from './follow';

const rel = (over: Partial<Relation>): Relation => ({
  name: 'fk', table: 'order_items', columns: ['order_id'], refSchema: 'public', refTable: 'orders', refColumns: ['id'], onDelete: '', onUpdate: '', ...over,
});

const relations = [rel({}), rel({ name: 'fk2', columns: ['shop', 'sku'], refTable: 'products', refColumns: ['shop_id', 'code'] })];
const names = ['id', 'order_id', 'shop', 'sku'];

describe('following foreign keys', () => {
  it('marks every column of a key', () => {
    expect([...linkedColumns(relations, names)].sort()).toEqual([1, 2, 3]);
  });

  it('filters the referenced table by all key columns', () => {
    expect(jumpTarget(relations, names, [1, 7, 2, 'A-1'], 3)).toMatchObject({
      rel: { refTable: 'products' },
      filters: [{ column: 'shop_id', op: '=', value: '2' }, { column: 'code', op: '=', value: 'A-1' }],
    });
  });

  it('has nowhere to go from NULL or an unlinked column', () => {
    expect(jumpTarget(relations, names, [1, null, 2, 'A'], 1)).toBeNull();
    expect(jumpTarget(relations, names, [1, 7, 2, 'A'], 0)).toBeNull();
  });
});
