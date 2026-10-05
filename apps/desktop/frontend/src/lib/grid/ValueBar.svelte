<script lang="ts">
  import type { CellValue, ResultColumn } from '../api/wire';
  import { copyText } from '../ui/format';
  import { copyToClipboard } from '../ui/clipboard';
  import Icon from '../ui/Icon.svelte';

  let { value, column, onopen }: { value: CellValue; column: ResultColumn; onopen?: () => void } = $props();

  const text = $derived(value === null ? 'NULL' : String(value));
</script>

<div class="value-bar">
  <span class="col faint">{column.name}{column.type ? ` · ${column.type}` : ''}</span>
  <span class="val mono" class:null={value === null} title={text}>{text}</span>
  {#if onopen && column.kind !== 'binary'}<button class="btn icon sm ghost" title="Open in editor (⇧↵)" aria-label="Open in editor" onclick={onopen}><Icon name="fit" size={12} /></button>{/if}
  <button class="btn icon sm ghost" title="Copy value (⌘C)" onclick={() => copyToClipboard(copyText(value))}><Icon name="copy" size={12} /></button>
</div>

<style>
  .value-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    max-width: 55%;
  }
  .col { font-size: 11.5px; white-space: nowrap; }
  .val {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 11.5px;
    color: var(--text-2);
    user-select: text;
    -webkit-user-select: text;
  }
  .val.null { font-style: italic; color: var(--cell-null); }
</style>
