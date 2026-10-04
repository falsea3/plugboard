<script lang="ts">
  import type { CellValue, ResultColumn } from '../api/wire';
  import { cellKind, formatCell } from '../ui/format';
  import { inCellEditor, type GridEditing } from './grid';
  import type { GridEditor } from './editor.svelte';
  import CellEditor from './CellEditor.svelte';
  import Icon from '../ui/Icon.svelte';

  let {
    r,
    row,
    columns,
    shown,
    widths,
    leftPad,
    numeric,
    numberWidth,
    rowOffset,
    height,
    editing,
    cursorCol,
    marked,
    links,
    editor,
    onpoint,
    onmenu,
    onfollow,
  }: {
    r: number;
    row: CellValue[];
    columns: ResultColumn[];
    shown: number[];
    widths: number[];
    leftPad: number;
    numeric: boolean[];
    numberWidth: number;
    rowOffset: number | null;
    height: number;
    editing: GridEditing | null;
    cursorCol: number | null;
    marked: boolean;
    links?: Set<number>;
    editor: GridEditor;
    onpoint: (c: number, e: MouseEvent) => void;
    onmenu: (e: MouseEvent, c: number) => void;
    onfollow: (c: number) => void;
  } = $props();

  const rowState = $derived(editing?.rowState(r) ?? '');
</script>

<div class="tr {rowState}" class:alt={r % 2 === 1} class:failed={editing?.isFailed(r) ?? false} class:row-selected={marked} role="row" style:height="{height}px">
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="rn" class:active={cursorCol !== null || marked} style:width="{numberWidth}px" onclick={e => onpoint(-1, e)} oncontextmenu={e => onmenu(e, -1)}>
    {#if rowState === 'new'}<span class="new-mark">+</span>{:else if rowOffset === null}·{:else}{rowOffset + r + 1}{/if}
  </div>
  <div class="pad" style:width="{leftPad}px"></div>
  {#each shown as c (c)}
    {@const value = row[c]}
    {@const state = editing?.cellState(r, c) ?? ''}
    {@const kind = state === 'default' ? 'null' : cellKind(value, columns[c].kind)}
    {@const open = editor.open?.r === r && editor.open?.c === c ? editor.open : null}
    {@const link = !!links?.has(c) && value !== null && rowState !== 'new'}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      class="td {kind} {state}"
      class:link
      class:num={numeric[c] || kind === 'number'}
      class:selected={cursorCol === c}
      class:editing={!!open}
      role="gridcell"
      tabindex="-1"
      style:width="{widths[c]}px"
      onmousedown={e => e.button === 0 && !inCellEditor(e) && onpoint(c, e)}
      ondblclick={e => !inCellEditor(e) && editor.start(r, c)}
      oncontextmenu={e => onmenu(e, c)}
    >
      {#if open}
        <CellEditor
          bind:text={() => open.text, t => editor.type(t)}
          options={open.options}
          placeholder={open.wasNull ? (state === 'default' ? 'DEFAULT' : 'NULL') : ''}
          width={widths[c]}
          label={columns[c].name}
          oninput={() => editor.touch()}
          onkey={e => editor.key(e)}
          onblur={() => editor.blur()}
          onpick={v => editor.pick(v)}
          onclose={(chosen, key) => editor.closeList(chosen, key)}
        />
      {:else}
        {state === 'default' ? 'DEFAULT' : formatCell(value)}
      {/if}
      {#if link && !open}
        <button
          class="follow"
          tabindex="-1"
          title="Go to the referenced row"
          aria-label="Go to the referenced row"
          onmousedown={e => e.stopPropagation()}
          ondblclick={e => e.stopPropagation()}
          onclick={e => {
            e.stopPropagation();
            onfollow(c);
          }}
        ><Icon name="follow" size={11} /></button>
      {/if}
    </div>
  {/each}
</div>

<style>
  .tr { display: flex; }
  .pad { flex: none; }
  .tr.alt { background: var(--grid-row-alt); }
  .tr.row-selected .td { background: var(--grid-selected); }

  .rn {
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
  .td.link { position: relative; padding-right: 24px; }
  .follow {
    position: absolute;
    top: 50%;
    right: 4px;
    transform: translateY(-50%);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: var(--elevated);
    color: var(--text-2);
    opacity: 0;
    cursor: pointer;
  }
  .tr:hover .follow, .td.selected .follow { opacity: 1; }
  .follow:hover { background: var(--accent); color: var(--on-accent); }
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
  .tr.new .td { background: color-mix(in srgb, var(--ok) 9%, transparent); }
  .tr.new .td.edited { background: color-mix(in srgb, var(--ok) 18%, transparent); }
  .tr.deleted .td { background: color-mix(in srgb, var(--danger) 12%, transparent); color: var(--text-3); text-decoration: line-through; }
  .tr.failed .td { background: color-mix(in srgb, var(--danger) 22%, transparent); }
  .tr.deleted .rn, .tr.failed .rn { color: var(--danger); }
  .new-mark { color: var(--ok); font-weight: 700; font-size: 13px; }
  .td.editing { position: relative; overflow: visible; z-index: 4; padding: 0; }
</style>
