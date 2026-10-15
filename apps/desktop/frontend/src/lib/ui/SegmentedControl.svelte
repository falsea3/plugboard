<script lang="ts" generics="T extends string">
  import Icon, { type IconName } from './Icon.svelte';

  let {
    options,
    value = $bindable(),
    label,
    size = 'md',
    onchange,
  }: {
    options: { value: T; label: string; icon?: IconName }[];
    value: T;
    label: string;
    size?: 'md' | 'sm';
    onchange?: (value: T) => void;
  } = $props();
</script>

<div class="seg" class:sm={size === 'sm'} role="tablist" aria-label={label}>
  {#each options as o (o.value)}
    <button
      role="tab"
      aria-selected={value === o.value}
      class:on={value === o.value}
      onclick={() => {
        value = o.value;
        onchange?.(o.value);
      }}
    >
      {#if o.icon}<Icon name={o.icon} size={13} />{/if}
      {o.label}
    </button>
  {/each}
</div>

<style>
  .seg { display: inline-flex; flex: none; padding: 3px; gap: 2px; background: var(--elevated); border-radius: var(--radius); }
  .seg button {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 28px;
    padding: 0 14px;
    border: 0;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-3);
    font-size: 13px;
    white-space: nowrap;
    transition: background 0.12s, color 0.12s;
  }
  .seg.sm button { height: 24px; padding: 0 10px; font-size: 12.5px; }
  .seg button:hover { color: var(--text); }
  .seg button.on { background: var(--active); color: var(--text); font-weight: 500; }
</style>
