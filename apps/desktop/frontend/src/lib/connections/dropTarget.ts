import type { ConnDrop } from './groups';

export type Box = { kind: 'head' | 'conn'; id: string; group: string; top: number; bottom: number };
export type SectionBox = { name: string; top: number; bottom: number };

export function connDropAt(y: number, boxes: Box[], dragging: string): { drop: ConnDrop; line: number } | null {
  const list = boxes.filter(b => b.kind === 'head' || b.id !== dragging);
  if (list.length === 0) return null;
  const i = Math.max(0, list.findIndex(b => y < b.bottom));
  const box = y >= list[list.length - 1].bottom ? list[list.length - 1] : list[i];
  const at = list.indexOf(box);
  const next = list[at + 1];
  const nextIn = next?.kind === 'conn' && next.group === box.group ? next.id : null;
  if (box.kind === 'head') return { drop: { group: box.group, before: nextIn }, line: box.bottom };
  if (y < (box.top + box.bottom) / 2) return { drop: { group: box.group, before: box.id }, line: box.top };
  return { drop: { group: box.group, before: nextIn }, line: box.bottom };
}

export function groupDropAt(y: number, secs: SectionBox[], dragging: string): { before: string | null; line: number } | null {
  const list = secs.filter(s => s.name !== dragging);
  if (list.length === 0) return null;
  for (const s of list) {
    if (!s.name || y < (s.top + s.bottom) / 2) return { before: s.name || null, line: s.top };
  }
  return { before: null, line: list[list.length - 1].bottom };
}
