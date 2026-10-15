<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import Icon, { type IconName } from './Icon.svelte';

  type Variant = 'default' | 'primary' | 'outline' | 'ghost' | 'danger';

  let {
    variant = 'default',
    size = 'md',
    icon,
    kbd,
    label,
    children,
    class: className = '',
    type = 'button',
    ...rest
  }: Omit<HTMLButtonAttributes, 'children'> & {
    variant?: Variant;
    size?: 'md' | 'sm';
    icon?: IconName;
    kbd?: string;
    label?: string;
    children?: Snippet;
  } = $props();
</script>

<button {type} class="btn {variant} {className}" class:sm={size === 'sm'} class:icon={!children} aria-label={label} {...rest}>
  {#if icon}<Icon name={icon} size={size === 'sm' ? 13 : 14} />{/if}
  {#if children}{@render children()}{/if}
  {#if kbd}<span class="kbd">{kbd}</span>{/if}
</button>
