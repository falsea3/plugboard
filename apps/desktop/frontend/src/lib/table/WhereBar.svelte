<script lang="ts">
  import Icon from '../ui/Icon.svelte';

  let { value = $bindable(), applied, placeholder = '', onapply }: { value: string; applied: string; placeholder?: string; onapply: () => void } = $props();

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
    <span class="kw">where</span>
    <input
      bind:this={input}
      bind:value
      {onkeydown}
      class="mono"
      {placeholder}
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
  .where { flex: none; margin-top: 8px; }
  .field { display: flex; align-items: center; gap: 10px; height: 38px; padding: 0 4px 0 12px; border: 1px solid var(--hover); border-radius: var(--radius); background: var(--surface); }
  .field:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
  .kw { flex: none; font-family: var(--font-mono); font-size: 12.5px; color: var(--text-4); }
  .field.active .kw { color: var(--accent-text); }
  input { flex: 1; min-width: 0; height: 100%; padding: 0; border: 0; outline: none; background: transparent; color: var(--text); font-family: var(--font-mono); font-size: 12.5px; }
  input::placeholder { color: var(--text-4); }
  .hint { flex: none; font-size: 11px; }
</style>
