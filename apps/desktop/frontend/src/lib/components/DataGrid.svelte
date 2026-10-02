<script lang="ts" module>
  import type { CellValue, ResultColumn } from '../wire';

  /** What a parent provides to make the grid editable. Row indexes are display rows. */
  export interface GridEditing {
    canEdit(r: number, c: number): boolean;
    /** 'default': an added row's untouched cell; 'expr': set to a server expression like now() */
    cellState(r: number, c: number): '' | 'edited' | 'default' | 'expr';
    rowState(r: number): '' | 'new' | 'deleted';
    isFailed(r: number): boolean;
    commit(r: number, c: number, value: CellValue): void;
    /** allowed values (enum columns): the cell editor becomes a list */
    options?(r: number, c: number): string[] | null;
  }

  /** A context-menu entry the parent adds; 'sep' draws a divider. */
  export type MenuItem = { id: string; label: string; kbd?: string; danger?: boolean; disabled?: boolean } | 'sep';

  /** Where a menu action applies: r is -1 for the column header menu. */
  export type MenuAt = { r: number; c: number; rows: number[] };
</script>

<script lang="ts">
  import { tick } from 'svelte';
  import { cellKind, copyText, calcColumnWidth, formatCell, isNumericType, toTSV } from '../format';
  import { copyToClipboard } from '../clipboard';
  import { startDrag } from '../drag';
  import Select from './Select.svelte';
  import Icon from './Icon.svelte';

  type Sort = { column: string; desc: boolean } | null;
  type Selection = { row: number; col: number } | { row: number; col: -1 } | null;

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
  }: {
    columns: ResultColumn[];
    rows: CellValue[][];
    /** row number of the first row; null when unknown (row numbers show as ·) */
    rowOffset?: number | null;
    sort?: Sort;
    keyColumns?: string[];
    onsort?: (column: string) => void;
    onselect?: (value: CellValue | undefined, column: ResultColumn | undefined) => void;
    editing?: GridEditing | null;
    /** extra items for a cell (c ≥ 0) or row-number (c = -1) menu */
    cellMenu?: (r: number, c: number) => MenuItem[];
    headerMenu?: (c: number) => MenuItem[];
    /** a parent menu item was picked, or a key bound to one was pressed; true = handled */
    onmenu?: (id: string, at: MenuAt) => boolean | void;
  } = $props();

  const ROW_H = 26;
  const HEADER_H = 30;
  const OVERSCAN = 8;

  let scroller = $state<HTMLDivElement>();
  let scrollTop = $state(0);
  let scrollLeft = $state(0);
  let viewportH = $state(400);
  let viewportW = $state(800);
  let widths = $state<number[]>([]);
  let selection = $state<Selection>(null);
  /** first row of a shift-click row range; the range ends at selection.row */
  let rowAnchor = $state<number | null>(null);
  let cellEditor = $state<{ r: number; c: number; text: string; wasNull: boolean; touched: boolean; options: string[] | null } | null>(null);
  let menu = $state<{ x: number; y: number; r: number; c: number } | null>(null);

  // Re-measure columns whenever a new result shape arrives.
  let shapeKey = '';
  $effect.pre(() => {
    const key = columns.map(c => c.name + ':' + c.type).join('|');
    if (key !== shapeKey) {
      shapeKey = key;
      widths = columns.map((c, i) => calcColumnWidth(c.name, c.type, rows, i));
      selection = null;
    }
  });

  const rowNumberWidth = $derived(Math.max(44, String((rowOffset ?? 0) + rows.length).length * 8 + 20));
  const totalWidth = $derived(rowNumberWidth + widths.reduce((a, b) => a + b, 0));
  const start = $derived(Math.max(0, Math.floor(scrollTop / ROW_H) - OVERSCAN));
  const end = $derived(Math.min(rows.length, Math.ceil((scrollTop + viewportH) / ROW_H) + OVERSCAN));
  const visible = $derived(rows.slice(start, end));

  // Only the columns in view (and a screen's width either side) are drawn, so
  // a wide table costs no more to scroll than a narrow one.
  const colStarts = $derived.by(() => {
    const out = [0];
    for (const w of widths) out.push(out[out.length - 1] + w);
    return out;
  });
  const shownCols = $derived.by(() => {
    const from = scrollLeft - viewportW;
    const to = scrollLeft + 2 * viewportW;
    const out: number[] = [];
    for (let c = 0; c < columns.length; c++) {
      if (colStarts[c + 1] > from && colStarts[c] < to) out.push(c);
    }
    return out;
  });
  const leftPad = $derived(shownCols.length ? colStarts[shownCols[0]] : 0);
  const numeric = $derived(columns.map(c => isNumericType(c.type)));
  const keys = $derived(new Set(keyColumns));

  $effect(() => {
    if (!onselect) return;
    if (selection && selection.col >= 0 && rows[selection.row]) {
      onselect(rows[selection.row][selection.col], columns[selection.col]);
    } else {
      onselect(undefined, undefined);
    }
  });

  function select(row: number, col: number) {
    if (cellEditor && (cellEditor.r !== row || cellEditor.c !== col)) commitEdit();
    selection = { row, col };
    rowAnchor = null;
    scroller?.focus({ preventScroll: true });
  }

  function selectRow(r: number, extend: boolean) {
    commitEdit();
    if (extend && selection) rowAnchor ??= selection.row;
    else rowAnchor = null;
    selection = { row: r, col: -1 };
    scroller?.focus({ preventScroll: true });
  }

  /** Rows the row actions apply to: a shift-selected range, else the selected row. */
  function selectedRows(): number[] {
    if (!selection) return [];
    const a = rowAnchor ?? selection.row;
    const [lo, hi] = a < selection.row ? [a, selection.row] : [selection.row, a];
    return Array.from({ length: hi - lo + 1 }, (_, i) => lo + i);
  }

  /** Rows a row action at r covers: the shift-selected range if r is in it, else just r. */
  export function selectedRowsFor(r: number): number[] {
    const rs = selectedRows();
    return rs.includes(r) ? rs : [r];
  }

  function inRowRange(r: number): boolean {
    if (!selection || selection.col !== -1) return false;
    const a = rowAnchor ?? selection.row;
    return r >= Math.min(a, selection.row) && r <= Math.max(a, selection.row);
  }

  function startEdit(r: number, c: number, replaceWith?: string) {
    if (!editing?.canEdit(r, c)) return;
    const v = rows[r]?.[c];
    const isDefault = editing.cellState(r, c) === 'default';
    cellEditor = {
      r,
      c,
      text: replaceWith ?? (v === null || isDefault ? '' : String(v)),
      wasNull: v === null || isDefault,
      touched: replaceWith !== undefined,
      options: editing.options?.(r, c) ?? null,
    };
    selection = { row: r, col: c };
  }

  function commitEdit() {
    const e = cellEditor;
    if (!e) return;
    cellEditor = null;
    // Opening and leaving a NULL cell without typing must not turn it into ''.
    if (e.touched) editing?.commit(e.r, e.c, e.text);
    scroller?.focus({ preventScroll: true });
  }

  // Leaving the cell commits; the whole window losing focus (⌘Tab to another
  // app) does not, so the edit is still open on return.
  function onEditorBlur() {
    const blurred = cellEditor;
    setTimeout(() => {
      if (document.hasFocus() && cellEditor === blurred) commitEdit();
    });
  }

  function cancelEdit() {
    cellEditor = null;
    scroller?.focus({ preventScroll: true });
  }

  function onEditorKey(e: KeyboardEvent) {
    e.stopPropagation();
    if (e.key === 'Escape') {
      e.preventDefault();
      cancelEdit();
    } else if (e.key === 'Enter' && !e.shiftKey && !e.altKey) {
      e.preventDefault();
      commitEdit();
    } else if (e.key === 'Tab') {
      e.preventDefault();
      commitAndMove(e.shiftKey ? -1 : 1);
    }
  }

  /** Tab out of an editor: commit it and select the next (or previous) cell in the row. */
  function commitAndMove(delta: number) {
    const ed = cellEditor;
    if (!ed) return;
    commitEdit();
    const col = Math.max(0, Math.min(columns.length - 1, ed.c + delta));
    selection = { row: ed.r, col };
    scrollIntoView(ed.r, col);
  }

  function focusEditor(node: HTMLTextAreaElement) {
    node.focus();
    node.setSelectionRange(node.value.length, node.value.length);
  }

  /** Enum cells: picking a value commits it straight away. */
  function onOptionPick(value: string) {
    if (!cellEditor) return;
    cellEditor.text = value;
    cellEditor.touched = true;
    commitEdit();
  }

  /** The enum list closed: picking the value it already had still ends the edit, Tab moves on like in a text editor. */
  function onOptionClose(picked: boolean, key?: KeyboardEvent) {
    if (key?.key === 'Tab') {
      key.preventDefault();
      commitAndMove(key.shiftKey ? -1 : 1);
    } else if (picked) {
      commitEdit();
    } else {
      cancelEdit();
    }
  }

  /** Sends a parent action for the current selection. */
  function sendMenu(id: string): boolean {
    if (!selection || !onmenu) return false;
    const at = { r: selection.row, c: selection.col, rows: selectedRows() };
    const handled = onmenu(id, at) === true;
    scroller?.focus({ preventScroll: true });
    return handled;
  }

  function pick(id: string, m: { r: number; c: number }) {
    menu = null;
    const { r, c } = m;
    switch (id) {
      case 'edit':
        startEdit(r, c);
        return;
      case 'copy-value':
        copyToClipboard(copyText(rows[r][c]));
        return;
      case 'copy-rows':
        copyToClipboard(toTSV(columns.map(col => col.name), selectedRows().map(i => rows[i])));
        return;
      case 'fit':
        autoFit(c);
        return;
      case 'copy-name':
        copyToClipboard(columns[c].name);
        return;
    }
    onmenu?.(id, { r, c, rows: r === -1 ? [] : selectedRows() });
    scroller?.focus({ preventScroll: true });
  }

  /** Built-in items plus the parent's, with stray dividers removed. */
  function menuItems(m: { r: number; c: number }): MenuItem[] {
    let items: MenuItem[];
    if (m.r === -1) {
      items = [...(headerMenu?.(m.c) ?? []), 'sep', { id: 'fit', label: 'Fit width' }, { id: 'copy-name', label: 'Copy column name' }];
    } else {
      const n = selectedRows().length;
      const rowsLabel = n > 1 ? `Copy ${n} rows` : 'Copy row';
      items = m.c >= 0
        ? [
            { id: 'edit', label: 'Edit', kbd: '↵', disabled: !editing?.canEdit(m.r, m.c) },
            { id: 'copy-value', label: 'Copy value', kbd: '⌘C' },
            { id: 'copy-rows', label: rowsLabel, kbd: '⇧⌘C' },
          ]
        : [{ id: 'copy-rows', label: rowsLabel, kbd: '⇧⌘C' }];
      items.push('sep', ...(cellMenu?.(m.r, m.c) ?? []));
    }
    return items.filter((it, i, all) => it !== 'sep' || (i > 0 && i < all.length - 1 && all[i - 1] !== 'sep'));
  }

  function openHeaderMenu(e: MouseEvent, c: number) {
    e.preventDefault();
    commitEdit();
    menu = { x: e.clientX, y: e.clientY, r: -1, c };
  }

  function openMenu(e: MouseEvent, r: number, c: number) {
    e.preventDefault();
    commitEdit();
    const insideRange = c === -1 && inRowRange(r);
    if (!insideRange) {
      selection = { row: r, col: c };
      rowAnchor = null;
    }
    menu = { x: e.clientX, y: e.clientY, r, c };
  }

  /** Scrolls row r (and column c, if ≥ 0) into view, selects it, and optionally starts editing. */
  export function showRow(r: number, c: number, startEditing: boolean) {
    selection = { row: r, col: c };
    rowAnchor = null;
    // Rows may have just been added; wait for the DOM to grow first.
    tick().then(() => {
      scrollIntoView(r, c);
      if (startEditing && c >= 0) startEdit(r, c);
      else scroller?.focus({ preventScroll: true });
    });
  }

  // Clicks inside the open editor (placing the caret, selecting a word) must
  // not reach the cell, or the cell would take focus and close the editor.
  const inEditor = (e: Event) => (e.target as HTMLElement).closest('.cell-editor') !== null;

  const isMultiline = (text: string) => text.includes('\n') || text.length > 60;

  function scrollIntoView(row: number, col: number) {
    if (!scroller) return;
    const top = row * ROW_H;
    const bodyH = scroller.clientHeight - HEADER_H;
    if (top < scroller.scrollTop) scroller.scrollTop = top;
    else if (top + ROW_H > scroller.scrollTop + bodyH) scroller.scrollTop = top + ROW_H - bodyH;
    if (col < 0) return;
    const left = widths.slice(0, col).reduce((a, b) => a + b, 0);
    const viewW = scroller.clientWidth - rowNumberWidth;
    if (left < scroller.scrollLeft) scroller.scrollLeft = left;
    else if (left + widths[col] > scroller.scrollLeft + viewW) scroller.scrollLeft = left + widths[col] - viewW;
  }

  async function onkeydown(e: KeyboardEvent) {
    const mod = e.metaKey || e.ctrlKey;
    if (editing && selection) {
      if (mod && e.key === 'Backspace') {
        e.preventDefault();
        const rs = selectedRows();
        sendMenu(rs.every(r => editing!.rowState(r) === 'deleted') ? 'restore' : 'delete');
        return;
      }
      if (selection.col >= 0) {
        if (e.altKey && e.key === 'Backspace') {
          e.preventDefault();
          sendMenu('null');
          return;
        }
        if (e.key === ' ' && !mod && sendMenu('toggle')) {
          e.preventDefault(); // a boolean flipped; anything else starts editing below
          return;
        }
        if (e.key === 'Enter' && !mod) {
          e.preventDefault();
          startEdit(selection.row, selection.col);
          return;
        }
        if ((e.key === 'Backspace' || e.key === 'Delete') && !mod) {
          e.preventDefault();
          startEdit(selection.row, selection.col, '');
          return;
        }
        if (e.key.length === 1 && !mod && !e.altKey) {
          e.preventDefault();
          startEdit(selection.row, selection.col, e.key);
          return;
        }
      }
    }
    if (mod && e.key.toLowerCase() === 'c') {
      if (!selection) return;
      e.preventDefault();
      if (selection.col >= 0 && !e.shiftKey) {
        await copyToClipboard(copyText(rows[selection.row][selection.col]));
      } else {
        await copyToClipboard(toTSV(columns.map(c => c.name), selectedRows().map(r => rows[r])));
      }
      return;
    }
    if (mod && e.key.toLowerCase() === 'a') {
      e.preventDefault();
      await copyToClipboard(toTSV(columns.map(c => c.name), rows));
      return;
    }
    const moves: Record<string, [number, number]> = {
      ArrowUp: [-1, 0], ArrowDown: [1, 0], ArrowLeft: [0, -1], ArrowRight: [0, 1],
      Tab: [0, e.shiftKey ? -1 : 1],
    };
    const move = moves[e.key];
    if (!move || rows.length === 0) return;
    e.preventDefault();
    const cur = selection ?? { row: 0, col: 0 };
    const pageJump = mod ? (move[0] !== 0 ? rows.length : columns.length) : 1;
    const row = Math.max(0, Math.min(rows.length - 1, cur.row + move[0] * pageJump));
    const col = Math.max(0, Math.min(columns.length - 1, Math.max(cur.col, 0) + move[1] * pageJump));
    selection = { row, col };
    scrollIntoView(row, col);
  }

  function startResize(e: PointerEvent, i: number) {
    e.stopPropagation();
    const w0 = widths[i];
    startDrag(e, 'col-resize', dx => (widths[i] = Math.max(48, w0 + dx)));
  }

  function autoFit(i: number) {
    const c = columns[i];
    widths[i] = Math.min(calcColumnWidth(c.name, c.type, rows, i) * 2, 900);
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
  class="grid"
  bind:this={scroller}
  bind:clientHeight={viewportH}
  bind:clientWidth={viewportW}
  onscroll={() => {
    scrollTop = scroller?.scrollTop ?? 0;
    scrollLeft = scroller?.scrollLeft ?? 0;
  }}
  {onkeydown}
  tabindex="0"
  role="grid"
  aria-rowcount={rows.length}
  aria-colcount={columns.length}
>
  <div class="inner" style:width="{totalWidth}px" style:height="{HEADER_H + rows.length * ROW_H}px">
    <div class="header" role="row" style:height="{HEADER_H}px">
      <div class="rn corner" style:width="{rowNumberWidth}px"></div>
      <div class="pad" style:width="{leftPad}px"></div>
      {#each shownCols as i (i)}
        {@const col = columns[i]}
        {@const sorted = sort?.column === col.name}
        <div class="th" class:sortable={!!onsort} class:num={numeric[i]} role="columnheader" style:width="{widths[i]}px" title="{col.name} · {col.type || 'unknown'}">
          <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
          <div class="th-label" onclick={() => onsort?.(col.name)} oncontextmenu={e => openHeaderMenu(e, i)}>
            {#if keys.has(col.name)}<span class="pk"><Icon name="key" size={11} /></span>{/if}
            <span class="th-name">{col.name}</span>
            {#if sorted}<Icon name={sort?.desc ? 'arrowDown' : 'arrowUp'} size={11} />{/if}
          </div>
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div class="resizer" onpointerdown={e => startResize(e, i)} ondblclick={() => autoFit(i)}></div>
        </div>
      {/each}
    </div>

    <div class="body" style:transform="translateY({start * ROW_H}px)">
      <!-- Unkeyed: scrolling reuses the row elements and only swaps their contents. -->
      {#each visible as row, vi}
        {@const r = start + vi}
        {@const rowSelected = selection?.row === r}
        {@const rowState = editing?.rowState(r) ?? ''}
        <div
          class="tr {rowState}"
          class:alt={r % 2 === 1}
          class:failed={editing?.isFailed(r) ?? false}
          class:row-selected={inRowRange(r)}
          role="row"
          style:height="{ROW_H}px"
        >
          <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
          <div
            class="rn"
            class:active={rowSelected || inRowRange(r)}
            style:width="{rowNumberWidth}px"
            onclick={e => selectRow(r, e.shiftKey)}
            oncontextmenu={e => openMenu(e, r, -1)}
          >
            {#if rowState === 'new'}<span class="new-mark">+</span>{:else if rowOffset === null}·{:else}{rowOffset + r + 1}{/if}
          </div>
          <div class="pad" style:width="{leftPad}px"></div>
          {#each shownCols as c (c)}
            {@const value = row[c]}
            {@const state = editing?.cellState(r, c) ?? ''}
            {@const kind = state === 'default' ? 'null' : cellKind(value, columns[c].type)}
            {@const isEditing = cellEditor?.r === r && cellEditor?.c === c}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <div
              class="td {kind} {state}"
              class:num={numeric[c] || kind === 'number'}
              class:selected={rowSelected && selection?.col === c}
              class:editing={isEditing}
              role="gridcell"
              tabindex="-1"
              style:width="{widths[c]}px"
              onmousedown={e => e.button === 0 && !inEditor(e) && select(r, c)}
              ondblclick={e => !inEditor(e) && startEdit(r, c)}
              oncontextmenu={e => openMenu(e, r, c)}
            >
              {#if isEditing && cellEditor && cellEditor.options}
                <div class="cell-editor list">
                  <Select
                    value={cellEditor.text}
                    options={cellEditor.options.map(o => ({ value: o, label: o }))}
                    placeholder={cellEditor.wasNull ? (state === 'default' ? 'DEFAULT' : 'NULL') : ''}
                    startOpen
                    onchange={onOptionPick}
                    onclose={onOptionClose}
                    aria-label={columns[c].name}
                  />
                </div>
              {:else if isEditing && cellEditor}
                <textarea
                  class="cell-editor"
                  class:multi={isMultiline(cellEditor.text)}
                  style:min-width="{widths[c]}px"
                  bind:value={cellEditor.text}
                  oninput={() => cellEditor && (cellEditor.touched = true)}
                  onkeydown={onEditorKey}
                  onblur={onEditorBlur}
                  placeholder={cellEditor.wasNull ? (state === 'default' ? 'DEFAULT' : 'NULL') : ''}
                  spellcheck="false"
                  use:focusEditor
                ></textarea>
              {:else}
                {state === 'default' ? 'DEFAULT' : formatCell(value)}
              {/if}
            </div>
          {/each}
        </div>
      {/each}
    </div>
  </div>

  {#if menu}
    {@const m = menu}
    {@const items = menuItems(m)}
    {@const top = Math.max(8, Math.min(m.y, window.innerHeight - items.length * 27 - 16))}
    {@const left = Math.max(8, Math.min(m.x, window.innerWidth - 236))}
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="menu-backdrop" onclick={() => (menu = null)} oncontextmenu={e => { e.preventDefault(); menu = null; }}></div>
    <div class="menu" role="menu" style:left="{left}px" style:top="{top}px">
      {#each items as it, i (i)}
        {#if it === 'sep'}
          <div class="sep"></div>
        {:else}
          <button role="menuitem" class:danger={it.danger} disabled={it.disabled} onclick={() => pick(it.id, m)}>
            {it.label}{#if it.kbd}<span class="kbd">{it.kbd}</span>{/if}
          </button>
        {/if}
      {/each}
    </div>
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

  .header {
    position: sticky;
    top: 0;
    z-index: 2;
    display: flex;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
    font-family: var(--font-ui);
    font-size: 12px;
  }
  .th {
    position: relative;
    flex: none;
    display: flex;
    align-items: center;
    border-right: 1px solid var(--border-subtle);
    color: var(--text);
    font-weight: 550;
  }
  .th-label {
    flex: 1;
    min-width: 0;
    height: 100%;
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 0 9px;
    overflow: hidden;
  }
  .th.num .th-label { justify-content: flex-end; }
  .th.sortable .th-label:hover { background: var(--hover); }
  .th-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .pk { color: var(--text-3); display: flex; }
  .resizer {
    position: absolute;
    top: 0;
    right: -4px;
    bottom: 0;
    width: 8px;
    z-index: 1;
    cursor: col-resize;
  }
  .resizer:hover { background: linear-gradient(to right, transparent 3px, var(--accent) 3px, var(--accent) 5px, transparent 5px); }

  .body { position: absolute; top: 30px; left: 0; right: 0; will-change: transform; }
  .tr { display: flex; }
  .pad { flex: none; }
  .tr.alt { background: var(--grid-row-alt); }
  .tr.row-selected .td { background: var(--grid-selected); }

  .rn, .corner {
    position: sticky;
    left: 0;
    z-index: 1;
    flex: none;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    padding-right: 8px;
    background: var(--surface);
    border-right: 1px solid var(--border);
    color: var(--text-3);
    font-size: 11px;
  }
  .corner { z-index: 3; }
  .rn.active { color: var(--text); background: var(--elevated); }

  .td {
    flex: none;
    padding: 0 9px;
    line-height: 26px;
    border-right: 1px solid var(--border-subtle);
    border-bottom: 1px solid var(--border-subtle);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text);
  }
  .td.num { text-align: right; }
  .td.number { color: var(--cell-number); }
  .td.bool { color: var(--cell-bool); }
  .td.null { color: var(--cell-null); font-style: italic; }
  .td.selected {
    background: var(--grid-selected);
    box-shadow: inset 0 0 0 1.5px var(--accent);
  }

  .td.edited { background: color-mix(in srgb, var(--warn) 16%, transparent); }
  .td.default { color: var(--text-3); font-style: italic; }
  .td.expr { background: color-mix(in srgb, var(--warn) 16%, transparent); color: var(--text-2); font-style: italic; }
  .cell-editor.list { padding: 0; box-shadow: none; background: none; overflow: visible; white-space: normal; }
  .cell-editor.list :global(.select-button) { height: 28px; border-radius: 3px; }
  .tr.new .td { background: color-mix(in srgb, var(--ok) 9%, transparent); }
  .tr.new .td.edited { background: color-mix(in srgb, var(--ok) 18%, transparent); }
  .tr.deleted .td { background: color-mix(in srgb, var(--danger) 12%, transparent); color: var(--text-3); text-decoration: line-through; }
  .tr.failed .td { background: color-mix(in srgb, var(--danger) 22%, transparent); }
  .tr.deleted .rn, .tr.failed .rn { color: var(--danger); }
  .new-mark { color: var(--ok); font-weight: 700; font-size: 13px; }

  .td.editing { position: relative; overflow: visible; z-index: 4; padding: 0; }
  .cell-editor {
    position: absolute;
    top: -1px;
    left: -1px;
    width: calc(100% + 2px);
    height: 28px;
    margin: 0;
    padding: 4px 8px;
    border: 0;
    border-radius: 3px;
    outline: none;
    box-shadow: 0 0 0 2px var(--accent), 0 6px 20px rgba(0, 0, 0, 0.25);
    background: var(--bg);
    color: var(--text);
    font: inherit;
    line-height: 20px;
    resize: none;
    overflow: hidden;
    white-space: pre;
  }
  .cell-editor.multi {
    width: max(calc(100% + 2px), 340px);
    height: 132px;
    white-space: pre-wrap;
    overflow: auto;
  }
  .cell-editor::placeholder { color: var(--text-3); font-style: italic; }

  .menu-backdrop { position: fixed; inset: 0; z-index: 40; }
  .menu {
    position: fixed;
    z-index: 41;
    min-width: 190px;
    padding: 4px;
    border-radius: 8px;
    background: var(--elevated);
    box-shadow: var(--shadow-modal);
    font-family: var(--font-ui);
    font-size: 12.5px;
  }
  .menu button {
    display: flex;
    align-items: center;
    width: 100%;
    height: 26px;
    padding: 0 8px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text);
    text-align: left;
  }
  .menu button .kbd { margin-left: auto; }
  .menu button:hover:not(:disabled) { background: var(--accent); color: var(--on-accent); }
  .menu button:hover:not(:disabled) .kbd { color: inherit; border-color: rgba(255, 255, 255, 0.4); }
  .menu button:disabled { color: var(--text-3); }
  .menu button.danger { color: var(--danger); }
  .menu button.danger:hover { color: var(--on-accent); background: var(--danger); }
  .menu .sep { height: 1px; margin: 4px 6px; background: var(--border-subtle); }

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
