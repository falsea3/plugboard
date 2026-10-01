/**
 * Follows a pointer drag started by e, calling move with how far the pointer
 * has gone since, until the button is released. The cursor is set on the whole
 * page meanwhile, so it doesn't flicker as the pointer leaves the handle.
 */
export function startDrag(e: PointerEvent, cursor: string, move: (dx: number, dy: number) => void) {
  e.preventDefault();
  const x0 = e.clientX;
  const y0 = e.clientY;
  const onMove = (ev: PointerEvent) => move(ev.clientX - x0, ev.clientY - y0);
  const onUp = () => {
    window.removeEventListener('pointermove', onMove);
    window.removeEventListener('pointerup', onUp);
    document.body.style.cursor = '';
  };
  document.body.style.cursor = cursor;
  window.addEventListener('pointermove', onMove);
  window.addEventListener('pointerup', onUp);
}
