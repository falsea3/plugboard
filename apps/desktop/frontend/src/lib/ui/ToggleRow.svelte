<script lang="ts">
  import type { Snippet } from 'svelte';
  import Switch from './Switch.svelte';

  let {
    checked = $bindable(false),
    onchange,
    title,
    children,
  }: { checked?: boolean; onchange?: (on: boolean) => void; title: Snippet; children: Snippet } = $props();
</script>

<label class="toggle-row">
  <Switch small bind:checked {onchange} />
  <span>
    <span class="toggle-title">{@render title()}</span>
    <span class="toggle-hint">{@render children()}</span>
  </span>
</label>

<style>
  .toggle-row { display: flex; align-items: flex-start; gap: 10px; padding: 4px 0; text-align: left !important; }
  .toggle-title { display: flex; align-items: center; gap: 6px; color: var(--text); font-weight: 550; font-size: 12.5px; }
  .toggle-hint { display: block; margin: 2px 0 0; color: var(--text-3); font-size: 11.5px; line-height: 1.45; }
  .toggle-hint :global(strong) { color: var(--text-2); font-weight: 550; }
  .toggle-hint :global(code) { font-family: var(--font-mono); font-size: 11px; }
</style>
