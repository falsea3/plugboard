<script lang="ts" generics="T extends string | number">
  import { onMount, tick } from 'svelte';
  import type { SelectOption } from './Select.svelte';
  import Icon from './Icon.svelte';

  let {
    options,
    value,
    at,
    label,
    onpick,
    oncancel,
  }: {
    options: SelectOption<T>[];
    value: T;
    at: { left: number; top: number; width: number; maxHeight: number; above: boolean };
    label?: string;
    onpick: (o: SelectOption<T>) => void;
    oncancel: (key?: KeyboardEvent) => void;
  } = $props();

  const SEARCH_FROM = 20;

  let list = $state<HTMLDivElement>();
  let search = $state<HTMLInputElement>();
  let query = $state('');
  let active = $state(-1);
  let typed = '';
  let typedAt = 0;

  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase();
    return q ? options.filter(o => o.label.toLowerCase().includes(q)) : options;
  });
  const searchable = $derived(options.length >= SEARCH_FROM);

  onMount(async () => {
    active = Math.max(0, shown.findIndex(o => o.value === value));
    await tick();
    (searchable ? search : list)?.focus();
    scrollToActive();
  });

  function move(delta: number) {
    if (shown.length === 0) return;
    let i = active;
    for (let n = 0; n < shown.length; n++) {
      i = (i + delta + shown.length) % shown.length;
      if (!shown[i].disabled) break;
    }
    active = i;
    scrollToActive();
  }

  async function scrollToActive() {
    await tick();
    list?.querySelector<HTMLElement>(`[data-index="${active}"]`)?.scrollIntoView({ block: 'nearest' });
  }

  function onkeydown(e: KeyboardEvent) {
    e.stopPropagation();
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      move(e.key === 'ArrowDown' ? 1 : -1);
    } else if (e.key === 'Home' || e.key === 'End') {
      e.preventDefault();
      active = e.key === 'Home' ? -1 : shown.length;
      move(e.key === 'Home' ? 1 : -1);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (shown[active]) onpick(shown[active]);
    } else if (e.key === 'Escape' || e.key === 'Tab') {
      if (e.key === 'Escape') e.preventDefault();
      oncancel(e);
    } else if (!searchable && e.key.length === 1 && !e.metaKey && !e.ctrlKey) {
      const now = Date.now();
      typed = now - typedAt < 700 ? typed + e.key.toLowerCase() : e.key.toLowerCase();
      typedAt = now;
      const i = shown.findIndex(o => o.label.toLowerCase().startsWith(typed));
      if (i >= 0) {
        active = i;
        scrollToActive();
      }
    }
  }

  function onSearch(e: Event) {
    query = (e.currentTarget as HTMLInputElement).value;
    active = shown.findIndex(o => !o.disabled);
  }

  function toBody(node: HTMLElement) {
    document.body.appendChild(node);
    return { destroy: () => node.remove() };
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="backdrop" use:toBody onmousedown={() => oncancel()}></div>
<div
  use:toBody
  class="popup select-popup"
  class:above={at.above}
  style:left="{at.left}px"
  style:top="{at.top}px"
  style:min-width="{at.width}px"
  style:max-height="{at.maxHeight}px"
>
  {#if searchable}
    <input bind:this={search} value={query} oninput={onSearch} class="search" placeholder="Search…" spellcheck="false" autocomplete="off" aria-label="Search the options" {onkeydown} />
  {/if}
  <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
  <div bind:this={list} class="options" role="listbox" tabindex="-1" aria-label={label} {onkeydown}>
    {#each shown as o, i (o.value)}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div
        class="option"
        class:active={i === active}
        class:chosen={o.value === value}
        class:disabled={o.disabled}
        class:muted={o.muted}
        role="option"
        aria-selected={o.value === value}
        aria-disabled={o.disabled}
        data-index={i}
        tabindex="-1"
        onmousemove={() => !o.disabled && (active = i)}
        onclick={() => !o.disabled && onpick(o)}
      >
        <span class="check">{#if o.value === value}<Icon name="check-mark" size={12} />{/if}</span>
        <span class="label">{o.label}</span>
        {#if o.hint}<span class="hint">{o.hint}</span>{/if}
      </div>
    {:else}
      <div class="empty">Nothing matches “{query}”</div>
    {/each}
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; z-index: 200; }
  .popup {
    position: fixed;
    z-index: 201;
    display: flex;
    flex-direction: column;
    max-width: 360px;
    padding: 4px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--elevated);
    box-shadow: var(--shadow-modal);
    animation: drop 0.1s ease-out;
  }
  .popup.above { transform: translateY(-100%); animation-name: rise; }
  .search {
    flex: none;
    height: 26px;
    margin-bottom: 4px;
    padding: 0 8px;
    border: 1px solid var(--border);
    border-radius: 5px;
    background: var(--surface);
    font-size: 12px;
    outline: none;
  }
  .search:focus { border-color: var(--accent); }
  .options { overflow-y: auto; outline: none; }
  .option {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 26px;
    padding: 0 10px 0 4px;
    border-radius: 5px;
    font-size: 12.5px;
    white-space: nowrap;
  }
  .option.active { background: var(--accent); color: var(--on-accent); }
  .option.disabled, .option.muted { color: var(--text-3); }
  .check { flex: none; display: inline-flex; width: 14px; justify-content: center; }
  .label { flex: 1; overflow: hidden; text-overflow: ellipsis; }
  .hint { color: var(--text-3); font-size: 11.5px; }
  .option.active .hint { color: inherit; opacity: 0.8; }
  .empty { padding: 6px 10px; color: var(--text-3); font-size: 12px; }
  @keyframes drop { from { opacity: 0; transform: translateY(-3px); } }
  @keyframes rise { from { opacity: 0; transform: translateY(calc(-100% + 3px)); } }
</style>
