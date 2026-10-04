import dagre from '@dagrejs/dagre';
import type { Column, Diagram, DiagramTable, Relation } from '../api/wire';

export const ROW_H = 22;
export const HEAD_H = 32;
export const MAX_COLUMNS = 16;
const PAD_BOTTOM = 6;
const GAP = 36;

export type Measure = (text: string) => number;

export interface Card {
  table: DiagramTable;
  columns: Column[];
  hidden: number;
  foreign: Set<string>;
  w: number;
  h: number;
}

export interface Point {
  x: number;
  y: number;
}

export interface Link {
  index: number;
  relation: Relation;
  path: string;
  from: Point;
  to: Point;
  fromDir: 1 | -1;
  toDir: 1 | -1;
}

export function linked(d: Diagram): Relation[] {
  const names = new Set(d.tables.map(t => t.name));
  return d.relations.filter(r => r.refSchema === d.schema && names.has(r.table) && names.has(r.refTable));
}

export function makeCards(d: Diagram, measure: Measure): Map<string, Card> {
  const foreign = new Map<string, Set<string>>();
  const referenced = new Map<string, Set<string>>();
  const add = (m: Map<string, Set<string>>, table: string, cols: string[]) => {
    const set = m.get(table) ?? new Set<string>();
    cols.forEach(c => set.add(c));
    m.set(table, set);
  };
  for (const r of linked(d)) {
    add(foreign, r.table, r.columns);
    add(referenced, r.refTable, r.refColumns);
  }
  const cards = new Map<string, Card>();
  for (const table of d.tables) {
    const fk = foreign.get(table.name) ?? new Set<string>();
    const ref = referenced.get(table.name) ?? new Set<string>();
    const columns =
      table.columns.length <= MAX_COLUMNS
        ? table.columns
        : table.columns.filter((c, i) => i < MAX_COLUMNS || c.primaryKey || fk.has(c.name) || ref.has(c.name));
    const hidden = table.columns.length - columns.length;
    const widest = Math.max(measure(table.name) + 56, ...columns.map(c => measure(c.name) + measure(c.type) + 64));
    const w = Math.round(Math.min(380, Math.max(180, widest)));
    const h = HEAD_H + (columns.length + (hidden > 0 ? 1 : 0)) * ROW_H + PAD_BOTTOM;
    cards.set(table.name, { table, columns, hidden, foreign: fk, w, h });
  }
  return cards;
}

export function layout(d: Diagram, cards: Map<string, Card>): Map<string, Point> {
  const relations = linked(d).filter(r => r.table !== r.refTable);
  const connected = new Set(relations.flatMap(r => [r.table, r.refTable]));
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: 'LR', nodesep: GAP, ranksep: 120, marginx: 0, marginy: 0 });
  g.setDefaultEdgeLabel(() => ({}));
  for (const name of connected) {
    const c = cards.get(name)!;
    g.setNode(name, { width: c.w, height: c.h });
  }
  for (const r of relations) g.setEdge(r.refTable, r.table);
  dagre.layout(g);

  const out = new Map<string, Point>();
  let bottom = 0;
  let right = 0;
  for (const name of connected) {
    const n = g.node(name);
    const c = cards.get(name)!;
    const p = { x: Math.round(n.x - c.w / 2), y: Math.round(n.y - c.h / 2) };
    out.set(name, p);
    bottom = Math.max(bottom, p.y + c.h);
    right = Math.max(right, p.x + c.w);
  }

  const alone = d.tables.map(t => t.name).filter(n => !connected.has(n));
  if (alone.length > 0) {
    const width = Math.max(right, Math.ceil(Math.sqrt(alone.length)) * (240 + GAP));
    let x = 0;
    let y = connected.size > 0 ? bottom + GAP * 2 : 0;
    let rowH = 0;
    for (const name of alone) {
      const c = cards.get(name)!;
      if (x > 0 && x + c.w > width) {
        x = 0;
        y += rowH + GAP;
        rowH = 0;
      }
      out.set(name, { x, y });
      x += c.w + GAP;
      rowH = Math.max(rowH, c.h);
    }
  }
  return out;
}

function rowY(card: Card, pos: Point, column: string): number {
  const i = card.columns.findIndex(c => c.name === column);
  if (i < 0) return pos.y + HEAD_H / 2;
  return pos.y + HEAD_H + i * ROW_H + ROW_H / 2;
}

export function makeLinks(d: Diagram, cards: Map<string, Card>, pos: Map<string, Point>): Link[] {
  const out: Link[] = [];
  d.relations.forEach((relation, index) => {
    if (relation.refSchema !== d.schema) return;
    const child = cards.get(relation.table);
    const parent = cards.get(relation.refTable);
    const cp = pos.get(relation.table);
    const pp = pos.get(relation.refTable);
    if (!child || !parent || !cp || !pp) return;
    const fy = rowY(child, cp, relation.columns[0]);
    const ty = rowY(parent, pp, relation.refColumns[0]);
    if (relation.table === relation.refTable) {
      const x = cp.x + child.w;
      const loop = 48;
      out.push({
        index, relation, from: { x, y: fy }, to: { x, y: ty }, fromDir: 1, toDir: 1,
        path: `M ${x} ${fy} C ${x + loop} ${fy}, ${x + loop} ${ty}, ${x} ${ty}`,
      });
      return;
    }
    const childLeft = cp.x + child.w / 2 >= pp.x + parent.w / 2;
    const fromDir = childLeft ? -1 : 1;
    const toDir = childLeft ? 1 : -1;
    const fx = childLeft ? cp.x : cp.x + child.w;
    const tx = childLeft ? pp.x + parent.w : pp.x;
    const bend = Math.max(40, Math.abs(tx - fx) / 2);
    out.push({
      index, relation, from: { x: fx, y: fy }, to: { x: tx, y: ty }, fromDir, toDir,
      path: `M ${fx} ${fy} C ${fx + fromDir * bend} ${fy}, ${tx + toDir * bend} ${ty}, ${tx} ${ty}`,
    });
  });
  return out;
}

export function manyMark(p: Point, dir: number): string {
  const tip = p.x + dir * 11;
  return `M ${tip} ${p.y} L ${p.x} ${p.y - 6} M ${tip} ${p.y} L ${p.x} ${p.y} M ${tip} ${p.y} L ${p.x} ${p.y + 6}`;
}

export function oneMark(p: Point, dir: number): string {
  const x = p.x + dir * 8;
  return `M ${x} ${p.y - 6} L ${x} ${p.y + 6}`;
}

export function bounds(cards: Map<string, Card>, pos: Map<string, Point>) {
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const [name, p] of pos) {
    const c = cards.get(name);
    if (!c) continue;
    minX = Math.min(minX, p.x);
    minY = Math.min(minY, p.y);
    maxX = Math.max(maxX, p.x + c.w);
    maxY = Math.max(maxY, p.y + c.h);
  }
  if (!isFinite(minX)) return { x: 0, y: 0, w: 0, h: 0 };
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY };
}

export const columnList = (cols: string[]) => (cols.length === 1 ? cols[0] : `(${cols.join(', ')})`);

export const relationText = (r: Relation) => `${r.table}.${columnList(r.columns)} → ${r.refTable}.${columnList(r.refColumns)}`;
