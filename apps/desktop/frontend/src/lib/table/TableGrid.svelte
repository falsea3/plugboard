<script lang="ts">
  import { untrack } from 'svelte';
  import { api, type CellValue, type Column, type ResultColumn } from '../api/backend';
  import { app } from '../app/app.svelte';
  import type { TableTab, Workspace } from '../app/workspace.svelte';
  import { DEFAULT, NOW, SET_DEFAULT, exprLabel, type TableEdits } from './edits.svelte';
  import type { TableRows } from './rows.svelte';
  import { isTrue } from '../ui/format';
  import { rowsStatement, type RowStatement } from '../sql/generate';
  import { copyToClipboard } from '../ui/clipboard';
  import { cellMenu as buildCellMenu, filterPick, headerMenu as buildHeaderMenu, type FilterPick } from './menus';
  import { jumpTarget, linkedColumns } from './follow';
  import type { GridEditing, MenuAt, MenuItem } from '../grid/grid';
  import { looksLikeJSON } from '../json/text';
  import type { Relation } from '../api/wire';
  import DataGrid from '../grid/DataGrid.svelte';
  import TableStatus from './TableStatus.svelte';
  import ValuePanel from './ValuePanel.svelte';
  import SqlPreview from './SqlPreview.svelte';
  import ValueEditor, { type ValueEdit } from './ValueEditor.svelte';
  import { TableSave } from './save.svelte';
  import { cutCells, cutKey, cutNote, markCut } from './cut';

  type Sort = { column: string; desc: boolean } | null;

  let {
    ws,
    tab,
    rows,
    edits,
    columns,
    keyColumns,
    relations,
    canEdit,
    readOnlyReason,
    sort,
    customSort,
    filtered,
    onsort,
    onsortby,
    onfilter,
  }: {
    ws: Workspace;
    tab: TableTab;
    rows: TableRows;
    edits: TableEdits;
    columns: Column[];
    keyColumns: string[];
    relations: Relation[];
    canEdit: boolean;
    readOnlyReason: string;
    sort: Sort;
    customSort: boolean;
    filtered: boolean;
    onsort: (column: string) => void;
    onsortby: (sort: Sort) => void;
    onfilter: (column: string, pick: FilterPick) => void;
  } = $props();

  const sessionId = untrack(() => ws.session.sessionId);
  const features = untrack(() => ws.session.engine);

  let grid = $state<DataGrid>();
  const save = untrack(() => new TableSave({ ws, tab, edits, rows, columns: () => page.result.columns, keyColumns: () => keyColumns, showRow: r => grid?.showRow(r, -1, false) }));
  let valueEdit = $state<ValueEdit | null>(null);

  const page = $derived(rows.page!);
  const names = $derived(page.result.columns.map(c => c.name));
  const binary = $derived(new Set(columns.filter(c => c.kind === 'binary').map(c => c.name)));
  const canFindRows = $derived(keyColumns.length > 0 && keyColumns.every(k => !binary.has(k)));
  const info = $derived(new Map(columns.map(c => [c.name, c])));
  const colAt = (c: number): Column | undefined => info.get(names[c]);
  const cut = $derived(cutCells(page.result.cut));
  const isCut = (r: number, c: number) => cut.has(cutKey(r, c)) && edits.cellState(r, c) === '';
  const displayRows = $derived(markCut(edits.shown(v => (v === DEFAULT ? null : exprLabel(v))), cut, (r, c) => edits.cellState(r, c) !== ''));
  const links = $derived(linkedColumns(relations, names));
  const boolValue = (on: boolean): CellValue => (features.booleanType ? on : on ? 1 : 0);

  const baseEditable = (r: number, c: number) => canEdit && edits.rowState(r) !== 'deleted' && !binary.has(names[c]);
  const cellEditable = (r: number, c: number) => baseEditable(r, c) && !isCut(r, c);
  const keyOf = (r: number) => Object.fromEntries(keyColumns.map(k => [k, page.result.rows[r][names.indexOf(k)]]));
  const fullValue = (r: number, c: number) => api.readCell(sessionId, { schema: tab.schema, table: tab.table, column: names[c], key: keyOf(r) });

  const editing: GridEditing = {
    canEdit: cellEditable,
    cellState: (r, c) => edits.cellState(r, c),
    rowState: r => edits.rowState(r),
    isFailed: r => edits.isFailed(r),
    commit: (r, c, v) => edits.set(r, c, v),
    options: (r, c) => colAt(c)?.enum ?? null,
    open: (r, c) => colAt(c)?.kind === 'json' && (openValue(r, c, true), true),
  };

  const jumpFor = (r: number, c: number) => (r < page.result.rows.length ? jumpTarget(relations, names, page.result.rows[r], c) : null);

  function follow(r: number, c: number) {
    const j = jumpFor(r, c);
    if (j) ws.openTable({ schema: j.rel.refSchema, name: j.rel.refTable, kind: 'table' }, j.filters);
  }

  async function openValue(r: number, c: number, json: boolean) {
    if (c < 0 || binary.has(names[c])) return;
    let value = displayRows[r]?.[c] ?? null;
    if (isCut(r, c)) {
      try {
        value = await fullValue(r, c);
      } catch (err) {
        return app.notify(err);
      }
    }
    valueEdit = { r, c, title: `${tab.table}.${names[c]}`, value, json, readOnly: !baseEditable(r, c) };
  }

  function cellLabel(sel: TableTab['selected'] | undefined) {
    if (!sel?.key) return '';
    return `row ${Number(sel.key.split(':')[0]) + 1} · ${sel.column.name}`;
  }

  const setCell = (r: number, c: number, v: string | null) => (edits.set(r, c, v), grid?.showRow(r, c, false));

  function saveValue(value: string) {
    const e = valueEdit;
    valueEdit = null;
    if (e) setCell(e.r, e.c, value);
  }

  function selection(value: CellValue | undefined, column: ResultColumn | undefined): TableTab['selected'] {
    const at = grid?.cursor();
    if (!column || value === undefined || !at) return null;
    const { row: r, col: c } = at;
    const edit = baseEditable(r, c) ? (v: string | null) => setCell(r, c, v) : undefined;
    const shown = String(page.result.rows[r]?.[c] ?? '').length;
    const partial = isCut(r, c) ? { note: cutNote(cut.get(cutKey(r, c)) ?? 0, shown), load: () => fullValue(r, c) } : undefined;
    return { value, column, key: `${r}:${c}`, nullable: colAt(c)?.nullable ?? true, edit, cut: partial };
  }

  function cellMenu(r: number, c: number): MenuItem[] {
    const col = c >= 0 ? colAt(c) : undefined;
    const isNew = edits.rowState(r) === 'new';
    const picked = grid?.selectedRowsFor(r) ?? [r];
    const cell = c >= 0 && {
      col,
      name: names[c],
      value: displayRows[r]?.[c] ?? null,
      editable: cellEditable(r, c),
      isNew,
      canSetDefault: !!col && (isNew ? edits.cellState(r, c) !== 'default' : col.default !== null && features.canUpdateToDefault),
      json: col?.kind === 'json' || looksLikeJSON(displayRows[r]?.[c]),
      followTo: jumpFor(r, c)?.rel.refTable ?? null,
    };
    return buildCellMenu({
      cell: cell || null,
      sqlRows: picked.filter(i => edits.rowState(i) !== 'new').length,
      canFindRows,
      editRows: canEdit ? { count: picked.length, allDeleted: picked.every(i => edits.rowState(i) === 'deleted') } : null,
    });
  }

  const headerMenu = (c: number) => buildHeaderMenu({ name: names[c], col: colAt(c), sort, defaultOrder: customSort ? (page.defaultOrder ?? null) : null });

  function rowsSQL(kind: RowStatement, picked: number[]): string {
    const keep = names.flatMap((n, i) => (binary.has(n) ? [] : [i]));
    const target = { schema: tab.schema, table: tab.table, columns: keep.map(i => names[i]), key: keyColumns };
    return rowsStatement(kind, features, target, picked.map(r => keep.map(i => page.result.rows[r][i])));
  }

  function onmenu(id: string, at: MenuAt): boolean {
    const name = at.c >= 0 ? names[at.c] : '';
    const v = at.r >= 0 && at.c >= 0 ? (displayRows[at.r]?.[at.c] ?? null) : null;
    const pick = filterPick(id, v, boolValue);
    if (id === 'sort-asc' || id === 'sort-desc') onsortby({ column: name, desc: id === 'sort-desc' });
    else if (id === 'sort-default') onsortby(null);
    else if (pick) onfilter(name, pick);
    else if (id === 'follow') follow(at.r, at.c);
    else if (id === 'json') openValue(at.r, at.c, true);
    else if (id === 'value') openValue(at.r, at.c, colAt(at.c)?.kind === 'json');
    else if (id.startsWith('sql-copy:') || id.startsWith('sql-open:')) {
      const [to, kind] = id.split(':');
      const picked = (at.rows.length ? at.rows : [at.r]).filter(r => edits.rowState(r) !== 'new');
      if (picked.length === 0) return true;
      if (picked.some(r => names.some((_, c) => isCut(r, c)))) return (app.notify('Some of these values are too long to show whole, so the SQL would cut them — open them, or write the statement in the SQL editor.', 'info'), true);
      const sql = rowsSQL(kind as RowStatement, picked);
      if (to === 'sql-open') ws.newQuery(sql);
      else copyToClipboard(sql);
    } else if (!canEdit) {
      if (readOnlyReason && ['null', 'delete', 'restore', 'toggle'].includes(id)) app.notify(`Can’t edit: ${readOnlyReason.toLowerCase()}.`, 'info');
      return false;
    } else if (id === 'delete') edits.deleteRows(at.rows);
    else if (id === 'restore') edits.restoreRows(at.rows);
    else return editCell(id, at, v);
    return true;
  }

  function editCell(id: string, at: MenuAt, v: CellValue): boolean {
    const col = colAt(at.c);
    if (at.c < 0 || !cellEditable(at.r, at.c) || !col) return false;
    if (id === 'null' && !col.nullable) app.notify(`${col.name} is NOT NULL.`, 'info');
    else if (id === 'null') edits.set(at.r, at.c, null);
    else if (id === 'set-true' || id === 'set-false') edits.set(at.r, at.c, boolValue(id === 'set-true'));
    else if (id === 'toggle' && col.kind === 'bool') edits.set(at.r, at.c, boolValue(v === null ? true : !isTrue(v)));
    else if (id === 'set-now') edits.set(at.r, at.c, NOW);
    else if (id === 'set-default') edits.set(at.r, at.c, edits.rowState(at.r) === 'new' ? DEFAULT : SET_DEFAULT);
    else return false;
    return true;
  }

  export function addRow() {
    if (!canEdit) {
      if (readOnlyReason) app.notify(`Can’t add rows: ${readOnlyReason.toLowerCase()}.`, 'info');
      return;
    }
    const r = edits.addRow();
    const first = names.findIndex(n => !binary.has(n) && !keyColumns.includes(n));
    grid?.showRow(r, Math.max(first, 0), true);
  }

  export function commit() {
    save.commit();
  }

  export function loaded(from: 'start' | 'reload') {
    save.error = '';
    if (from === 'start') grid?.scrollToTop();
  }

</script>

<div class="body">
<div class="grid-area">
  <DataGrid
    bind:this={grid}
    columns={page.result.columns}
    rows={displayRows}
    onend={() => edits.rowCount <= rows.loaded && rows.more()}
    {links}
    onfollow={follow}
    {sort}
    {keyColumns}
    {onsort}
    editing={canEdit ? editing : null}
    {cellMenu}
    {headerMenu}
    {onmenu}
    onselect={(value, column) => ws.selectCell(tab, selection(value, column))}
  />
</div>
{#if ws.valueShown}<ValuePanel selected={tab.selected ?? null} width={ws.valueWidth} onclose={() => (ws.valueShown = false)} />{/if}
</div>
<TableStatus
  {rows}
  {filtered}
  cell={cellLabel(tab.selected)}
  pending={edits.dirty ? { count: edits.count, error: save.error, saving: save.saving } : null}
  valueShown={ws.valueShown}
  ontogglevalue={() => (ws.valueShown = !ws.valueShown)}
  ondiscard={() => save.discard()}
  onpreview={() => save.showPreview(false)}
/>

{#if valueEdit}
  <ValueEditor edit={valueEdit} onsave={saveValue} onclose={() => (valueEdit = null)} />
{/if}

{#if save.preview}
  {@const p = save.preview}
  <SqlPreview
    title={p.confirm ? 'Commit to Production?' : 'Pending changes'}
    sql={p.sql}
    confirm={p.confirm}
    confirmLabel="Commit to Production"
    onapply={() => save.apply()}
    oncommit={() => ((save.preview = null), commit())}
    onclose={() => (save.preview = null)}
  >
    {#if p.confirm}<strong>{ws.connection.name}</strong> is tagged Production. These statements will run in one transaction:{:else}Commit runs these statements in one transaction — all of them apply, or none do.{/if}
  </SqlPreview>
{/if}

<style>
  .body { flex: 1; min-height: 0; display: flex; gap: 10px; margin-top: 10px; }
  .grid-area { flex: 1; min-width: 0; position: relative; border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--card); overflow: hidden; }
</style>
