<script lang="ts">
  import type { Snippet } from 'svelte';
  import Switch from './Switch.svelte';

  let {
    checked = $bindable(false),
    title,
    hint,
    onchange,
    children,
  }: { checked?: boolean; title: string; hint: string; onchange?: (on: boolean) => void; children?: Snippet } = $props();
</script>

<label class="toggle-card">
  <span class="text">
    <span class="title">{title}</span>
    <span class="hint">{hint}</span>
    {#if children}<span class="more">{@render children()}</span>{/if}
  </span>
  <Switch bind:checked {onchange} label={title} />
</label>

<style>
  .toggle-card { display: flex; align-items: center; gap: 10px; padding: 12px 14px; border: 1px solid var(--border-subtle); border-radius: 12px; background: var(--surface); cursor: default; }
  .text { flex: 1; min-width: 0; }
  .title { display: block; font-weight: 500; }
  .hint { display: block; margin-top: 1px; color: var(--text-3); font-size: 12px; }
  .more { display: block; margin-top: 6px; color: var(--text-3); font-size: 11.5px; line-height: 1.45; }
</style>
