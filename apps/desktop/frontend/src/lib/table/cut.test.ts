import { describe, expect, it } from 'vitest';
import { appendCut, cutCells, cutNote, markCut } from './cut';

describe('cut cells', () => {
  it('marks only the cut cells and leaves the page rows untouched', () => {
    const rows = [[1, 'a'], [2, 'b']];
    const marked = markCut(rows, cutCells([{ row: 1, col: 1, size: 0 }]));
    expect(marked).toEqual([[1, 'a'], [2, 'b…']]);
    expect(rows[1][1]).toBe('b');
    expect(marked[0]).toBe(rows[0]);
    expect(markCut(rows, cutCells(undefined))).toBe(rows);
  });
  it('shifts the markers of an appended chunk', () => {
    expect(appendCut([{ row: 0, col: 1, size: 5 }], [{ row: 2, col: 1, size: 0 }], 300)).toEqual([{ row: 0, col: 1, size: 5 }, { row: 302, col: 1, size: 0 }]);
  });
  it('says how much is shown', () => {
    expect(cutNote(3.5 * (1 << 20), 65536)).toBe('Showing the first 64 KB of 3.5 MB');
    expect(cutNote(0, 65536)).toBe('Showing the first 64 KB — the value is longer');
  });
});
