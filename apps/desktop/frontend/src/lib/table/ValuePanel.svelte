<script lang="ts">
  import type { CellValue, ResultColumn } from '../api/backend';
  import { copyText } from '../ui/format';
  import { copyToClipboard } from '../ui/clipboard';
  import Icon from '../ui/Icon.svelte';

  let { selected }: { selected: { value: CellValue; column: ResultColumn } | null } = $props();

  const text = $derived(selected ? (selected.value === null ? 'NULL' : String(selected.value)) : '');

  function copy() {
    if (selected) void copyToClipboard(copyText(selected.value));
  }
</script>

<section class="value-panel" aria-label="Selected cell value">
  <header>
    <strong>Value</strong>
    {#if selected}
      <button class="btn icon sm ghost" onclick={copy} title="Copy value" aria-label="Copy value"><Icon name="copy" size={13} /></button>
    {/if}
  </header>
  {#if selected}
    <div class="field-info">
      <span class="column" title={selected.column.name}>{selected.column.name}</span>
      <span class="type" title={selected.column.type}>{selected.column.type || 'Unknown type'}</span>
    </div>
    <pre class:null={selected.value === null}>{text}</pre>
  {:else}
    <div class="empty">Select a cell to view its value</div>
  {/if}
</section>

<style>
  .value-panel {
    display: flex;
    flex-direction: column;
    width: 100%;
    min-width: 0;
    min-height: 0;
    height: 100%;
    background: var(--bg);
  }
  header {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    height: 34px;
    padding: 0 8px 0 12px;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
  }
  strong { font-size: 12px; font-weight: 600; }
  .field-info {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    height: 32px;
    padding: 0 12px;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface);
  }
  .column { min-width: 0; overflow: hidden; color: var(--text); font: 11.5px var(--font-mono); text-overflow: ellipsis; white-space: nowrap; }
  .type { flex: none; max-width: 45%; overflow: hidden; padding: 2px 6px; border: 1px solid var(--border-subtle); border-radius: 4px; color: var(--text-3); font-size: 10.5px; text-overflow: ellipsis; white-space: nowrap; }
  pre {
    flex: 1;
    min-height: 0;
    margin: 8px;
    padding: 10px;
    overflow: auto;
    color: var(--text);
    border: 1px solid var(--border-subtle);
    border-radius: 6px;
    background: var(--surface);
    font: 12px/1.55 var(--font-mono);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    user-select: text;
    -webkit-user-select: text;
  }
  pre.null { color: var(--cell-null); font-style: italic; }
  .empty { display: grid; flex: 1; place-items: center; padding: 20px; color: var(--text-3); font-size: 12px; text-align: center; }
</style>
