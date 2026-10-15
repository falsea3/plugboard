import type { CellValue, ColumnKind } from '../api/wire';
import { formatCell } from '../ui/format';

type Font = 'name' | 'type' | 'text' | 'mono';
type Measure = (text: string, font: Font) => number;

const estimate: Measure = (t, f) => t.length * (f === 'type' ? 6.6 : f === 'mono' ? 7.3 : 7.4);
let measurer: Measure | null | undefined;

function measure(): Measure {
  if (measurer === undefined) {
    measurer = null;
    try {
      const ctx = document.createElement('canvas').getContext('2d');
      if (ctx) {
        const css = getComputedStyle(document.documentElement);
        const ui = css.getPropertyValue('--font-ui').trim() || 'sans-serif';
        const mono = css.getPropertyValue('--font-mono').trim() || 'monospace';
        const fonts: Record<Font, string> = { name: `500 13px ${ui}`, type: `400 10.5px ${ui}`, text: `13px ${ui}`, mono: `12px ${mono}` };
        measurer = (t, f) => ((ctx.font = fonts[f]), ctx.measureText(t).width);
      }
    } catch {
      measurer = null;
    }
  }
  return measurer ?? estimate;
}

const PAD = 24;
const ICON = 17;
const TYPE_CHARS = 14;

export type WidthColumn = { name: string; type: string; kind?: ColumnKind };

export function columnWidth(col: WidthColumn, rows: CellValue[][], index: number, opts: { fit?: boolean; key?: boolean } = {}): number {
  const m = measure();
  const max = opts.fit ? 800 : 360;
  const named = PAD + (opts.key ? ICON : 0) + m(col.name, 'name') + ICON;
  const type = col.type.toUpperCase().slice(0, TYPE_CHARS);
  const header = named + (type ? 6 + m(type, 'type') + type.length * 0.3 : 0);
  const font: Font = col.kind === 'number' || col.kind === 'bool' ? 'mono' : 'text';
  const sample = Math.min(rows.length, opts.fit ? 20000 : 200);
  const texts: string[] = [];
  for (let i = 0; i < sample; i++) texts.push(formatCell(rows[i][index]));
  texts.sort((a, b) => b.length - a.length);
  let body = 0;
  for (const t of texts.slice(0, 20)) body = Math.max(body, m(t, typeof rows[0]?.[index] === 'number' ? 'mono' : font));
  const width = Math.min(Math.max(header, body + PAD, 64), max);
  return Math.round(Math.max(width, Math.min(named, 480)));
}
