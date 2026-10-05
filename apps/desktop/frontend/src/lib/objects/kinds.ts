import type { DBObject } from '../api/wire';
import type { MenuItem } from '../grid/grid';
import type { IconName } from '../ui/Icon.svelte';

export const GROUPS: { kinds: DBObject['kind'][]; label: string; icon: IconName }[] = [
  { kinds: ['function'], label: 'Functions', icon: 'function' },
  { kinds: ['procedure'], label: 'Procedures', icon: 'procedure' },
  { kinds: ['sequence'], label: 'Sequences', icon: 'sequence' },
  { kinds: ['enum', 'domain'], label: 'Types', icon: 'enum' },
  { kinds: ['trigger'], label: 'Triggers', icon: 'trigger' },
  { kinds: ['event'], label: 'Events', icon: 'event' },
  { kinds: ['extension'], label: 'Extensions', icon: 'extension' },
];

export const iconOf = (o: DBObject): IconName => (o.kind === 'domain' ? 'domain' : (GROUPS.find(g => g.kinds.includes(o.kind))?.icon ?? 'ddl'));

export function hintOf(o: DBObject): string {
  if (o.kind === 'function' || o.kind === 'procedure') return `(${o.detail ?? ''})`;
  if (o.kind === 'trigger') return `on ${o.detail}`;
  if (o.kind === 'extension') return o.detail ? `v${o.detail}` : '';
  if (o.kind === 'domain') return o.detail ?? '';
  if (o.kind === 'enum') return `${o.values?.length ?? 0} values`;
  return '';
}

export const keyOf = (o: DBObject) => `${o.kind}:${o.schema}.${o.name}(${o.detail ?? ''})`;

export function groupObjects(objects: DBObject[], filter: string) {
  const q = filter.trim().toLowerCase();
  const shown = q ? objects.filter(o => o.name.toLowerCase().includes(q)) : objects;
  return GROUPS.map(g => ({ ...g, items: shown.filter(o => g.kinds.includes(o.kind)) })).filter(g => g.items.length > 0);
}

export function objectMenu(): MenuItem[] {
  return [
    { id: 'obj-ddl', label: 'Show DDL' },
    { id: 'obj-query', label: 'Open DDL in new query' },
    'sep',
    { id: 'obj-copy-name', label: 'Copy name' },
  ];
}
