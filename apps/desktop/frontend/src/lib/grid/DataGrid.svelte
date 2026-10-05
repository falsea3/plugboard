<script lang="ts">
  import { tick, untrack } from 'svelte';
  import type { CellValue, ResultColumn } from '../api/wire';
  import { copyText, calcColumnWidth, toTSV } from '../ui/format';
  import { copyToClipboard } from '../ui/clipboard';
  import { gridMenuItems, inCellEditor, keyAction, type GridEditing, type KeyAction, type MenuAt, type MenuItem } from './grid';
  import { GridSelection } from './selection.svelte';
  import { GridView, HEADER_H, OVERSCAN, ROW_H } from './view.svelte';
  import { GridEditor } from './editor.svelte';
  import GridHeader from './GridHeader.svelte';
  import GridMenu from './GridMenu.svelte';
  import GridRow from './GridRow.svelte';

  type Sort = { column: string; desc: boolean } | null;

  let {
    columns,
    rows,
    rowOffset = 0,
    sort = null,
    keyColumns = [],
    onsort,
    onselect,
    editing = null,
    cellMenu,
    headerMenu,
    onmenu,
    onend,
    links,
    onfollow,
  }: {
    columns: ResultColumn[];
    rows: CellValue[][];
    rowOffset?: number | null;
    sort?: Sort;
    keyColumns?: string[];
    onsort?: (column: string) => void;
    onselect?: (value: CellValue | undefined, column: ResultColumn | undefined) => void;
    editing?: GridEditing | null;
    cellMenu?: (r: number, c: number) => MenuItem[];
    headerMenu?: (c: number) => MenuItem[];
    onmenu?: (id: string, at: MenuAt) => boolean | void;
    onend?: () => void;
    links?: Set<number>;
    onfollow?: (r: number, c: number) => void;
  } = $props();

  const sel = new GridSelection();
  const view = new GridView();
  const editor = new GridEditor({
    editing: () => editing,
    value: (r, c) => rows[r]?.[c],
    columns: () => columns.length,
    select: (r, c) => {
      sel.cursor = { row: r, col: c };
    },
    show: (r, c) => showCell(r, c),
    focus: () => focusGrid(),
  });
  let menu = $state<{ x: number; y: number; r: number; c: number } | null>(null);

  let shapeKey = '';
  $effect.pre(() => {
    const key = columns.map(c => c.name + ':' + c.type).join('|');
    if (key !== shapeKey) {
      shapeKey = key;
      view.widths = columns.map((c, i) => calcColumnWidth(c.name, c.type, rows, i));
      sel.clear();
    }
  });

  const rowNumberWidth = $derived(Math.max(44, String((rowOffset ?? 0) + rows.length).length * 8 + 20));
  const totalWidth = $derived(rowNumberWidth + view.widths.reduce((a, b) => a + b, 0));
  const end = $derived(view.end(rows.length));
  const visible = $derived(rows.slice(view.first, end));
  const bodyH = $derived(Math.max(0, Math.min(view.height - HEADER_H, rows.length * ROW_H)));
  const numeric = $derived(columns.map(c => c.kind === 'number'));
  const keys = $derived(new Set(keyColumns));

  export function scrollToTop() {
    view.to(0, 0);
  }

  $effect(() => view.listen(e => e.ctrlKey || inCellEditor(e)));

  $effect(() => {
    if (!onend || rows.length === 0) return;
    if (end >= rows.length - Math.max(OVERSCAN, Math.ceil(view.height / ROW_H))) untrack(() => onend());
  });

  $effect(() => {
    if (!onselect) return;
    const at = sel.cursor;
    if (at && at.col >= 0 && rows[at.row]) onselect(rows[at.row][at.col], columns[at.col]);
    else onselect(undefined, undefined);
  });

  const focusGrid = () => view.el?.focus({ preventScroll: true });

  function point(r: number, c: number, e: MouseEvent) {
    if (c < 0 || !editor.at(r, c)) editor.commit();
    sel.point(r, c, e);
    focusGrid();
  }

  export const cursor = () => sel.cursor;

  export function selectedRowsFor(r: number): number[] {
    return sel.rowsFor(r);
  }

  export function showRow(r: number, c: number, startEditing: boolean) {
    sel.only(r, c);
    tick().then(() => {
      showCell(r, c);
      if (startEditing && c >= 0) editor.start(r, c);
      else focusGrid();
    });
  }

  const showCell = (r: number, c: number) => view.show(r, c, rowNumberWidth);

  function sendMenu(id: string): boolean {
    const at = sel.cursor;
    if (!at || !onmenu) return false;
    const handled = onmenu(id, { r: at.row, c: at.col, rows: sel.rows() }) === true;
    focusGrid();
    return handled;
  }

  function openMenu(e: MouseEvent, r: number, c: number) {
    e.preventDefault();
    editor.commit();
    if (r >= 0) sel.menuAt(r, c);
    menu = { x: e.clientX, y: e.clientY, r, c };
  }

  function menuItems(m: { r: number; c: number }): MenuItem[] {
    return gridMenuItems(m, {
      rows: sel.rows().length,
      canEdit: !!editing?.canEdit(m.r, m.c),
      header: () => headerMenu?.(m.c) ?? [],
      cell: () => cellMenu?.(m.r, m.c) ?? [],
    });
  }

  function pick(id: string, { r, c }: { r: number; c: number }) {
    menu = null;
    if (id === 'edit') return editor.start(r, c);
    if (id === 'copy-value') return copyToClipboard(copyText(rows[r][c]));
    if (id === 'copy-rows') return copy('rows');
    if (id === 'fit') return autoFit(c);
    if (id === 'copy-name') return copyToClipboard(columns[c].name);
    onmenu?.(id, { r, c, rows: r === -1 ? [] : sel.rows() });
    focusGrid();
  }

  function copy(whole: 'cell' | 'rows' | 'all') {
    const at = sel.cursor;
    if (whole === 'cell' && at) return copyToClipboard(copyText(rows[at.row][at.col]));
    return copyToClipboard(toTSV(columns.map(c => c.name), whole === 'rows' ? sel.rows().map(r => rows[r]) : rows));
  }

  function page(a: Extract<KeyAction, { kind: 'page' }>) {
    const at = sel.cursor;
    if (a.row === null || !at) {
      if (typeof a.scroll === 'number') view.by(a.scroll);
      else view.edge(a.scroll);
      return;
    }
    sel.only(a.row, at.col);
    if (typeof a.scroll === 'number') view.by(a.scroll);
    showCell(a.row, at.col);
  }

  function onkeydown(e: KeyboardEvent) {
    if (inCellEditor(e)) return;
    const a = keyAction(
      { key: e.key, mod: e.metaKey || e.ctrlKey, shift: e.shiftKey, alt: e.altKey },
      { cursor: sel.cursor, rows: rows.length, cols: columns.length, pageRows: view.pageRows(), editable: !!editing },
    );
    if (!a) return;
    if (a.kind === 'menu' && a.id === 'toggle') {
      if (sendMenu('toggle')) e.preventDefault();
      return;
    }
    e.preventDefault();
    if (a.kind === 'menu' && a.id === 'delete-rows') sendMenu(sel.rows().every(r => editing!.rowState(r) === 'deleted') ? 'restore' : 'delete');
    else if (a.kind === 'menu') sendMenu(a.id);
    else if (a.kind === 'edit' && sel.cursor) editor.start(sel.cursor.row, sel.cursor.col, a.text);
    else if (a.kind === 'copy') copy(a.whole);
    else if (a.kind === 'page') page(a);
    else if (a.kind === 'move') {
      const at = sel.move(a.row, a.col, a.extend);
      showCell(at.row, at.col);
    }
  }

  function autoFit(i: number) {
    const c = columns[i];
    view.widths[i] = calcColumnWidth(c.name, c.type, rows, i, true);
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
  class="grid"
  bind:this={view.el}
  bind:clientHeight={view.height}
  bind:clientWidth={view.width}
  onscroll={() => view.sync()}
  {onkeydown}
  tabindex="0"
  role="grid"
  aria-rowcount={rows.length}
  aria-colcount={columns.length}
>
  <div class="inner" style:width="{totalWidth}px" style:height="{HEADER_H + rows.length * ROW_H}px">
    <GridHeader
      {columns}
      widths={view.widths}
      shown={view.shownCols}
      leftPad={view.leftPad}
      numberWidth={rowNumberWidth}
      height={HEADER_H}
      {sort}
      {keys}
      {numeric}
      {onsort}
      onmenu={(e, c) => openMenu(e, -1, c)}
      onresize={(e, i) => view.resize(e, i)}
      onfit={autoFit}
    />

    <div class="body" style:top="{HEADER_H}px" style:height="{bodyH}px">
      <div class="rows" style:transform="translateY({view.first * ROW_H - view.top}px)">
        {#each visible as row, vi}
          {@const r = view.first + vi}
          <GridRow
            {r}
            {row}
            {columns}
            shown={view.shownCols}
            widths={view.widths}
            leftPad={view.leftPad}
            {numeric}
            numberWidth={rowNumberWidth}
            {rowOffset}
            height={ROW_H}
            {editing}
            cursorCol={sel.cursor?.row === r ? sel.cursor.col : null}
            marked={sel.marked?.has(r) ?? false}
            {links}
            {editor}
            onpoint={(c, e) => point(r, c, e)}
            onmenu={(e, c) => openMenu(e, r, c)}
            onfollow={c => onfollow?.(r, c)}
          />
        {/each}
      </div>
    </div>
  </div>

  {#if menu}
    {@const m = menu}
    <GridMenu items={menuItems(m)} x={m.x} y={m.y} onpick={id => pick(id, m)} onclose={() => (menu = null)} />
  {/if}

  {#if columns.length === 0}
    <div class="empty faint">No columns</div>
  {:else if rows.length === 0}
    <div class="empty faint" style:top="{HEADER_H}px">No rows</div>
  {/if}
</div>

<style>
  .grid {
    position: relative;
    height: 100%;
    overflow: auto;
    outline: none;
    background: var(--bg);
    font-family: var(--font-mono);
    font-size: 12px;
    font-variant-numeric: tabular-nums;
  }
  .inner { position: relative; min-width: 100%; }
  .body { position: sticky; }
  .rows { will-change: transform; }
  .empty {
    position: absolute;
    left: 0;
    right: 0;
    top: 0;
    padding: 28px;
    text-align: center;
    font-family: var(--font-ui);
  }
</style>
