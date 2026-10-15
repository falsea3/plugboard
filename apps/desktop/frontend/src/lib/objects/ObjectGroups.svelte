<script lang="ts">
  import { SvelteSet } from 'svelte/reactivity';
  import type { DBObject } from '../api/wire';
  import type { Workspace } from '../app/workspace.svelte';
  import Icon from '../ui/Icon.svelte';
  import { groupObjects, hintOf, iconOf, keyOf } from './kinds';

  let { ws, filter, onmenu }: { ws: Workspace; filter: string; onmenu: (e: MouseEvent, o: DBObject) => void } = $props();

  const groups = $derived(groupObjects(ws.objects, filter));
  const openGroups = new SvelteSet<string>();
  const openItems = new SvelteSet<string>();
  const active = $derived(ws.activeTab?.kind === 'ddl' ? keyOf(ws.activeTab.object) : '');

  const toggle = (set: SvelteSet<string>, key: string) => (set.has(key) ? set.delete(key) : set.add(key));
</script>

{#if groups.length > 0}
  <div class="heading"><span>Objects</span><span class="count">{groups.reduce((n, g) => n + g.items.length, 0)}</span></div>
{/if}
{#each groups as g (g.label)}
  {@const open = openGroups.has(g.label) || !!filter.trim()}
  <button class="item folder" aria-expanded={open} onclick={() => toggle(openGroups, g.label)}>
    <span class="twist" class:open><Icon name="chevron-right" size={12} /></span>
    <Icon name={g.icon} size={14} />
    <span class="label">{g.label}</span>
    <span class="hint">{g.items.length}</span>
  </button>
  {#if open}
    {#each g.items as o (keyOf(o))}
      {@const key = keyOf(o)}
      <button class="item nested" class:active={active === key} onclick={() => ws.openDDL(o)} oncontextmenu={e => onmenu(e, o)} title="{o.name} {hintOf(o)}">
        {#if o.values?.length}
          <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
          <span class="twist" class:open={openItems.has(key)} onclick={e => (e.stopPropagation(), toggle(openItems, key))}><Icon name="chevron-right" size={12} /></span>
        {:else}
          <span class="twist"></span>
        {/if}
        <Icon name={iconOf(o)} size={14} />
        <span class="label">{o.name}</span>
        <span class="hint">{hintOf(o)}</span>
      </button>
      {#if openItems.has(key)}
        {#each o.values ?? [] as v, i (i)}<div class="value">{v}</div>{/each}
      {/if}
    {/each}
  {/if}
{/each}
{#if ws.objectsError}<div class="note">{ws.objectsError}</div>{/if}

<style>
  .heading { display: flex; justify-content: space-between; padding: 18px 8px 8px; color: var(--text-3); font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; }
  .count { font-weight: 500; }
  .item.nested { padding-left: 22px; }
  .item.folder .twist :global(.icon), .item .twist :global(.icon) { color: var(--text-4); }
  .item { width: 100%; display: flex; align-items: center; gap: 6px; height: 30px; padding: 0 8px; border: 0; border-radius: 8px; background: transparent; color: var(--text); font-size: 13px; text-align: left; }
  .item:hover { background: var(--hover); color: var(--text); }
  .item.active { background: var(--tree-selected); color: var(--accent-text); font-weight: 500; }
  .item.active :global(.icon) { color: var(--accent-text); }
  .item :global(.icon) { color: var(--text-3); }
  .twist { flex: none; display: flex; align-items: center; justify-content: center; width: 12px; height: 18px; border-radius: 4px; transition: transform 0.12s; }
  .twist.open { transform: rotate(90deg); }
  .item .twist:hover { background: var(--active); }
  .label { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .hint { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-4); font-size: 11.5px; text-align: right; }
  .value { height: 22px; padding: 0 8px 0 60px; color: var(--text-2); font-family: var(--font-mono); font-size: 11.5px; line-height: 22px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .note { padding: 8px; color: var(--danger); font-size: 11.5px; }
</style>
