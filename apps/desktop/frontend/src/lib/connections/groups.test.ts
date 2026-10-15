import { describe, expect, it } from 'vitest';
import type { Connection } from '../api/wire';
import { arrangement, dropGroup, filterSections, groupNameProblem, moveConnection, moveGroup, renameGroup, sections } from './groups';

const conn = (id: string, group = '') => ({ id, name: id, group }) as Connection;
const names = (secs: ReturnType<typeof sections>) => secs.map(s => `${s.name}:${s.items.map(c => c.id).join('')}`);

describe('connection groups', () => {
  const conns = [conn('a', 'Shop'), conn('b'), conn('c', 'Shop'), conn('d', 'Lost')];
  const secs = sections(conns, [{ name: 'Empty', collapsed: true }, { name: 'Shop', collapsed: false }]);

  it('keeps the group order, adds unknown groups, puts ungrouped last', () => {
    expect(names(secs)).toEqual(['Empty:', 'Shop:ac', 'Lost:d', ':b']);
    expect(secs[0].collapsed).toBe(true);
    expect(names(sections([], []))).toEqual([':']);
  });

  it('moves a connection inside and across groups', () => {
    expect(names(moveConnection(secs, 'c', { group: 'Shop', before: 'a' }))).toEqual(['Empty:', 'Shop:ca', 'Lost:d', ':b']);
    const moved = moveConnection(secs, 'b', { group: 'Empty', before: null });
    expect(names(moved)).toEqual(['Empty:b', 'Shop:ac', 'Lost:d', ':']);
    expect(moved[0].items[0].group).toBe('Empty');
    expect(moveConnection(secs, 'a', { group: 'Shop', before: 'a' })).toBe(secs);
  });

  it('moves groups but keeps ungrouped at the end', () => {
    expect(names(moveGroup(secs, 'Lost', 'Empty'))).toEqual(['Lost:d', 'Empty:', 'Shop:ac', ':b']);
    expect(names(moveGroup(secs, 'Empty', null))).toEqual(['Shop:ac', 'Lost:d', 'Empty:', ':b']);
    expect(moveGroup(secs, '', 'Shop')).toBe(secs);
  });

  it('renames and removes groups', () => {
    expect(renameGroup(secs, 'Shop', 'Store')[1].items.map(c => c.group)).toEqual(['Store', 'Store']);
    expect(names(dropGroup(secs, 'Shop'))).toEqual(['Empty:', 'Lost:d', ':bac']);
    expect(groupNameProblem(' Shop ', secs)).toMatch(/exists/);
    expect(groupNameProblem('Shop', secs, 'Shop')).toBe('');
    expect(groupNameProblem(' ', secs)).toBe('Enter a name');
  });

  it('filters without collapsing and builds the arrangement', () => {
    expect(names(filterSections(secs, c => c.id !== 'd'))).toEqual(['Shop:ac', ':b']);
    expect(arrangement(secs)).toEqual({
      groups: [{ name: 'Empty', collapsed: true }, { name: 'Shop', collapsed: false }, { name: 'Lost', collapsed: false }],
      connections: [{ id: 'a', group: 'Shop' }, { id: 'c', group: 'Shop' }, { id: 'd', group: 'Lost' }, { id: 'b', group: '' }],
    });
  });
});
