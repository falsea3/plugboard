export type DragGhost = {
  move(x: number, y: number): void;
  destroy(): void;
};

export function createDragGhost(source: HTMLElement, startX: number, startY: number): DragGhost {
  const bounds = source.getBoundingClientRect();
  const offsetX = startX - bounds.left;
  const offsetY = startY - bounds.top;
  const ghost = source.cloneNode(true) as HTMLElement;

  ghost.classList.add('drag-ghost');
  ghost.removeAttribute('data-tab');
  ghost.removeAttribute('data-ws');
  ghost.setAttribute('aria-hidden', 'true');
  ghost.style.position = 'fixed';
  ghost.style.left = `${bounds.left}px`;
  ghost.style.top = `${bounds.top}px`;
  ghost.style.width = `${bounds.width}px`;
  ghost.style.height = `${bounds.height}px`;
  ghost.style.margin = '0';
  ghost.style.opacity = '0.82';
  ghost.style.pointerEvents = 'none';
  ghost.style.zIndex = '10000';
  ghost.style.boxShadow = '0 4px 16px rgba(0, 0, 0, 0.28)';
  ghost.style.willChange = 'left, top';
  ghost.querySelectorAll<HTMLElement>('button').forEach(button => button.setAttribute('tabindex', '-1'));
  document.body.append(ghost);

  return {
    move(x, y) {
      ghost.style.left = `${x - offsetX}px`;
      ghost.style.top = `${y - offsetY}px`;
    },
    destroy() {
      ghost.remove();
    },
  };
}
