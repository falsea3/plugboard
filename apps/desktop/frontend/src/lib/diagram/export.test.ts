import { describe, expect, it } from 'vitest';
import type { Diagram } from '../api/wire';
import { layout, makeCards, makeLinks } from './layout';
import { diagramSvg, svgBase64, type Palette } from './export';

const col = (name: string, primaryKey = false) => ({ name, type: name === 'id' ? 'bigint' : 'text', nullable: !primaryKey, default: null, primaryKey, enum: null, kind: 'text' as const });
const d: Diagram = {
  schema: 'public',
  tables: [
    { name: 'customers', kind: 'table', columns: [col('id', true), col('email')] },
    { name: 'orders', kind: 'table', columns: [col('id', true), col('customer_id'), col('note <&>')] },
  ],
  relations: [{ name: 'fk', table: 'orders', columns: ['customer_id'], refSchema: 'public', refTable: 'customers', refColumns: ['id'], onDelete: 'NO ACTION', onUpdate: 'NO ACTION' }],
};
const palette: Palette = { bg: '#111', surface: '#222', head: '#333', border: '#444', text: '#eee', faint: '#888', line: '#999', mono: 'Menlo', ui: 'Inter' };

describe('diagramSvg', () => {
  const cards = makeCards(d, s => s.length * 7);
  const pos = layout(d, cards);
  const { svg, w, h } = diagramSvg(cards, pos, makeLinks(d, cards, pos), palette);

  it('draws every table, column and relation on a plain background', () => {
    expect(svg.startsWith('<svg xmlns="http://www.w3.org/2000/svg"')).toBe(true);
    expect(svg).toContain(`<rect width="${w}" height="${h}" fill="#111"/>`);
    for (const text of ['>customers<', '>orders<', '>customer_id<', '>bigint<', '>PK<', '>FK<']) expect(svg).toContain(text);
    expect(svg.match(/<path /g)?.length).toBe(3);
  });
  it('escapes names and keeps a margin around the tables', () => {
    expect(svg).toContain('note &lt;&amp;&gt;');
    expect(w).toBeGreaterThan(Math.max(...[...cards.values()].map(c => c.w)) + 60);
    expect(h).toBeGreaterThan(64);
  });
  it('encodes text as UTF-8 base64', () => {
    expect(atob(svgBase64('<t>é</t>'))).toBe('<t>Ã©</t>');
  });
});
