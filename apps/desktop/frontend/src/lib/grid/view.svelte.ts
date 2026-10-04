import { flushSync } from 'svelte';
import { clamp } from './grid';
import { startDrag } from '../ui/drag';

export const ROW_H = 26;
export const HEADER_H = 30;
export const OVERSCAN = 8;

export class GridView {
  el = $state<HTMLDivElement>();
  top = $state(0);
  left = $state(0);
  height = $state(400);
  width = $state(800);
  widths = $state<number[]>([]);
  first = $derived(Math.max(0, Math.floor(this.top / ROW_H) - OVERSCAN));
  starts = $derived.by(() => {
    const out = [0];
    for (const w of this.widths) out.push(out[out.length - 1] + w);
    return out;
  });
  shownCols = $derived.by(() => {
    const from = this.left - this.width;
    const to = this.left + 2 * this.width;
    const out: number[] = [];
    for (let c = 0; c < this.widths.length; c++) {
      if (this.starts[c + 1] > from && this.starts[c] < to) out.push(c);
    }
    return out;
  });
  leftPad = $derived(this.shownCols.length ? this.starts[this.shownCols[0]] : 0);

  end(rows: number): number {
    return Math.min(rows, Math.ceil((this.top + this.height) / ROW_H) + OVERSCAN);
  }

  listen(skip: (e: WheelEvent) => boolean) {
    const el = this.el;
    if (!el) return;
    const onwheel = (e: WheelEvent) => !skip(e) && this.wheel(e);
    el.addEventListener('wheel', onwheel, { passive: false });
    return () => el.removeEventListener('wheel', onwheel);
  }

  resize(e: PointerEvent, i: number) {
    e.stopPropagation();
    const w0 = this.widths[i];
    startDrag(e, 'col-resize', dx => (this.widths[i] = Math.max(48, w0 + dx)));
  }

  to(top: number, left = this.el?.scrollLeft ?? 0) {
    const el = this.el;
    if (!el) return;
    el.scrollTop = clamp(top, 0, el.scrollHeight - el.clientHeight);
    el.scrollLeft = clamp(left, 0, el.scrollWidth - el.clientWidth);
    this.sync();
    flushSync();
  }

  sync() {
    this.top = this.el?.scrollTop ?? 0;
    this.left = this.el?.scrollLeft ?? 0;
  }

  by(rows: number) {
    if (this.el) this.to(this.el.scrollTop + rows * ROW_H);
  }

  edge(where: 'top' | 'bottom') {
    if (this.el) this.to(where === 'top' ? 0 : this.el.scrollHeight);
  }

  pageRows(): number {
    return Math.max(1, Math.floor(((this.el?.clientHeight ?? this.height) - HEADER_H) / ROW_H) - 1);
  }

  private wheel(e: WheelEvent) {
    const el = this.el;
    if (!el) return;
    const unit = e.deltaMode === 1 ? ROW_H : e.deltaMode === 2 ? el.clientHeight - HEADER_H : 1;
    let dx = e.deltaX * unit;
    let dy = e.deltaY * unit;
    if (e.shiftKey && dx === 0) [dx, dy] = [dy, 0];
    e.preventDefault();
    this.to(el.scrollTop + dy, el.scrollLeft + dx);
  }

  show(row: number, col: number, gutter: number) {
    const el = this.el;
    if (!el) return;
    let top = el.scrollTop;
    const rowTop = row * ROW_H;
    const viewH = el.clientHeight - HEADER_H;
    if (rowTop < top) top = rowTop;
    else if (rowTop + ROW_H > top + viewH) top = rowTop + ROW_H - viewH;
    let left = el.scrollLeft;
    if (col >= 0) {
      const cellLeft = this.starts[col];
      const viewW = el.clientWidth - gutter;
      if (cellLeft < left) left = cellLeft;
      else if (cellLeft + this.widths[col] > left + viewW) left = cellLeft + this.widths[col] - viewW;
    }
    this.to(top, left);
  }
}
