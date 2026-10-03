import { describe, expect, it } from 'vitest';
import { HEAD_H, MAX_COLUMNS, ROW_H, layout, makeCards, makeLinks } from './diagramLayout';
import type { Column, Diagram, Relation } from './wire';

const col = (name: string, primaryKey = false): Column => ({
  name, type: 'bigint', nullable: !primaryKey, default: null, primaryKey, enum: null, kind: 'number',
});

const rel = (table: string, columns: string[], refTable: string, refColumns: string[]): Relation => ({
  name: `${table}_fk`, table, columns, refSchema: 'public', refTable, refColumns, onDelete: 'NO ACTION', onUpdate: 'NO ACTION',
});

const measure = (s: string) => s.length * 7;

const shop: Diagram = {
  schema: 'public',
  tables: [
    { name: 'customers', kind: 'table', columns: [col('id', true), col('email')] },
    { name: 'orders', kind: 'table', columns: [col('id', true), col('total'), col('customer_id')] },
    { name: 'employees', kind: 'table', columns: [col('id', true), col('manager_id')] },
    { name: 'settings', kind: 'table', columns: [col('key', true)] },
  ],
  relations: [
    rel('orders', ['customer_id'], 'customers', ['id']),
    rel('employees', ['manager_id'], 'employees', ['id']),
    { ...rel('orders', ['customer_id'], 'accounts', ['id']), refSchema: 'billing' },
  ],
};

describe('diagram layout', () => {
  const cards = makeCards(shop, measure);
  const pos = layout(shop, cards);
  const links = makeLinks(shop, cards, pos);

  it('puts the referenced table to the left of the one pointing at it', () => {
    expect(pos.get('customers')!.x + cards.get('customers')!.w).toBeLessThan(pos.get('orders')!.x);
  });

  it('draws a relation from the foreign key row to the referenced key row', () => {
    const link = links.find(l => l.relation.table === 'orders')!;
    expect(link.from).toEqual({ x: pos.get('orders')!.x, y: pos.get('orders')!.y + HEAD_H + 2 * ROW_H + ROW_H / 2 });
    expect(link.to.y).toBe(pos.get('customers')!.y + HEAD_H + ROW_H / 2);
    expect(link.fromDir).toBe(-1);
  });

  it('loops a table that points at itself and skips other schemas', () => {
    const loop = links.find(l => l.relation.table === 'employees')!;
    expect(loop.from.x).toBe(loop.to.x);
    expect(links).toHaveLength(2);
  });

  it('places every table without overlapping another', () => {
    const boxes = [...pos].map(([name, p]) => ({ ...p, w: cards.get(name)!.w, h: cards.get(name)!.h }));
    expect(boxes).toHaveLength(4);
    for (const a of boxes) {
      for (const b of boxes) {
        if (a === b) continue;
        expect(a.x + a.w <= b.x || b.x + b.w <= a.x || a.y + a.h <= b.y || b.y + b.h <= a.y).toBe(true);
      }
    }
  });

  it('keeps key columns of a wide table and counts the rest', () => {
    const wide: Diagram = {
      schema: 'public',
      tables: [
        { name: 'parent', kind: 'table', columns: [col('id', true)] },
        { name: 'wide', kind: 'table', columns: [...Array.from({ length: 30 }, (_, i) => col(`c${i}`)), col('parent_id')] },
      ],
      relations: [rel('wide', ['parent_id'], 'parent', ['id'])],
    };
    const card = makeCards(wide, measure).get('wide')!;
    expect(card.columns).toHaveLength(MAX_COLUMNS + 1);
    expect(card.columns.at(-1)!.name).toBe('parent_id');
    expect(card.hidden).toBe(31 - MAX_COLUMNS - 1);
  });
});
