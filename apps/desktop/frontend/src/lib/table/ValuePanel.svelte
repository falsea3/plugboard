<script lang="ts">
  import { untrack } from 'svelte';
  import type { TableTab } from '../app/workspace.svelte';
  import { copyText } from '../ui/format';
  import { copyToClipboard } from '../ui/clipboard';
  import { jsonError, jsonToSave, looksLikeJSON } from '../json/text';
  import Icon from '../ui/Icon.svelte';

  let { selected }: { selected: TableTab['selected'] | null } = $props();

  const original = $derived(selected && selected.value !== null ? String(selected.value) : '');
  const json = $derived(!!selected && (selected.column.kind === 'json' || looksLikeJSON(selected.value)));
  let draft = $state('');
  let problem = $state('');

  $effect(() => {
    const text = original;
    void selected?.key;
    untrack(() => {
      draft = text;
      problem = '';
    });
  });

  const dirty = $derived(!!selected && draft !== original);

  function copy() {
    if (selected) void copyToClipboard(copyText(selected.value));
  }

  function apply() {
    if (!selected?.edit || !dirty) return;
    if (json && draft.trim() !== '') {
      problem = jsonError(draft);
      if (problem) return;
      const value = jsonToSave(original, draft);
      if (value !== null) selected.edit(value);
      else draft = original;
      return;
    }
    selected.edit(draft);
  }

  function onkeydown(e: KeyboardEvent) {
    const mod = e.metaKey || e.ctrlKey;
    if (mod && e.key === 'Enter') {
      e.preventDefault();
      apply();
    } else if (mod && e.key.toLowerCase() === 's' && dirty) {
      apply();
    } else if (e.key === 'Escape' && dirty) {
      e.stopPropagation();
      draft = original;
      problem = '';
    }
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
    {#if selected.edit}
      <textarea
        class="editor"
        class:null={selected.value === null && !dirty}
        bind:value={draft}
        oninput={() => (problem = '')}
        {onkeydown}
        placeholder={selected.value === null ? 'NULL' : ''}
        spellcheck="false"
        aria-label="Value of {selected.column.name}"
      ></textarea>
      {#if problem}<p class="problem" role="alert">{problem}</p>{/if}
      <div class="actions">
        {#if selected.nullable && selected.value !== null}
          <button class="btn sm ghost" onclick={() => selected?.edit?.(null)}>Set NULL</button>
        {/if}
        <span style="flex:1"></span>
        <button class="btn sm ghost" onclick={() => ((draft = original), (problem = ''))} disabled={!dirty}>Revert</button>
        <button class="btn sm primary" onclick={apply} disabled={!dirty} title="Apply to the cell (⌘↵) — commit with ⌘S">Apply</button>
      </div>
    {:else}
      <pre class:null={selected.value === null}>{selected.value === null ? 'NULL' : original}</pre>
    {/if}
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
  pre, .editor {
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
  .editor { resize: none; outline: none; }
  .editor:focus { border-color: var(--accent); }
  pre.null, .editor.null::placeholder { color: var(--cell-null); font-style: italic; }
  .problem { flex: none; margin: 0 8px 4px; color: var(--danger); font-size: 11.5px; }
  .actions { flex: none; display: flex; align-items: center; gap: 6px; padding: 0 8px 8px; }
  .empty { display: grid; flex: 1; place-items: center; padding: 20px; color: var(--text-3); font-size: 12px; text-align: center; }
</style>
