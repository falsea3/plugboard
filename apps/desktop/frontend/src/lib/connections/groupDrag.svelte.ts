import { createDragGhost, type DragGhost } from '../ui/dragGhost';
import { connDropAt, groupDropAt, type Box } from './dropTarget';
import { moveConnection, moveGroup, type Section } from './groups';

type Options = {
  root: () => HTMLElement | undefined;
  sections: () => Section[];
  apply: (secs: Section[]) => void;
};

type Kind = 'conn' | 'group';

export class GroupDrag {
  dragging = $state<{ kind: Kind; id: string } | null>(null);
  line = $state<number | null>(null);
  private start: { kind: Kind; id: string; pointerId: number; x: number; y: number; source: HTMLElement } | null = null;
  private ghost: DragGhost | null = null;
  private skip = '';

  constructor(private o: Options) {}

  down(e: PointerEvent, kind: Kind, id: string) {
    if (!e.isPrimary || e.button !== 0) return;
    this.skip = '';
    this.start = { kind, id, pointerId: e.pointerId, x: e.clientX, y: e.clientY, source: e.currentTarget as HTMLElement };
  }

  pointermove = (e: PointerEvent) => {
    const s = this.start;
    if (!s || s.pointerId !== e.pointerId) return;
    if (!this.dragging) {
      if (Math.hypot(e.clientX - s.x, e.clientY - s.y) < 5) return;
      this.ghost = createDragGhost(s.source, s.x, s.y);
      this.dragging = { kind: s.kind, id: s.id };
    }
    this.ghost?.move(e.clientX, e.clientY);
    this.target(e.clientY);
  };

  pointerup = (e: PointerEvent) => {
    const s = this.start;
    if (!s || s.pointerId !== e.pointerId) return;
    if (this.dragging) {
      const next = this.target(e.clientY);
      if (next) this.o.apply(next);
      this.skip = `${s.kind}:${s.id}`;
      setTimeout(() => (this.skip = ''));
    }
    this.stop();
  };

  pointercancel = (e: PointerEvent) => {
    if (this.start?.pointerId === e.pointerId) this.stop();
  };

  clicked(e: MouseEvent, kind: Kind, id: string): boolean {
    if (this.skip !== `${kind}:${id}`) return true;
    e.preventDefault();
    this.skip = '';
    return false;
  }

  private stop() {
    this.ghost?.destroy();
    this.ghost = null;
    this.start = null;
    this.dragging = null;
    this.line = null;
  }

  private target(y: number): Section[] | null {
    const root = this.o.root();
    const d = this.dragging;
    if (!root || !d) return null;
    const top = root.getBoundingClientRect().top - root.scrollTop;
    const rel = (r: DOMRect) => ({ top: r.top - top, bottom: r.bottom - top });
    const at = y - top;
    if (d.kind === 'group') {
      const secs = [...root.querySelectorAll<HTMLElement>('[data-section]')].map(el => ({ name: el.dataset.section ?? '', ...rel(el.getBoundingClientRect()) }));
      const hit = groupDropAt(at, secs, d.id);
      this.line = hit?.line ?? null;
      return hit ? moveGroup(this.o.sections(), d.id, hit.before) : null;
    }
    const boxes: Box[] = [...root.querySelectorAll<HTMLElement>('[data-head], [data-conn]')].map(el => ({
      kind: el.dataset.conn !== undefined ? 'conn' : 'head',
      id: el.dataset.conn ?? `g:${el.dataset.head}`,
      group: el.dataset.in ?? el.dataset.head ?? '',
      ...rel(el.getBoundingClientRect()),
    }));
    const hit = connDropAt(at, boxes, d.id);
    this.line = hit?.line ?? null;
    return hit ? moveConnection(this.o.sections(), d.id, hit.drop) : null;
  }
}
