import { describe, expect, it } from 'vitest';
import { dropAt, moveItem } from './reorder';

const slots = [
  { id: 'a', left: 0, right: 100 },
  { id: 'b', left: 100, right: 200 },
  { id: 'c', left: 200, right: 300 },
];
const bounds = { left: 0, right: 250 };

describe('dropAt', () => {
  it('drops before the slot whose first half is under the pointer', () => {
    expect(dropAt(120, bounds, slots)).toEqual({ id: 'b', position: 'before', x: 99 });
    expect(dropAt(180, bounds, slots)).toEqual({ id: 'c', position: 'before', x: 199 });
  });

  it('goes to the ends past the edges', () => {
    expect(dropAt(-5, bounds, slots)).toMatchObject({ id: 'a', position: 'before' });
    expect(dropAt(400, bounds, slots)).toMatchObject({ id: 'c', position: 'after' });
  });

  it('drops after the last visible slot and skips hidden ones', () => {
    expect(dropAt(240, { left: 0, right: 250 }, slots.slice(0, 2))).toEqual({ id: 'b', position: 'after', x: 199 });
    expect(dropAt(60, { left: 50, right: 250 }, [{ id: 'x', left: -100, right: 40 }, ...slots.slice(1)])).toMatchObject({ id: 'b', position: 'before' });
  });

  it('has nowhere to drop without slots', () => {
    expect(dropAt(10, bounds, [])).toBeNull();
  });
});

describe('moveItem', () => {
  const ids = (l: { id: string }[]) => l.map(i => i.id).join('');
  const list = () => ['a', 'b', 'c', 'd'].map(id => ({ id }));

  it('moves before and after another item', () => {
    const l = list();
    expect(moveItem(l, 'a', { id: 'c', position: 'after', x: 0 })).toBe(true);
    expect(ids(l)).toBe('bcad');
    expect(moveItem(l, 'd', { id: 'b', position: 'before', x: 0 })).toBe(true);
    expect(ids(l)).toBe('dbca');
  });

  it('leaves the list alone when nothing moves', () => {
    const l = list();
    expect(moveItem(l, 'b', { id: 'c', position: 'before', x: 0 })).toBe(false);
    expect(moveItem(l, 'x', { id: 'c', position: 'before', x: 0 })).toBe(false);
    expect(moveItem(l, 'b', { id: 'gone', position: 'before', x: 0 })).toBe(false);
    expect(ids(l)).toBe('abcd');
  });
});
