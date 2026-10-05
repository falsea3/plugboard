import { describe, expect, it } from 'vitest';
import type { DBObject } from '../api/wire';
import { groupObjects, hintOf, keyOf } from './kinds';

const o = (kind: DBObject['kind'], name: string, more: Partial<DBObject> = {}): DBObject => ({ schema: 'public', name, kind, ...more });

describe('object groups', () => {
  const objects = [o('function', 'add', { detail: 'a integer' }), o('enum', 'mood', { values: ['sad', 'ok'] }), o('domain', 'email', { detail: 'text' }), o('trigger', 'touch', { detail: 'orders' })];

  it('groups by kind, keeps empty groups out and filters by name', () => {
    expect(groupObjects(objects, '').map(g => [g.label, g.items.map(i => i.name)])).toEqual([
      ['Functions', ['add']],
      ['Types', ['mood', 'email']],
      ['Triggers', ['touch']],
    ]);
    expect(groupObjects(objects, 'MO').map(g => g.label)).toEqual(['Types']);
  });

  it('says what each object is at a glance', () => {
    expect(objects.map(hintOf)).toEqual(['(a integer)', '2 values', 'text', 'on orders']);
    expect(keyOf(objects[0])).not.toBe(keyOf(o('function', 'add', { detail: 'a bigint' })));
  });
});
