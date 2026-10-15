<script lang="ts">
  import { untrack } from 'svelte';
  import type { TableTab } from '../app/workspace.svelte';
  import { jsonError, jsonToSave, looksLikeJSON } from '../json/text';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import { app } from '../app/app.svelte';

  let { selected, width = 280, onclose }: { selected: TableTab['selected'] | null; width?: number; onclose: () => void } = $props();

  let full = $state<string | null>(null);
  let loading = $state(false);
  const partial = $derived(!!selected?.cut && full === null);
  const original = $derived(full ?? (selected && selected.value !== null ? String(selected.value) : ''));
  const editable = $derived(!!selected?.edit && !partial);
  const json = $derived(!!selected && (selected.column.kind === 'json' || looksLikeJSON(selected.value)));
  let draft = $state('');
  let problem = $state('');

  $effect(() => {
    void selected?.key;
    void selected?.value;
    untrack(() => (full = null));
  });

  $effect(() => {
    const text = original;
    void selected?.key;
    untrack(() => {
      draft = text;
      problem = '';
    });
  });

  const dirty = $derived(!!selected && draft !== original);

  async function loadWhole() {
    if (!selected?.cut || loading) return;
    loading = true;
    try {
      full = await selected.cut.load();
    } catch (err) {
      app.notify(err);
    } finally {
      loading = false;
    }
  }

  function apply() {
    if (!selected?.edit || !editable || !dirty) return;
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

<section class="value-panel" aria-label="Selected cell value" style:width="{width}px">
  <header>
    <span class="title">Value</span>
    <button class="close" onclick={onclose} title="Hide the Value panel" aria-label="Hide the Value panel"><Icon name="x" size={13} /></button>
  </header>
  {#if selected}
    <div class="field-info">
      <span class="column" title={selected.column.name}>{selected.column.name}</span>
      <span class="chip" title={selected.column.type}>{selected.column.type || 'unknown'}</span>
      {#if selected.nullable}<span class="chip">nullable</span>{/if}
    </div>
    {#if partial}
      <div class="partial">
        <span>{selected.cut?.note}</span>
        <button class="btn sm" onclick={loadWhole} disabled={loading}>{#if loading}<Spinner size={11} />{/if}Load whole value</button>
      </div>
    {/if}
    {#if editable}
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
      {#if problem}<p class="problem" role="alert">{problem}</p>{:else if dirty}<p class="was">was <span>{selected.value === null ? 'NULL' : original}</span></p>{/if}
      <div class="actions">
        {#if selected.nullable && selected.value !== null}
          <button class="btn sm outline" onclick={() => selected?.edit?.(null)}>Set NULL</button>
        {/if}
        <span style="flex:1"></span>
        <button class="btn sm apply" onclick={apply} disabled={!dirty} title="Apply to the cell — commit with ⌘S">Apply ⌘↵</button>
      </div>
    {:else}
      <pre class:null={selected.value === null}>{selected.value === null ? 'NULL' : original}</pre>
    {/if}
  {:else}
    <div class="empty">Select a cell to view its value</div>
  {/if}
</section>

<style>
  .value-panel { flex: none; display: flex; flex-direction: column; min-width: 0; min-height: 0; padding: 14px; border: 1px solid var(--hover); border-radius: var(--radius-lg); background: var(--elevated); overflow: auto; }
  header { flex: none; display: flex; align-items: center; justify-content: space-between; }
  .title { font-weight: 500; }
  .close { display: grid; place-items: center; width: 24px; height: 24px; padding: 0; border: 0; border-radius: 7px; background: transparent; color: var(--text-4); }
  .close:hover { background: var(--hover); color: var(--text); }
  .field-info { flex: none; display: flex; align-items: center; gap: 6px; min-width: 0; margin-top: 10px; }
  .column { min-width: 0; overflow: hidden; font-family: var(--font-mono); text-overflow: ellipsis; white-space: nowrap; }
  .chip { flex: none; max-width: 45%; overflow: hidden; padding: 1px 6px; border-radius: 5px; background: var(--hover); color: var(--text-3); font-size: 10.5px; text-transform: uppercase; text-overflow: ellipsis; white-space: nowrap; }
  .chip + .chip { text-transform: none; }
  pre, .editor { flex: none; height: 120px; margin: 12px 0 0; padding: 10px 12px; overflow: auto; border: 1px solid var(--border); border-radius: var(--radius); background: var(--card); color: var(--text); font: 12.5px/1.5 var(--font-mono); white-space: pre-wrap; overflow-wrap: anywhere; user-select: text; -webkit-user-select: text; }
  .editor { resize: vertical; outline: none; }
  .editor:focus { border-color: var(--accent); }
  pre.null, .editor.null::placeholder { color: var(--cell-null); font-style: italic; }
  .problem { flex: none; margin: 6px 0 0; color: var(--danger); font-size: 12px; }
  .was { flex: none; margin: 6px 0 0; overflow: hidden; color: var(--text-4); font-size: 12px; white-space: nowrap; text-overflow: ellipsis; }
  .was span { color: var(--text-2); }
  .actions { flex: none; display: flex; align-items: center; gap: 6px; margin-top: 12px; }
  .apply { background: var(--active); }
  .partial { flex: none; display: flex; align-items: center; gap: 8px; margin-top: 12px; padding: 6px 8px; border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); color: var(--text-2); font-size: 11.5px; }
  .partial span { flex: 1; min-width: 0; }
  .empty { display: grid; flex: 1; place-items: center; padding: 20px; color: var(--text-3); font-size: 12px; text-align: center; }
</style>
