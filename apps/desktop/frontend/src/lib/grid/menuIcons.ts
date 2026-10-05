import type { IconName } from '../ui/Icon.svelte';

const BY_ID: Record<string, IconName> = {
  edit: 'pencil',
  null: 'pencil',
  'set-true': 'pencil',
  'set-false': 'pencil',
  'set-now': 'pencil',
  'set-default': 'pencil',
  nullable: 'pencil',
  'not-null': 'pencil',
  'copy-value': 'copy',
  'copy-rows': 'copy',
  'copy-name': 'copy',
  'sql-copy': 'copy',
  'sql-open': 'code',
  'filter-value': 'funnel',
  'exclude-value': 'funnel',
  follow: 'follow',
  json: 'json',
  delete: 'trash',
  restore: 'restore',
  'sort-asc': 'arrowUp',
  'sort-desc': 'arrowDown',
  'sort-default': 'sortDefault',
  fit: 'fitWidth',
  'tree-open': 'table',
  'tree-structure': 'columns',
  'tree-query': 'code',
  'tree-diagram': 'diagram',
  'tree-copy-name': 'copy',
  'tree-copy-select': 'copy',
  'tree-rename': 'pencil',
  'tree-truncate': 'trash',
  'tree-drop': 'trash',
  'tree-ddl': 'ddl',
  'tree-type': 'pencil',
  'obj-ddl': 'ddl',
  'obj-query': 'code',
  'obj-copy-name': 'copy',
};

export function menuIcon(id: string): IconName | undefined {
  if (id.startsWith('f-') || id.startsWith('fv-')) return 'funnel';
  return BY_ID[id];
}
