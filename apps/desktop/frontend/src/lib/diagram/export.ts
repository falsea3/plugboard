import { bounds, HEAD_H, manyMark, oneMark, ROW_H, type Card, type Link, type Point } from './layout';

export type Palette = { bg: string; surface: string; head: string; border: string; text: string; faint: string; line: string; mono: string; ui: string };

const PAD = 32;

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

function card(c: Card, at: Point, p: Palette, id: number): string {
  const clip = `c${id}`;
  const rows = c.columns.map((col, i) => {
    const y = HEAD_H + i * ROW_H + ROW_H / 2 + 4;
    const mark = col.primaryKey ? 'PK' : c.foreign.has(col.name) ? 'FK' : '';
    return (
      (mark ? `<text x="7" y="${y - 1}" fill="${p.faint}" font-size="8" font-family="${p.ui}" font-weight="600">${mark}</text>` : '') +
      `<text x="26" y="${y}" fill="${p.text}" font-size="12" font-family="${p.mono}">${esc(col.name)}</text>` +
      `<text x="${c.w - 10}" y="${y}" fill="${p.faint}" font-size="11" font-family="${p.mono}" text-anchor="end">${esc(col.type)}</text>`
    );
  });
  if (c.hidden > 0) {
    const y = HEAD_H + c.columns.length * ROW_H + ROW_H / 2 + 4;
    rows.push(`<text x="26" y="${y}" fill="${p.faint}" font-size="12" font-style="italic" font-family="${p.mono}">+${c.hidden} more</text>`);
  }
  return (
    `<g transform="translate(${at.x} ${at.y})">` +
    `<clipPath id="${clip}"><rect width="${c.w}" height="${c.h}" rx="8"/></clipPath>` +
    `<g clip-path="url(#${clip})"><rect width="${c.w}" height="${c.h}" fill="${p.surface}"/><rect width="${c.w}" height="${HEAD_H}" fill="${p.head}"/></g>` +
    `<line x1="0" y1="${HEAD_H}" x2="${c.w}" y2="${HEAD_H}" stroke="${p.border}"/>` +
    `<rect x="0.5" y="0.5" width="${c.w - 1}" height="${c.h - 1}" rx="8" fill="none" stroke="${p.border}"/>` +
    `<text x="12" y="${HEAD_H / 2 + 4.5}" fill="${p.text}" font-size="12.5" font-weight="600" font-family="${p.ui}">${esc(c.table.name)}</text>` +
    rows.join('') +
    `</g>`
  );
}

export function diagramSvg(cards: Map<string, Card>, pos: Map<string, Point>, links: Link[], p: Palette): { svg: string; w: number; h: number } {
  const b = bounds(cards, pos);
  const w = Math.ceil(b.w + PAD * 2);
  const h = Math.ceil(b.h + PAD * 2);
  const lines = links
    .map(l => `<path d="${l.path}"/><path d="${manyMark(l.from, l.fromDir)}"/><path d="${oneMark(l.to, l.toDir)}"/>`)
    .join('');
  let id = 0;
  const boxes = [...pos].flatMap(([name, at]) => {
    const c = cards.get(name);
    return c ? [card(c, at, p, id++)] : [];
  });
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}">` +
    `<rect width="${w}" height="${h}" fill="${p.bg}"/>` +
    `<g transform="translate(${PAD - b.x} ${PAD - b.y})">` +
    `<g fill="none" stroke="${p.line}" stroke-width="1.4">${lines}</g>` +
    boxes.join('') +
    `</g></svg>`;
  return { svg, w, h };
}

export function themePalette(): Palette {
  const css = getComputedStyle(document.documentElement);
  const v = (name: string) => css.getPropertyValue(name).trim();
  return {
    bg: v('--bg'), surface: v('--surface'), head: v('--elevated'), border: v('--border'), text: v('--text'), faint: v('--text-3'),
    line: v('--text-3'), mono: esc(v('--font-mono') || 'monospace'), ui: esc(v('--font-ui') || 'sans-serif'),
  };
}

function base64(bytes: Uint8Array): string {
  let s = '';
  for (let i = 0; i < bytes.length; i += 0x8000) s += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(s);
}

export const svgBase64 = (svg: string) => base64(new TextEncoder().encode(svg));

export async function pngBase64(svg: string, w: number, h: number, scale = 2): Promise<string> {
  const url = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml' }));
  try {
    const img = new Image();
    img.src = url;
    await img.decode();
    const canvas = document.createElement('canvas');
    canvas.width = Math.round(w * scale);
    canvas.height = Math.round(h * scale);
    const ctx = canvas.getContext('2d')!;
    ctx.scale(scale, scale);
    ctx.drawImage(img, 0, 0, w, h);
    return canvas.toDataURL('image/png').slice('data:image/png;base64,'.length);
  } finally {
    URL.revokeObjectURL(url);
  }
}
