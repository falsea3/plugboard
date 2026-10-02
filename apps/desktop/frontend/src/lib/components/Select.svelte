<script lang="ts" module>
  export type SelectOption<T> = { value: T; label: string; hint?: string; disabled?: boolean };
</script>

<script lang="ts" generics="T extends string | number">
  import { onMount, tick } from 'svelte';
  import Icon from './Icon.svelte';

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
    /**
     * The list closed; picked says whether an option was chosen (maybe the one
     * already set), key is the Escape or Tab that closed it.
     */
    onclose?: (picked: boolean, key?: KeyboardEvent) => void;
    /** shown when no option matches value */
    placeholder?: string;
    id?: string;
    class?: string;
    /** open on mount, for editors that appear in place (a grid cell) */
    startOpen?: boolean;
    'aria-label'?: string;
  } = $props();

  // Long lists (columns, schemas) get a search box at the top.
  const SEARCH_FROM = 20;
  const MAX_HEIGHT = 280;

  let button = $state<HTMLButtonElement>();
  let popup = $state<HTMLDivElement>();
  let list = $state<HTMLDivElement>();
  let search = $state<HTMLInputElement>();
  let open = $state(false);
  let query = $state('');
  let active = $state(-1);
  let pos = $state({ left: 0, top: 0, width: 0, maxHeight: MAX_HEIGHT, above: false });

  const current = $derived(options.find(o => o.value === value));
  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase();
    return q ? options.filter(o => o.label.toLowerCase().includes(q)) : options;
  });
  const searchable = $derived(options.length >= SEARCH_FROM);

  async function show() {
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
    query = '';
    active = Math.max(0, shown.findIndex(o => o.value === value));
    open = true;
    await tick();
    (searchable ? search : list)?.focus();
    scrollToActive();
  }

  function hide(picked: boolean, key?: KeyboardEvent) {
    if (!open) return;
    open = false;
    onclose?.(picked, key);
    // Focus goes back to the button, unless onchange or onclose moved it on
    // (a grid cell editor hands it back to the grid).
    const at = document.activeElement;
    if (!at || at === document.body || popup?.contains(at)) button?.focus();
  }

  function pick(o: SelectOption<T>) {
    if (o.disabled) return;
    const changed = o.value !== value;
    value = o.value;
    if (changed) onchange?.(o.value);
    hide(true);
  }

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

  function onButtonKey(e: KeyboardEvent) {
    if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(e.key)) {
      e.preventDefault();
      show();
    }
  }

  let typed = '';
  let typedAt = 0;

  function onListKey(e: KeyboardEvent) {
    e.stopPropagation(); // Escape here closes the list, not the dialog around it
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        move(1);
        return;
      case 'ArrowUp':
        e.preventDefault();
        move(-1);
        return;
      case 'Home':
      case 'End':
        e.preventDefault();
        active = e.key === 'Home' ? -1 : shown.length;
        move(e.key === 'Home' ? 1 : -1);
        return;
      case 'Enter':
        e.preventDefault();
        if (shown[active]) pick(shown[active]);
        return;
      case 'Escape':
        e.preventDefault();
        hide(false, e);
        return;
      case 'Tab':
        hide(false, e);
        return;
    }
    // Without a search box, typing jumps to the first option starting with it.
    if (!searchable && e.key.length === 1 && !e.metaKey && !e.ctrlKey) {
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

  onMount(() => {
    if (startOpen) show();
  });

  // The list lives in <body>: inside a transformed ancestor (the grid's rows)
  // position: fixed would be relative to that ancestor, not the window.
  function toBody(node: HTMLElement) {
    document.body.appendChild(node);
    return { destroy: () => node.remove() };
  }

  function onSearch(e: Event) {
    query = (e.currentTarget as HTMLInputElement).value;
    active = shown.findIndex(o => !o.disabled);
  }
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
  onkeydown={onButtonKey}
>
  <span class="value" class:placeholder={!current}>{current?.label ?? placeholder}</span>
  <Icon name="chevron-down" size={12} />
</button>

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="backdrop" use:toBody onmousedown={() => hide(false)}></div>
  <div
    bind:this={popup}
    use:toBody
    class="popup"
    class:above={pos.above}
    style:left="{pos.left}px"
    style:top="{pos.top}px"
    style:min-width="{pos.width}px"
    style:max-height="{pos.maxHeight}px"
  >
    {#if searchable}
      <input
        bind:this={search}
        value={query}
        oninput={onSearch}
        class="search"
        placeholder="Search…"
        spellcheck="false"
        autocomplete="off"
        aria-label="Search the options"
        onkeydown={onListKey}
      />
    {/if}
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <div bind:this={list} class="options" role="listbox" tabindex="-1" aria-label={ariaLabel} onkeydown={onListKey}>
      {#each shown as o, i (o.value)}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div
          class="option"
          class:active={i === active}
          class:chosen={o.value === value}
          class:disabled={o.disabled}
          role="option"
          aria-selected={o.value === value}
          aria-disabled={o.disabled}
          data-index={i}
          tabindex="-1"
          onmousemove={() => !o.disabled && (active = i)}
          onclick={() => pick(o)}
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
  .option.disabled { color: var(--text-3); }
  .check { flex: none; display: inline-flex; width: 14px; justify-content: center; }
  .label { flex: 1; overflow: hidden; text-overflow: ellipsis; }
  .hint { color: var(--text-3); font-size: 11.5px; }
  .option.active .hint { color: inherit; opacity: 0.8; }
  .empty { padding: 6px 10px; color: var(--text-3); font-size: 12px; }
  @keyframes drop { from { opacity: 0; transform: translateY(-3px); } }
  @keyframes rise { from { opacity: 0; transform: translateY(calc(-100% + 3px)); } }
</style>
