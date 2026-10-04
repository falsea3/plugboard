<script lang="ts" module>
  export type SelectOption<T> = { value: T; label: string; hint?: string; disabled?: boolean };
</script>

<script lang="ts" generics="T extends string | number">
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';
  import SelectPopup from './SelectPopup.svelte';

  let {
    value = $bindable(),
    options,
    onchange,
    onclose,
    placeholder = '',
    id,
    class: className = '',
    startOpen = false,
    'aria-label': ariaLabel,
  }: {
    value: T;
    options: SelectOption<T>[];
    onchange?: (value: T) => void;
    onclose?: (picked: boolean, key?: KeyboardEvent) => void;
    placeholder?: string;
    id?: string;
    class?: string;
    startOpen?: boolean;
    'aria-label'?: string;
  } = $props();

  const MAX_HEIGHT = 280;

  let button = $state<HTMLButtonElement>();
  let open = $state(false);
  let pos = $state({ left: 0, top: 0, width: 0, maxHeight: MAX_HEIGHT, above: false });

  const current = $derived(options.find(o => o.value === value));

  function show() {
    if (open || !button) return;
    const r = button.getBoundingClientRect();
    const below = window.innerHeight - r.bottom - 8;
    const above = r.top - 8;
    const flip = below < Math.min(MAX_HEIGHT, 160) && above > below;
    pos = {
      left: r.left,
      top: flip ? r.top - 4 : r.bottom + 4,
      width: r.width,
      maxHeight: Math.min(MAX_HEIGHT, flip ? above : below),
      above: flip,
    };
    open = true;
  }

  function hide(picked: boolean, key?: KeyboardEvent) {
    if (!open) return;
    open = false;
    onclose?.(picked, key);
    const at = document.activeElement;
    if (!at || at === document.body || at.closest('.select-popup')) button?.focus();
  }

  function pick(o: SelectOption<T>) {
    if (o.disabled) return;
    const changed = o.value !== value;
    value = o.value;
    if (changed) onchange?.(o.value);
    hide(true);
  }

  function onkeydown(e: KeyboardEvent) {
    if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(e.key)) {
      e.preventDefault();
      show();
    }
  }

  onMount(() => {
    if (startOpen) show();
  });
</script>

<button
  bind:this={button}
  {id}
  type="button"
  class="select-button {className}"
  class:open
  aria-haspopup="listbox"
  aria-expanded={open}
  aria-label={ariaLabel}
  onclick={() => (open ? hide(false) : show())}
  {onkeydown}
>
  <span class="value" class:placeholder={!current}>{current?.label ?? placeholder}</span>
  <Icon name="chevron-down" size={12} />
</button>

{#if open}
  <SelectPopup {options} {value} at={pos} label={ariaLabel} onpick={pick} oncancel={key => hide(false, key)} />
{/if}

<style>
  .select-button {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 28px;
    width: 100%;
    min-width: 0;
    padding: 0 8px 0 9px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font-size: 12.5px;
    text-align: left;
    outline: none;
    transition: border-color 0.12s, box-shadow 0.12s;
  }
  .select-button:hover { border-color: var(--text-3); }
  .select-button:focus-visible, .select-button.open {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-dim);
  }
  .select-button > :global(.icon) { color: var(--text-3); }
  .value { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .value.placeholder { color: var(--text-3); }
</style>
