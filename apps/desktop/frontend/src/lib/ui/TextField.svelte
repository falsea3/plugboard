<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import Icon, { type IconName } from './Icon.svelte';

  let {
    value = $bindable(''),
    element = $bindable(),
    icon,
    kbd,
    label,
    mono = false,
    size = 'md',
    prefix,
    ...rest
  }: Omit<HTMLInputAttributes, 'value' | 'size'> & {
    value?: string;
    element?: HTMLInputElement;
    icon?: IconName;
    kbd?: string;
    label?: string;
    mono?: boolean;
    size?: 'md' | 'sm';
    prefix?: string;
  } = $props();
</script>

<label class="field" class:mono class:sm={size === 'sm'}>
  {#if icon}<Icon name={icon} size={14} />{/if}
  {#if prefix}<span class="prefix">{prefix}</span>{/if}
  <input bind:this={element} bind:value aria-label={label} spellcheck="false" autocomplete="off" {...rest} />
  {#if kbd}<span class="kbd">{kbd}</span>{/if}
</label>

<style>
  .field {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 34px;
    min-width: 0;
    padding: 0 10px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius);
    color: var(--text-4);
    transition: border-color 0.12s, box-shadow 0.12s;
  }
  .field.sm { height: 30px; border-radius: 9px; }
  .field:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
  .field input { flex: 1; min-width: 0; height: 100%; padding: 0; background: transparent; border: 0; outline: none; color: var(--text); font-size: 13px; }
  .field input::placeholder { color: var(--text-4); }
  .mono input { font-family: var(--font-mono); font-size: 12.5px; }
  .prefix { font-family: var(--font-mono); font-size: 12px; color: var(--text-3); }
  .kbd { background: transparent; padding: 0; }
</style>
