<script lang="ts">
  import { api, type CellValue, type Column, type ColumnChange, type Driver, type ResultColumn, type RowChange } from '../backend';
  import { app } from '../stores/app.svelte';
  import { DEFAULT, TableEdits } from '../stores/edits.svelte';
  import { formatCount, isTrue } from '../format';
  import DataGrid, { type GridEditing, type MenuAt, type MenuItem } from './DataGrid.svelte';
  import Icon from './Icon.svelte';
  import Modal from './Modal.svelte';

  let {
    sessionId,
    schema,
    table,
    driver,
    columns,
    prod,
    readOnlyReason,
    rowChanges,
    queryId,
    onchanged,
    ondirty,
  }: {
    sessionId: string;
    schema: string;
    table: string;
    driver: Driver;
    columns: Column[];
    /** the connection is tagged Production */
    prod: boolean;
    /** why the structure can't be changed, or '' */
    readOnlyReason: string;
    /** uncommitted row edits in the Data tab, which a structure change would drop */
    rowChanges: number;
    /** lets Stop cancel a running change */
    queryId: string;
    /** the table changed: reload its columns and rows */
    onchanged: () => void;
    ondirty: (dirty: boolean) => void;
  } = $props();

  // Columns of the grid.
  const COL_NAME = 0;
  const COL_NULLABLE = 2;
  const COL_DEFAULT = 3;
  const COL_KEY = 4;

  const gridColumns: ResultColumn[] = [
    { name: 'column', type: 'text' },
    { name: 'type', type: 'text' },
    { name: 'nullable', type: 'bool' },
    { name: 'default', type: 'text' },
    { name: 'primary key', type: 'bool' },
  ];
  const rows = $derived(columns.map(c => [c.name, c.type, c.nullable, c.default, c.primaryKey] as CellValue[]));
  const edits = new TableEdits(() => rows, () => gridColumns.length);

  let grid = $state<DataGrid>();
  let saving = $state(false);
  let saveError = $state('');
  let preview = $state<{ sql: string[]; confirm: boolean } | null>(null);

  $effect(() => {
    ondirty(edits.dirty);
  });

  // SQLite's ALTER TABLE can only rename, add and drop columns.
  const sqlite = $derived(driver === 'sqlite');

  const displayRows = $derived.by(() => {
    const out: CellValue[][] = [];
    for (let r = 0; r < edits.rowCount; r++) {
      const row: CellValue[] = [];
      for (let c = 0; c < gridColumns.length; c++) {
        const v = edits.value(r, c);
        row.push(v === DEFAULT ? (c === COL_NULLABLE ? true : c === COL_KEY ? false : null) : (v as CellValue));
      }
      out.push(row);
    }
    return out;
  });

  function canEditCell(r: number, c: number): boolean {
    if (readOnlyReason || edits.rowState(r) === 'deleted' || c === COL_KEY) return false;
    return !sqlite || edits.isNew(r) || c === COL_NAME;
  }

  const editing: GridEditing = {
    canEdit: canEditCell,
    cellState: (r, c) => edits.cellState(r, c),
    rowState: r => edits.rowState(r),
    isFailed: r => edits.isFailed(r),
    commit: (r, c, v) => edits.set(r, c, v),
  };

  function cellMenu(r: number, c: number): MenuItem[] {
    const items: MenuItem[] = [];
    if (c === COL_NULLABLE && canEditCell(r, c)) {
      items.push(isTrue(displayRows[r][c]) ? { id: 'not-null', label: 'Make NOT NULL', kbd: 'Space' } : { id: 'nullable', label: 'Allow NULL', kbd: 'Space' });
    }
    if (c === COL_DEFAULT && canEditCell(r, c) && displayRows[r][c] !== null) {
      items.push({ id: 'null', label: 'Remove the default', kbd: '⌥⌫' });
    }
    if (!readOnlyReason) {
      const rs = grid?.selectedRowsFor(r) ?? [r];
      const label = rs.length > 1 ? `${rs.length} columns` : 'column';
      items.push('sep', rs.every(i => edits.rowState(i) === 'deleted')
        ? { id: 'restore', label: `Restore ${label}`, kbd: '⌘⌫' }
        : { id: 'delete', label: `Drop ${label}`, kbd: '⌘⌫', danger: true });
    }
    return items;
  }

  function onmenu(id: string, at: MenuAt): boolean {
    if (readOnlyReason) {
      if (['delete', 'toggle', 'null'].includes(id)) app.notify(`Can’t change the structure: ${readOnlyReason.toLowerCase()}.`, 'info');
      return false;
    }
    switch (id) {
      case 'delete':
        edits.deleteRows(at.rows);
        return true;
      case 'restore':
        edits.restoreRows(at.rows);
        return true;
    }
    if (at.c < 0 || !canEditCell(at.r, at.c)) return false;
    switch (id) {
      case 'toggle':
        if (at.c !== COL_NULLABLE) return false;
        edits.set(at.r, at.c, !isTrue(displayRows[at.r][at.c]));
        return true;
      case 'nullable':
      case 'not-null':
        edits.set(at.r, COL_NULLABLE, id === 'nullable');
        return true;
      case 'null':
        if (at.c === COL_DEFAULT) {
          edits.set(at.r, at.c, edits.isNew(at.r) ? DEFAULT : null);
          return true;
        }
        app.notify(`${gridColumns[at.c].name} can’t be empty.`, 'info');
        return true;
    }
    return false;
  }

  /** Adds a column row and starts typing its name. */
  export function addColumn() {
    if (readOnlyReason) {
      app.notify(`Can’t add columns: ${readOnlyReason.toLowerCase()}.`, 'info');
      return;
    }
    const r = edits.addRow();
    // A new column is nullable, has no default and isn't part of the key.
    edits.set(r, COL_NULLABLE, true);
    edits.set(r, COL_DEFAULT, null);
    edits.set(r, COL_KEY, false);
    grid?.showRow(r, COL_NAME, true);
  }

  function structureChange(): { changes: ColumnChange[]; rows: number[] } {
    const built = edits.changes(gridColumns, ['column']);
    return { changes: built.changes.map(toColumnChange), rows: built.rows };
  }

  function toColumnChange(ch: RowChange): ColumnChange {
    const v = ch.values;
    const out: ColumnChange = {
      kind: ch.kind,
      column: String(ch.key.column ?? ''),
      defaultSet: 'default' in v,
      default: v.default == null || String(v.default).trim() === '' ? null : String(v.default),
    };
    if ('column' in v) out.name = String(v.column ?? '');
    if ('type' in v) out.type = String(v.type ?? '');
    if ('nullable' in v) out.nullable = isTrue(v.nullable as CellValue);
    return out;
  }

  async function showPreview(confirm: boolean) {
    try {
      preview = { sql: await api.previewStructure(sessionId, { schema, table, changes: structureChange().changes }), confirm };
    } catch (err) {
      saveError = err instanceof Error ? err.message : String(err);
    }
  }

  export function commit() {
    if (!edits.dirty || saving) return;
    if (rowChanges > 0) {
      app.notify(`Commit or discard the ${formatCount(rowChanges, 'row change')} in Data first — changing the structure reloads the rows.`, 'info');
      return;
    }
    if (prod && app.settings.confirmProdWrites) {
      showPreview(true);
      return;
    }
    apply();
  }

  async function apply() {
    preview = null;
    saving = true;
    saveError = '';
    const { changes, rows: changeRows } = structureChange();
    try {
      const res = await api.applyStructure(sessionId, queryId, { schema, table, changes });
      if (res.cancelled && driver === 'mysql') {
        // The server doesn't stop an ALTER TABLE when its client goes away.
        app.notify('Stopped. MySQL may still finish the statement that was running — refresh in a while to see the table as it ends up.', 'info');
        edits.discard();
        onchanged();
        return;
      }
      if (res.cancelled) {
        saveError = 'Stopped — nothing was changed.';
        return;
      }
      if (res.error) {
        if (res.partial) {
          // MySQL kept what ran before the failure: show the table as it is now.
          app.notify(`${formatCount(res.applied, 'change')} applied before one failed — the table now shows what was saved. ${res.error}`);
          edits.discard();
          onchanged();
          return;
        }
        edits.failedRow = res.failedIndex >= 0 ? changeRows[res.failedIndex] : null;
        saveError = `Nothing was changed: ${res.error}`;
        if (edits.failedRow !== null) grid?.showRow(edits.failedRow, -1, false);
        return;
      }
      app.notify(`Saved ${formatCount(res.applied, 'change')} to the structure of ${table}.`, 'info');
      edits.discard();
      onchanged();
    } catch (err) {
      saveError = err instanceof Error ? err.message : String(err);
    } finally {
      saving = false;
    }
  }

  function discard() {
    edits.discard();
    saveError = '';
  }
</script>

<div class="structure">
  <div class="grid-wrap">
    <DataGrid bind:this={grid} columns={gridColumns} rows={displayRows} editing={readOnlyReason ? null : editing} {cellMenu} {onmenu} />
  </div>

  {#if edits.dirty}
    <div class="footer pending" class:has-error={!!saveError}>
      <span class="dot"></span>
      <span class="small strong">{formatCount(edits.count, 'change')}</span>
      {#if saveError}
        <span class="save-error" title={saveError}><Icon name="alert" size={12} />{saveError}</span>
      {:else}
        <span class="small faint">not saved</span>
      {/if}
      <span style="flex:1"></span>
      {#if saving}
        <button class="btn sm" onclick={() => api.cancelQuery(queryId)}><Icon name="stop" size={11} />Stop</button>
      {/if}
      <button class="btn sm ghost" onclick={discard} disabled={saving}>Discard</button>
      <button class="btn sm" onclick={() => showPreview(false)} disabled={saving}><Icon name="code" size={12} />Preview SQL</button>
      <button class="btn sm primary" onclick={commit} disabled={saving}>{saving ? 'Saving…' : 'Commit'}<span class="kbd on-accent">⌘S</span></button>
    </div>
  {:else}
    <div class="footer">
      <span class="small muted">{formatCount(columns.length, 'column')}</span>
      {#if sqlite && !readOnlyReason}<span class="small faint">· SQLite can rename, add and drop columns here; anything else needs the SQL editor</span>{/if}
    </div>
  {/if}
</div>

{#if preview}
  {@const p = preview}
  <Modal title={p.confirm ? 'Change Production?' : 'Pending structure changes'} width={620} onclose={() => (preview = null)}>
    <p class="confirm-text">
      {#if p.confirm}<strong>{table}</strong> is on a connection tagged Production.{/if}
      {#if driver === 'mysql'}
        MySQL applies each statement on its own: if one fails, the ones before it stay applied.
      {:else}
        These statements run in one transaction — all of them apply, or none do.
      {/if}
    </p>
    <div class="sql-list" class:prod={p.confirm}>
      {#each p.sql as stmt, i (i)}<pre>{stmt};</pre>{/each}
    </div>
    {#snippet footer()}
      <span class="faint" style="font-size:11.5px">{formatCount(p.sql.length, 'statement')}</span>
      <span style="flex:1"></span>
      {#if p.confirm}
        <!-- svelte-ignore a11y_autofocus -->
        <button class="btn" autofocus onclick={() => (preview = null)}>Cancel</button>
        <button class="btn primary danger-fill" onclick={apply}>Change Production</button>
      {:else}
        <button class="btn" onclick={() => (preview = null)}>Close</button>
        <button class="btn primary" onclick={() => { preview = null; commit(); }}>Commit</button>
      {/if}
    {/snippet}
  </Modal>
{/if}

<style>
  .structure { height: 100%; display: flex; flex-direction: column; }
  .grid-wrap { flex: 1; min-height: 0; position: relative; }
  .footer {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 36px;
    padding: 0 10px;
    border-top: 1px solid var(--border);
    background: var(--surface);
  }
  .footer.pending .dot { width: 7px; height: 7px; border-radius: 50%; background: var(--warn); }
  .footer.has-error .dot { background: var(--danger); }
  .save-error { display: inline-flex; align-items: center; gap: 5px; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--danger); font-size: 12px; }
  .confirm-text { margin: 0 0 12px; color: var(--text-2); line-height: 1.5; }
  .confirm-text strong { color: var(--text); }
  .sql-list { display: flex; flex-direction: column; gap: 6px; max-height: 300px; overflow: auto; }
  .sql-list pre {
    margin: 0;
    padding: 8px 10px;
    border-radius: 6px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    font-family: var(--font-mono);
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-word;
    user-select: text;
    -webkit-user-select: text;
  }
  .sql-list.prod pre { border-left: 3px solid var(--env-prod); }
</style>
