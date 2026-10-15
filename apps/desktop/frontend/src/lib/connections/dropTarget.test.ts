import { describe, expect, it } from 'vitest';
import { connDropAt, groupDropAt, type Box } from './dropTarget';

const boxes: Box[] = [
  { kind: 'head', id: 'g:A', group: 'A', top: 0, bottom: 30 },
  { kind: 'conn', id: 'a1', group: 'A', top: 30, bottom: 70 },
  { kind: 'conn', id: 'a2', group: 'A', top: 70, bottom: 110 },
  { kind: 'head', id: 'g:B', group: 'B', top: 110, bottom: 140 },
  { kind: 'head', id: 'g:', group: '', top: 140, bottom: 170 },
  { kind: 'conn', id: 'x', group: '', top: 170, bottom: 210 },
];

describe('connDropAt', () => {
  it('drops onto a header at the start of its group', () => {
    expect(connDropAt(10, boxes, 'x')).toEqual({ drop: { group: 'A', before: 'a1' }, line: 30 });
    expect(connDropAt(120, boxes, 'x')).toEqual({ drop: { group: 'B', before: null }, line: 140 });
  });

  it('drops before or after a row by its middle', () => {
    expect(connDropAt(40, boxes, 'x')).toEqual({ drop: { group: 'A', before: 'a1' }, line: 30 });
    expect(connDropAt(60, boxes, 'x')).toEqual({ drop: { group: 'A', before: 'a2' }, line: 70 });
    expect(connDropAt(100, boxes, 'x')).toEqual({ drop: { group: 'A', before: null }, line: 110 });
  });

  it('skips the dragged row and clamps past the end', () => {
    expect(connDropAt(80, boxes, 'a1')).toEqual({ drop: { group: 'A', before: 'a2' }, line: 70 });
    expect(connDropAt(500, boxes, 'a1')).toEqual({ drop: { group: '', before: null }, line: 210 });
    expect(connDropAt(10, [], 'a1')).toBeNull();
  });
});

describe('groupDropAt', () => {
  const secs = [{ name: 'A', top: 0, bottom: 110 }, { name: 'B', top: 110, bottom: 140 }, { name: '', top: 140, bottom: 210 }];
  it('drops before a section by its middle and never after ungrouped', () => {
    expect(groupDropAt(20, secs, 'B')).toEqual({ before: 'A', line: 0 });
    expect(groupDropAt(80, secs, 'B')).toEqual({ before: null, line: 140 });
    expect(groupDropAt(115, secs, 'A')).toEqual({ before: 'B', line: 110 });
    expect(groupDropAt(300, secs.slice(0, 2), 'A')).toEqual({ before: null, line: 140 });
  });
});
