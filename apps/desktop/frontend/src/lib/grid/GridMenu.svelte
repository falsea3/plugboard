<script lang="ts">
  import { onDestroy } from 'svelte';
  import type { MenuItem } from './grid';
  import { menuIcon } from './menuIcons';
  import Icon from '../ui/Icon.svelte';

  let { items, x, y, onpick, onclose }: { items: MenuItem[]; x: number; y: number; onpick: (id: string) => void; onclose: () => void } = $props();

  const ITEM_H = 27;
  const WIDTH = 236;

  const SWITCH_DELAY = 250;

  let submenu = $state<number | null>(null);
  let timer: ReturnType<typeof setTimeout> | undefined;

  function want(i: number | null, now = false) {
    clearTimeout(timer);
    if (submenu === i) return;
    if (now || submenu === null) submenu = i;
    else timer = setTimeout(() => (submenu = i), SWITCH_DELAY);
  }

  onDestroy(() => clearTimeout(timer));
  const top = $derived(Math.max(8, Math.min(y, window.innerHeight - items.length * ITEM_H - 16)));
  const left = $derived(Math.max(8, Math.min(x, window.innerWidth - WIDTH)));
  const flip = $derived(left + 2 * WIDTH > window.innerWidth);
</script>

{#snippet mark(item: Exclude<MenuItem, 'sep'>)}
  {@const name = item.icon ?? menuIcon(item.id)}
  <span class="mi">{#if name}<Icon {name} size={14} />{/if}</span>
{/snippet}

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="menu-backdrop" onclick={onclose} oncontextmenu={e => { e.preventDefault(); onclose(); }}></div>
<div class="menu" role="menu" style:left="{left}px" style:top="{top}px">
  {#each items as it, i (i)}
    {#if it === 'sep'}
      <div class="sep"></div>
    {:else if it.items}
      {@const up = top + (i + it.items.length) * ITEM_H + 16 > window.innerHeight}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="sub-wrap" onmouseenter={() => want(i)} onmouseleave={() => want(null)}>
        <button role="menuitem" aria-haspopup="menu" aria-expanded={submenu === i} class:open={submenu === i} disabled={it.disabled} onclick={() => want(i, true)}>
          {@render mark(it)}{it.label}<span class="arrow"><Icon name="chevron-right" size={11} /></span>
        </button>
        {#if submenu === i}
          {@const childIcons = it.items.some(x => x !== 'sep' && !!(x.icon ?? menuIcon(x.id)))}
          <div class="menu sub" class:flip class:up role="menu">
            {#each it.items as child, j (j)}
              {#if child === 'sep'}
                <div class="sep"></div>
              {:else}
                <button role="menuitem" class:danger={child.danger} disabled={child.disabled} onclick={() => onpick(child.id)}>
                  {#if childIcons}{@render mark(child)}{/if}{child.label}{#if child.kbd}<span class="kbd">{child.kbd}</span>{/if}
                </button>
              {/if}
            {/each}
          </div>
        {/if}
      </div>
    {:else}
      <button role="menuitem" class:danger={it.danger} disabled={it.disabled} onclick={() => onpick(it.id)} onmouseenter={() => want(null)}>
        {@render mark(it)}{it.label}{#if it.kbd}<span class="kbd">{it.kbd}</span>{/if}
      </button>
    {/if}
  {/each}
</div>

<style>
  .menu-backdrop { position: fixed; inset: 0; z-index: 40; }
  .menu {
    position: fixed;
    z-index: 41;
    min-width: 190px;
    padding: 4px;
    border-radius: 8px;
    background: var(--elevated);
    box-shadow: var(--shadow-modal);
    font-family: var(--font-ui);
    font-size: 12.5px;
  }
  .menu button {
    display: flex;
    align-items: center;
    width: 100%;
    height: 26px;
    padding: 0 8px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text);
    text-align: left;
  }
  .menu button .kbd { margin-left: auto; }
  .menu .mi { flex: none; display: flex; width: 16px; margin-right: 8px; color: var(--text-2); }
  .menu button:hover:not(:disabled) .mi, .menu button.danger .mi { color: inherit; }
  .menu button:disabled .mi { color: var(--text-3); }
  .menu button:hover:not(:disabled) { background: var(--accent); color: var(--on-accent); }
  .menu button:hover:not(:disabled) .kbd { color: inherit; border-color: rgba(255, 255, 255, 0.4); }
  .menu button:disabled { color: var(--text-3); }
  .menu button.danger { color: var(--danger); }
  .menu button.danger:hover { color: var(--on-accent); background: var(--danger); }
  .menu .sep { height: 1px; margin: 4px 6px; background: var(--border-subtle); }
  .sub-wrap { position: relative; }
  .menu button .arrow { display: flex; margin-left: auto; color: var(--text-3); }
  .menu button.open { background: var(--hover); }
  .menu button:hover .arrow { color: inherit; }
  .menu.sub { position: absolute; left: calc(100% - 4px); top: -4px; }
  .menu.sub.flip { left: auto; right: calc(100% - 4px); }
  .menu.sub.up { top: auto; bottom: -4px; }
</style>
