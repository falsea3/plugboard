import { describe, expect, it } from 'vitest';
import { GridSelection } from './selection.svelte';

const range = (from: number, to: number) => Array.from({ length: to - from + 1 }, (_, i) => from + i);

describe('GridSelection.menuAt', () => {
  it('keeps a ⇧ range when the menu opens on a row inside it', () => {
    const sel = new GridSelection();
    sel.point(0, 1);
    sel.point(49, 1, { shiftKey: true, metaKey: false, ctrlKey: false });
    sel.menuAt(30, 2);
    expect(sel.rows()).toEqual(range(0, 49));
  });
  it('keeps ⌘-picked rows', () => {
    const sel = new GridSelection();
    sel.point(3, 0);
    sel.point(7, 0, { shiftKey: false, metaKey: true, ctrlKey: false });
    sel.point(9, 0, { shiftKey: false, metaKey: true, ctrlKey: false });
    sel.menuAt(7, 0);
    expect(sel.rows()).toEqual([3, 7, 9]);
  });
  it('selects just the row when the menu opens outside the selection', () => {
    const sel = new GridSelection();
    sel.point(0, 0);
    sel.point(10, 0, { shiftKey: true, metaKey: false, ctrlKey: false });
    sel.menuAt(20, 3);
    expect(sel.rows()).toEqual([20]);
    expect(sel.cursor).toEqual({ row: 20, col: 3 });
  });
});
