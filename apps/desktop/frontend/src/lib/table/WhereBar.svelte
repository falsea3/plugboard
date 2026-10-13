<script lang="ts">
  import Icon from '../ui/Icon.svelte';

  let { value = $bindable(), applied, onapply }: { value: string; applied: string; onapply: () => void } = $props();

  let input = $state<HTMLInputElement>();
  const pending = $derived(value.trim() !== applied);

  export function focus() {
    input?.focus();
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      onapply();
    } else if (e.key === 'Escape' && pending) {
      e.preventDefault();
      e.stopPropagation();
      value = applied;
    }
  }

  function clear() {
    value = '';
    if (applied) onapply();
    input?.focus();
  }
</script>

<div class="where">
  <div class="field" class:active={!!applied}>
    <Icon name="funnel" size={12} />
    <input
      bind:this={input}
      bind:value
      {onkeydown}
      class="mono"
      placeholder="Enter a SQL expression to filter rows, like plan = 'pro' AND total > 100"
      spellcheck="false"
      autocomplete="off"
      aria-label="WHERE condition"
    />
    {#if pending}<span class="hint faint"><span class="kbd">↵</span> apply · <span class="kbd">esc</span> undo</span>{/if}
    {#if value || applied}
      <button class="btn icon sm ghost" onclick={clear} title="Clear the condition" aria-label="Clear the condition"><Icon name="x" size={12} /></button>
    {/if}
  </div>
</div>

<style>
  .where { flex: none; padding: 6px 8px; border-bottom: 1px solid var(--border); background: var(--surface); }
  .field { display: flex; align-items: center; gap: 8px; height: 28px; padding: 0 2px 0 9px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--bg); color: var(--text-3); }
  .field:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
  .field.active { color: var(--accent); }
  input { flex: 1; min-width: 0; height: 100%; padding: 0; border: 0; outline: none; background: transparent; color: var(--text); font-size: 12px; }
  input::placeholder { color: var(--text-3); }
  .hint { flex: none; font-size: 11px; }
</style>
