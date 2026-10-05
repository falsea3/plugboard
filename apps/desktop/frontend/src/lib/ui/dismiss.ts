export function dismiss(node: HTMLElement, onclose: () => void) {
  let close = onclose;
  let fromBackdrop = false;
  const down = (e: PointerEvent) => {
    fromBackdrop = e.target === node;
  };
  const click = (e: MouseEvent) => {
    if (fromBackdrop && e.target === node) close();
    fromBackdrop = false;
  };
  node.addEventListener('pointerdown', down);
  node.addEventListener('click', click);
  return {
    update(next: () => void) {
      close = next;
    },
    destroy() {
      node.removeEventListener('pointerdown', down);
      node.removeEventListener('click', click);
    },
  };
}
