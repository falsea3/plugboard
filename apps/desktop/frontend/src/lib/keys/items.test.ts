import { describe, expect, it } from 'vitest';
import { addChange, canEdit, createChange, itemColumns, patchItems, removeChange, sizeLabel } from './items';

const h = [
  { field: 'a', value: '1' },
  { field: 'b', value: '2' },
];

describe('itemColumns', () => {
  it('turns a cell edit into the change for its type', () => {
    const [field, value] = itemColumns('hash');
    expect(field.edit!(h[0], 'aa')).toEqual({ op: 'hrename', field: 'a', value: 'aa' });
    expect(value.edit!(h[0], '9')).toEqual({ op: 'hset', field: 'a', value: '9' });
    const [, listValue] = itemColumns('list');
    expect(listValue.edit!({ field: '3', value: 'x' }, 'y')).toEqual({ op: 'lset', index: 3, old: 'x', value: 'y' });
    expect(itemColumns('zset')[1].edit!({ field: 'm', value: 'm', score: '1' }, '2')).toEqual({ op: 'zscore', field: 'm', score: '2' });
    expect(itemColumns('stream').some(c => c.edit)).toBe(false);
  });

  it('won’t edit binary values or fields', () => {
    const [field, value] = itemColumns('hash');
    expect(canEdit({ field: 'a', value: '\\x00', binary: true }, value)).toBe(false);
    expect(canEdit({ field: '\0hex:ff', label: '\\xff', value: 'v' }, field)).toBe(false);
    expect(canEdit(h[0], field)).toBe(true);
  });
});

describe('removeChange', () => {
  it('removes by field, or by index and the value it had', () => {
    expect(removeChange('hash', h[0])).toEqual({ op: 'hdel', field: 'a' });
    expect(removeChange('list', { field: '2', value: 'x' })).toEqual({ op: 'lrem', index: 2, old: 'x' });
    expect(removeChange('list', { field: '2', value: '\\x00', binary: true })).toBeNull();
    expect(removeChange('string', h[0])).toBeNull();
  });
});

describe('addChange and createChange', () => {
  it('maps the add form to the type’s command', () => {
    expect(addChange('zset', { value: 'm', score: '' })).toEqual({ op: 'zadd', field: '', value: 'm', score: '0' });
    expect(createChange('hash', { field: 'f', value: 'v' }).op).toBe('create:hash');
  });
});

describe('patchItems', () => {
  it('applies a change to the loaded items', () => {
    expect(patchItems(h, { op: 'hset', field: 'b', value: '20' })![1].value).toBe('20');
    expect(patchItems(h, { op: 'hrename', field: 'a', value: 'z' })![0].field).toBe('z');
    expect(patchItems(h, { op: 'hdel', field: 'a' })).toEqual([h[1]]);
    expect(patchItems(h, { op: 'hadd', field: 'c', value: '3' })).toHaveLength(3);
    expect(patchItems([{ field: '0', value: 'x' }], { op: 'lset', index: 0, value: 'y' })![0].value).toBe('y');
  });

  it('asks for a reload when the positions move', () => {
    expect(patchItems(h, { op: 'lrem', index: 0 })).toBeNull();
    expect(patchItems(h, { op: 'rpush', value: 'x' })).toBeNull();
  });
});

it('sizeLabel names what the type holds', () => {
  expect(sizeLabel('string', 1)).toBe('1 byte');
  expect(sizeLabel('hash', 1200)).toBe('1,200 fields');
  expect(sizeLabel('list', 2)).toBe('2 items');
});
