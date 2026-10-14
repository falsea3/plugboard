import type { CellValue, CutCell } from '../api/backend';

export const CUT_MARK = '…';

export const cutKey = (r: number, c: number) => `${r}:${c}`;

export function cutCells(cut: CutCell[] | undefined): Map<string, number> {
  return new Map((cut ?? []).map(c => [cutKey(c.row, c.col), c.size]));
}

export function markCut(rows: CellValue[][], cut: Map<string, number>, skip: (r: number, c: number) => boolean = () => false): CellValue[][] {
  if (cut.size === 0) return rows;
  const out = rows.slice();
  for (const key of cut.keys()) {
    const [r, c] = key.split(':').map(Number);
    if (!out[r] || typeof out[r][c] !== 'string' || skip(r, c)) continue;
    if (out[r] === rows[r]) out[r] = rows[r].slice();
    out[r][c] = `${out[r][c]}${CUT_MARK}`;
  }
  return out;
}

export function appendCut(before: CutCell[] | undefined, more: CutCell[] | undefined, offset: number): CutCell[] {
  return [...(before ?? []), ...(more ?? []).map(c => ({ ...c, row: c.row + offset }))];
}

const mb = (n: number) => (n >= 1 << 20 ? `${(n / (1 << 20)).toFixed(1)} MB` : `${Math.ceil(n / 1024)} KB`);

export function cutNote(size: number, shown: number): string {
  return size > 0 ? `Showing the first ${mb(shown)} of ${mb(size)}` : `Showing the first ${mb(shown)} — the value is longer`;
}
