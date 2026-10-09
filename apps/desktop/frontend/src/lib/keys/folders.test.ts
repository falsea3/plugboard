import { describe, expect, it } from 'vitest';
import { keyRows, scanPattern } from './folders';

const k = (name: string, type = 'string') => ({ name, key: name, type });
const keys = [k('shop:customer:1', 'hash'), k('shop:customer:2', 'hash'), k('shop:config:currency'), k('stats'), k('shop:events', 'stream')];
const brief = (rows: ReturnType<typeof keyRows>) => rows.map(r => `${'  '.repeat(r.depth)}${r.kind === 'folder' ? `${r.name}/ ${r.count}` : r.name}`);

describe('keyRows', () => {
  it('groups keys into folders by the separator, folders first', () => {
    expect(brief(keyRows(keys, new Set()))).toEqual(['shop/ 4', 'stats']);
    expect(brief(keyRows(keys, new Set(['shop:', 'shop:customer:'])))).toEqual([
      'shop/ 4',
      '  config:currency',
      '  customer/ 2',
      '    1',
      '    2',
      '  events',
      'stats',
    ]);
  });

  it('shows a folder with a single key as that key', () => {
    expect(brief(keyRows([k('a:b:c'), k('x')], new Set()))).toEqual(['a:b:c', 'x']);
  });

  it('keeps a key named like a folder next to it', () => {
    expect(brief(keyRows([k('a'), k('a:1'), k('a:2')], new Set(['a:'])))).toEqual(['a/ 2', '  1', '  2', 'a']);
  });

  it('lists flat without a separator', () => {
    expect(brief(keyRows(keys.slice(0, 2), new Set(), ''))).toEqual(['shop:customer:1', 'shop:customer:2']);
  });
});

describe('scanPattern', () => {
  it('matches anywhere unless the filter is already a pattern', () => {
    expect(scanPattern('')).toBe('*');
    expect(scanPattern(' user ')).toBe('*user*');
    expect(scanPattern('user:*')).toBe('user:*');
    expect(scanPattern('a\\b')).toBe('*a\\\\b*');
  });
});
