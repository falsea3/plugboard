export type Drop = { id: string; position: 'before' | 'after'; x: number };
export type Slot = { id: string; left: number; right: number };

export function dropAt(x: number, bounds: { left: number; right: number }, slots: Slot[]): Drop | null {
  if (slots.length === 0) return null;
  if (x <= bounds.left) return { id: slots[0].id, position: 'before', x: bounds.left };
  if (x >= bounds.right) return { id: slots[slots.length - 1].id, position: 'after', x: bounds.right - 2 };
  let last: Slot | undefined;
  for (const s of slots) {
    if (s.right <= bounds.left || s.left >= bounds.right) continue;
    if (x < s.left + (s.right - s.left) / 2) return { id: s.id, position: 'before', x: Math.max(bounds.left, s.left - 1) };
    last = s;
  }
  return last ? { id: last.id, position: 'after', x: Math.min(bounds.right - 2, last.right - 1) } : null;
}

export function moveItem<T extends { id: string }>(list: T[], id: string, drop: Drop): boolean {
  const from = list.findIndex(item => item.id === id);
  if (from < 0 || drop.id === id) return false;
  const [item] = list.splice(from, 1);
  let to = list.findIndex(other => other.id === drop.id);
  if (to < 0) {
    list.splice(from, 0, item);
    return false;
  }
  if (drop.position === 'after') to++;
  list.splice(to, 0, item);
  return to !== from;
}
