import { describe, expect, it } from 'vitest';
import type { Diagram } from '../api/wire';
import { inView, layered, lineInView } from './layers';
import { layout, makeCards } from './layout';

const col = (name: string, primaryKey = false) => ({ name, type: 'int', nullable: !primaryKey, default: null, primaryKey, enum: null, kind: 'number' as const });
const chain = (n: number): Diagram => ({
  schema: 'm',
  tables: Array.from({ length: n }, (_, i) => ({ name: 't' + i, kind: 'table' as const, columns: [col('id', true), col('parent')] })),
  relations: Array.from({ length: n - 1 }, (_, i) => ({ name: 'f' + i, table: 't' + (i + 1), columns: ['parent'], refSchema: 'm', refTable: 't' + i, refColumns: ['id'], onDelete: '', onUpdate: '' })),
});
const measure = (s: string) => s.length * 7;

describe('layered', () => {
  it('lays out a chain of 10 000 tables without recursion, in bands', () => {
    const d = chain(10000);
    const cards = makeCards(d, measure);
    const pos = layout(d, cards);
    expect(pos.size).toBe(10000);
    expect(pos.get('t1')!.x).toBeGreaterThan(pos.get('t0')!.x);
    expect(pos.get('t24')!.y).toBeGreaterThan(pos.get('t23')!.y);
    const seen = new Set([...pos.values()].map(p => `${p.x},${p.y}`));
    expect(seen.size).toBe(10000);
  });
  it('puts referencing tables after the ones they reference, and survives a cycle', () => {
    const d = chain(3);
    const cards = makeCards(d, measure);
    const pos = layered(['t0', 't1', 't2'], [['t0', 't1'], ['t1', 't2'], ['t2', 't0']], cards);
    expect(pos.size).toBe(3);
    const acyclic = layered(['t0', 't1', 't2'], [['t0', 't2'], ['t1', 't2']], cards);
    expect(acyclic.get('t2')!.x).toBeGreaterThan(acyclic.get('t0')!.x);
    expect(acyclic.get('t1')!.x).toBe(acyclic.get('t0')!.x);
  });
});

describe('viewport', () => {
  const d = chain(3);
  const cards = makeCards(d, measure);
  const pos = new Map([['t0', { x: 0, y: 0 }], ['t1', { x: 5000, y: 0 }], ['t2', { x: 0, y: 5000 }]]);
  it('keeps the cards on screen, at any zoom', () => {
    expect([...inView(cards, pos, { x: 0, y: 0, k: 1, width: 800, height: 600 })]).toEqual(['t0']);
    expect(inView(cards, pos, { x: 0, y: 0, k: 0.1, width: 800, height: 600 }).size).toBe(3);
    expect([...inView(cards, pos, { x: -5000, y: 0, k: 1, width: 800, height: 600 })]).toEqual(['t1']);
  });
  it('keeps a long line that crosses the screen', () => {
    const v = { x: -2000, y: 0, k: 1, width: 800, height: 600 };
    expect(lineInView({ x: 0, y: 100 }, { x: 5000, y: 100 }, v)).toBe(true);
    expect(lineInView({ x: 0, y: 3000 }, { x: 100, y: 3000 }, v)).toBe(false);
  });
});
