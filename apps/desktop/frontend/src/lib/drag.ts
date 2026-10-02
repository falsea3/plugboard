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
