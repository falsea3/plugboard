import type { KeyEdit, KeyItem } from '../api/wire';

export type Change = Partial<KeyEdit> & { op: string };

export type ItemColumn = {
  name: string;
  get: (it: KeyItem) => string;
  edit?: (it: KeyItem, text: string) => Change;
  mono?: boolean;
};

export type AddField = { name: string; label: string; placeholder: string };

const fieldOf = (it: KeyItem) => it.label ?? it.field;

export function itemColumns(type: string): ItemColumn[] {
  switch (type) {
    case 'hash':
      return [
        { name: 'Field', get: fieldOf, edit: (it, text) => ({ op: 'hrename', field: it.field, value: text }) },
        { name: 'Value', get: it => it.value, edit: (it, text) => ({ op: 'hset', field: it.field, value: text }) },
      ];
    case 'list':
      return [
        { name: 'Index', get: it => it.field, mono: true },
        { name: 'Value', get: it => it.value, edit: (it, text) => ({ op: 'lset', index: Number(it.field), old: it.value, value: text }) },
      ];
    case 'set':
      return [{ name: 'Member', get: it => it.value, edit: (it, text) => ({ op: 'sreplace', field: it.field, value: text }) }];
    case 'zset':
      return [
        { name: 'Member', get: it => it.value, edit: (it, text) => ({ op: 'zrename', field: it.field, value: text }) },
        { name: 'Score', get: it => it.score ?? '', edit: (it, text) => ({ op: 'zscore', field: it.field, score: text }), mono: true },
      ];
    case 'stream':
      return [
        { name: 'ID', get: it => it.field, mono: true },
        { name: 'Fields', get: it => it.value, mono: true },
      ];
  }
  return [];
}

export function canEdit(it: KeyItem, col: ItemColumn): boolean {
  return !!col.edit && !it.binary && !it.label;
}

export function removeChange(type: string, it: KeyItem): Change | null {
  switch (type) {
    case 'hash':
      return { op: 'hdel', field: it.field };
    case 'list':
      return it.binary ? null : { op: 'lrem', index: Number(it.field), old: it.value };
    case 'set':
      return { op: 'srem', field: it.field };
    case 'zset':
      return { op: 'zrem', field: it.field };
    case 'stream':
      return { op: 'xdel', field: it.field };
  }
  return null;
}

export function addFields(type: string): AddField[] {
  switch (type) {
    case 'hash':
      return [
        { name: 'field', label: 'Field', placeholder: 'field' },
        { name: 'value', label: 'Value', placeholder: 'value' },
      ];
    case 'list':
      return [{ name: 'value', label: 'Value', placeholder: 'added at the end' }];
    case 'set':
      return [{ name: 'value', label: 'Member', placeholder: 'member' }];
    case 'zset':
      return [
        { name: 'value', label: 'Member', placeholder: 'member' },
        { name: 'score', label: 'Score', placeholder: '0' },
      ];
    case 'stream':
      return [{ name: 'value', label: 'Fields', placeholder: '{"event": "signup"}' }];
  }
  return [];
}

const ADD: Record<string, string> = { hash: 'hadd', list: 'rpush', set: 'sadd', zset: 'zadd', stream: 'xadd' };

export function addChange(type: string, values: Record<string, string>): Change {
  return { op: ADD[type], field: values.field ?? '', value: values.value ?? '', score: values.score || '0' };
}

export function createChange(type: string, values: Record<string, string>): Change {
  return { ...addChange(type, values), op: `create:${type}` };
}

export function patchItems(items: KeyItem[], ch: Change): KeyItem[] | null {
  const at = (field?: string) => items.findIndex(it => it.field === field);
  const i = at(ch.field);
  const replace = (patch: Partial<KeyItem>) => items.map((it, n) => (n === i ? { ...it, ...patch } : it));
  switch (ch.op) {
    case 'hset':
      return replace({ value: ch.value });
    case 'hrename':
      return replace({ field: ch.value, label: undefined });
    case 'sreplace':
    case 'zrename':
      return replace({ field: ch.value, value: ch.value });
    case 'zscore':
      return replace({ score: ch.score?.trim() });
    case 'lset':
      return items.map(it => (it.field === String(ch.index) ? { ...it, value: ch.value ?? '' } : it));
    case 'hdel':
    case 'srem':
    case 'zrem':
    case 'xdel':
      return items.filter((_, n) => n !== i);
    case 'hadd':
      return [...items, { field: ch.field ?? '', value: ch.value ?? '' }];
    case 'sadd':
      return [...items, { field: ch.value ?? '', value: ch.value ?? '' }];
  }
  return null;
}

export function sizeLabel(type: string, size: number): string {
  const n = size.toLocaleString('en-US');
  switch (type) {
    case 'string':
      return size === 1 ? '1 byte' : `${n} bytes`;
    case 'hash':
      return size === 1 ? '1 field' : `${n} fields`;
    case 'stream':
      return size === 1 ? '1 entry' : `${n} entries`;
    case 'set':
    case 'zset':
      return size === 1 ? '1 member' : `${n} members`;
  }
  return size === 1 ? '1 item' : `${n} items`;
}
