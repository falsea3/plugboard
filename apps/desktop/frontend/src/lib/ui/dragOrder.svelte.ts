import { createDragGhost, type DragGhost } from './dragGhost';
import { dropAt, type Drop } from './reorder';

type Options = {
  list: () => HTMLElement | undefined;
  attr: string;
  ids: () => string[];
  move: (id: string, drop: Drop) => void;
  edgeScroll?: boolean;
};

export class DragOrder {
  dragging = $state('');
  indicator = $state<{ x: number; top: number; height: number } | null>(null);
  private start: { id: string; pointerId: number; x: number; y: number } | null = null;
  private ghost: DragGhost | null = null;
  private skipClick = '';

  constructor(private o: Options) {}

  down(e: PointerEvent, id: string) {
    if (!e.isPrimary || e.button !== 0) return;
    this.skipClick = '';
    this.start = { id, pointerId: e.pointerId, x: e.clientX, y: e.clientY };
  }

  pointermove = (e: PointerEvent) => {
    const s = this.start;
    if (!s || s.pointerId !== e.pointerId) return;
    if (!this.dragging) {
      if (Math.hypot(e.clientX - s.x, e.clientY - s.y) < 5) return;
      const source = this.o.list()?.querySelector<HTMLElement>(`[data-${this.o.attr}="${s.id}"]`);
      if (source) this.ghost = createDragGhost(source, s.x, s.y);
      this.dragging = s.id;
    }
    this.ghost?.move(e.clientX, e.clientY);
    this.dropAt(e.clientX);
  };

  pointerup = (e: PointerEvent) => {
    const s = this.start;
    if (!s || s.pointerId !== e.pointerId) return;
    if (this.dragging === s.id) {
      const drop = this.dropAt(e.clientX);
      if (drop) this.o.move(s.id, drop);
      this.skipClick = s.id;
      setTimeout(() => (this.skipClick = ''));
    }
    this.stop();
  };

  pointercancel = (e: PointerEvent) => {
    if (this.start?.pointerId === e.pointerId) this.stop();
  };

  clicked(e: MouseEvent, id: string): boolean {
    if (this.skipClick !== id) return true;
    e.preventDefault();
    this.skipClick = '';
    return false;
  }

  private stop() {
    this.ghost?.destroy();
    this.ghost = null;
    this.start = null;
    this.dragging = '';
    this.indicator = null;
  }

  private dropAt(x: number): Drop | null {
    const list = this.o.list();
    if (!list) return null;
    const b = list.getBoundingClientRect();
    if (this.o.edgeScroll && x <= b.left) list.scrollLeft = 0;
    if (this.o.edgeScroll && x >= b.right) list.scrollLeft = list.scrollWidth;
    const nodes = new Map<string, HTMLElement>();
    for (const node of list.querySelectorAll<HTMLElement>(`[data-${this.o.attr}]`)) nodes.set(node.dataset[this.o.attr] ?? '', node);
    const slots = this.o.ids().flatMap(id => {
      const r = id === this.dragging ? undefined : nodes.get(id)?.getBoundingClientRect();
      return r ? [{ id, left: r.left, right: r.right }] : [];
    });
    const drop = dropAt(x, b, slots);
    this.indicator = drop ? { x: drop.x, top: b.top + 2, height: b.height - 4 } : null;
    return drop;
  }
}
