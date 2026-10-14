import type { Card, Point } from './layout';

const LAYER_GAP = 120;
const NODE_GAP = 36;
const BAND_LAYERS = 24;

export function layered(names: string[], edges: [string, string][], cards: Map<string, Card>): Map<string, Point> {
  const into = new Map<string, number>(names.map(n => [n, 0]));
  const next = new Map<string, string[]>(names.map(n => [n, []]));
  for (const [from, to] of edges) {
    if (from === to || !into.has(from) || !into.has(to)) continue;
    next.get(from)!.push(to);
    into.set(to, into.get(to)! + 1);
  }
  const layer = new Map<string, number>();
  const queue = names.filter(n => into.get(n) === 0);
  for (const n of queue) layer.set(n, 0);
  for (let i = 0; i < queue.length; i++) {
    const n = queue[i];
    for (const m of next.get(n)!) {
      layer.set(m, Math.max(layer.get(m) ?? 0, layer.get(n)! + 1));
      into.set(m, into.get(m)! - 1);
      if (into.get(m) === 0) queue.push(m);
    }
  }
  let top = 0;
  for (const v of layer.values()) top = Math.max(top, v);
  for (const n of names) if (!layer.has(n)) layer.set(n, ++top);

  const byLayer: string[][] = [];
  for (const n of names) (byLayer[layer.get(n)!] ??= []).push(n);
  const out = new Map<string, Point>();
  let bandY = 0;
  for (let start = 0; start < byLayer.length; start += BAND_LAYERS) {
    let x = 0;
    let bandH = 0;
    for (const group of byLayer.slice(start, start + BAND_LAYERS)) {
      if (!group) continue;
      let y = bandY;
      let w = 0;
      for (const n of group) {
        const c = cards.get(n)!;
        out.set(n, { x, y });
        y += c.h + NODE_GAP;
        w = Math.max(w, c.w);
      }
      bandH = Math.max(bandH, y - bandY);
      x += w + LAYER_GAP;
    }
    bandY += bandH + LAYER_GAP;
  }
  return out;
}

export type View = { x: number; y: number; k: number; width: number; height: number };

export function inView(cards: Map<string, Card>, pos: Map<string, Point>, v: View, margin = 200): Set<string> {
  const left = (-v.x - margin) / v.k;
  const top = (-v.y - margin) / v.k;
  const right = (v.width - v.x + margin) / v.k;
  const bottom = (v.height - v.y + margin) / v.k;
  const out = new Set<string>();
  for (const [name, p] of pos) {
    const c = cards.get(name);
    if (c && p.x < right && p.x + c.w > left && p.y < bottom && p.y + c.h > top) out.add(name);
  }
  return out;
}

export function lineInView(from: Point, to: Point, v: View, margin = 200): boolean {
  const left = (-v.x - margin) / v.k;
  const top = (-v.y - margin) / v.k;
  const right = (v.width - v.x + margin) / v.k;
  const bottom = (v.height - v.y + margin) / v.k;
  return Math.min(from.x, to.x) < right && Math.max(from.x, to.x) > left && Math.min(from.y, to.y) < bottom && Math.max(from.y, to.y) > top;
}
