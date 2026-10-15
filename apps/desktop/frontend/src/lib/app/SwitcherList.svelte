<script lang="ts">
  import { keys } from '../ui/keys';
  import { app } from './app.svelte';
  import { engine } from '../engines';
  import type { Connection } from '../api/wire';
  import type { Section } from '../connections/groups';
  import { GroupDrag } from '../connections/groupDrag.svelte';
  import { connectionTarget } from '../ui/format';
  import { usedAgo } from '../connections/home';
  import ConnAvatar from '../connections/ConnAvatar.svelte';
  import EnvBadge from '../connections/EnvBadge.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let {
    secs,
    selected,
    arranging,
    onchoose,
    onselect,
    onrename,
  }: {
    secs: Section[];
    selected: string;
    arranging: boolean;
    onchoose: (c: Connection) => void;
    onselect: (id: string) => void;
    onrename: (group: string) => void;
  } = $props();

  let root = $state<HTMLDivElement>();
  const drag = new GroupDrag({ root: () => root, sections: () => app.groups.sections, apply: s => app.groups.apply(s) });
  const heads = $derived(secs.length > 1 || !!secs[0]?.name);

  function tabs(c: Connection) {
    const ws = app.workspaceFor(c.id);
    return ws ? ws.tabs.length : -1;
  }

  export function scrollTo(id: string) {
    root?.querySelector(`[data-conn="${CSS.escape(id)}"]`)?.scrollIntoView({ block: 'nearest' });
  }
</script>

<svelte:window onpointermove={drag.pointermove} onpointerup={drag.pointerup} onpointercancel={drag.pointercancel} />

<div class="tree" role="tree" aria-label="Connections" bind:this={root} class:dragging={!!drag.dragging}>
  {#each secs as s (s.name)}
    {@const open = s.items.filter(c => app.workspaceFor(c.id)).length}
    <div class="section" data-section={s.name} role="group" aria-label={s.name || 'Ungrouped'}>
      {#if heads}
        <div
          class="head"
          class:lifted={drag.dragging?.kind === 'group' && drag.dragging.id === s.name}
          data-head={s.name}
          role="treeitem"
          aria-selected="false"
          aria-expanded={!s.collapsed}
          tabindex="-1"
          onpointerdown={e => arranging && s.name && drag.down(e, 'group', s.name)}
          onclick={e => drag.clicked(e, 'group', s.name) && arranging && app.groups.toggle(s.name)}
          onkeydown={() => {}}
        >
          <Icon name={s.collapsed ? 'chevron-right' : 'chevron-down'} size={11} />
          <span class="gname">{s.name || 'Ungrouped'}</span>
          {#if open}<span class="open">{open} open</span>{/if}
          <span class="count">{s.items.length}</span>
          {#if s.name && arranging}
            <span class="tools">
              <button title="Rename group" aria-label="Rename group {s.name}" onpointerdown={e => e.stopPropagation()} onclick={e => { e.stopPropagation(); onrename(s.name); }}><Icon name="pencil" size={12} /></button>
              <button title="Remove group — its connections stay, ungrouped" aria-label="Remove group {s.name}" onpointerdown={e => e.stopPropagation()} onclick={e => { e.stopPropagation(); app.groups.remove(s.name); }}><Icon name="x" size={12} /></button>
              <span class="grip"><Icon name="grip" size={12} /></span>
            </span>
          {/if}
        </div>
      {/if}
      {#if !s.collapsed}
        {#each s.items as c (c.id)}
          {@const n = tabs(c)}
          <div
            class="row"
            class:nested={heads}
            class:selected={c.id === selected}
            class:current={app.active?.connection.id === c.id}
            class:lifted={drag.dragging?.kind === 'conn' && drag.dragging.id === c.id}
            data-conn={c.id}
            data-in={s.name}
            role="treeitem"
            aria-selected={c.id === selected}
            tabindex="-1"
            onpointerdown={e => arranging && drag.down(e, 'conn', c.id)}
            onclick={e => drag.clicked(e, 'conn', c.id) && onchoose(c)}
            onkeydown={() => {}}
            onmousemove={() => !drag.dragging && onselect(c.id)}
          >
            {#if arranging}<span class="grip"><Icon name="grip" size={12} /></span>{/if}
            <ConnAvatar connection={c} size={30} />
            <span class="meta">
              <span class="name"><span class="text">{c.name}</span><EnvBadge env={c.env} />{#if c.readOnly}<span class="mark" title="Read-only"><Icon name="lock" size={10} /></span>{/if}</span>
              <span class="target">{engine(c.driver).name} · {connectionTarget(c)}</span>
            </span>
            {#if app.connectingId === c.id}
              <span class="state" role="status"><Spinner size={10} />Connecting…</span>
            {:else if n >= 0}
              <span class="state open-state">● {n === 0 ? 'open' : n === 1 ? '1 tab' : `${n} tabs`}</span>
              <button class="disconnect" title={keys('Disconnect (⇧⌘W)')} aria-label="Disconnect {c.name}" onpointerdown={e => e.stopPropagation()} onclick={e => { e.stopPropagation(); const ws = app.workspaceFor(c.id); if (ws) app.requestClose(ws); }}><Icon name="x" size={12} /></button>
            {:else if c.lastUsed}
              <span class="state">{usedAgo(c.lastUsed)}</span>
            {/if}
          </div>
        {/each}
      {/if}
    </div>
  {/each}
  {#if drag.line !== null}<div class="line" style:top="{drag.line - 1}px"></div>{/if}
</div>

<style>
  .tree { position: relative; padding: 6px; overflow-y: auto; max-height: min(460px, 60vh); }
  .tree.dragging { cursor: grabbing; }
  .section + .section { margin-top: 4px; }
  .head { display: flex; align-items: center; gap: 8px; height: 30px; padding: 0 8px; border-radius: 9px; color: var(--text-3); cursor: default; }
  .head:hover { background: var(--hover); }
  .gname { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text); font-weight: 600; font-size: 12.5px; }
  .open { padding: 0 6px; border-radius: 5px; background: var(--accent-bg); color: var(--accent-text); font-size: 11px; }
  .count { font-size: 11.5px; color: var(--text-4); }
  .tools { display: none; align-items: center; gap: 2px; }
  .head:hover .tools { display: flex; }
  .tools button { display: grid; place-items: center; width: 22px; height: 22px; border: 0; border-radius: 6px; background: transparent; color: var(--text-3); }
  .tools button:hover { background: var(--active); color: var(--text); }
  .grip { display: flex; color: var(--text-4); cursor: grab; }
  .row { --avatar-ring: var(--elevated); display: flex; align-items: center; gap: 9px; padding: 5px 8px; border-radius: 10px; cursor: default; }
  .row.nested { margin-left: 10px; }
  .row .grip { visibility: hidden; margin: 0 -4px 0 -6px; }
  .row:hover .grip { visibility: visible; }
  .row.selected { background: var(--hover); }
  .row.current { background: var(--accent-bg); }
  .lifted { opacity: 0.4; }
  .meta { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
  .name { display: flex; align-items: center; gap: 6px; min-width: 0; font-weight: 500; color: var(--text); }
  .text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .mark { display: flex; color: var(--text-3); }
  .target { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; color: var(--text-3); }
  .state { flex: none; display: flex; align-items: center; gap: 5px; font-size: 11.5px; color: var(--text-3); }
  .open-state { color: var(--text-2); }
  .disconnect { display: none; place-items: center; width: 22px; height: 22px; flex: none; border: 0; border-radius: 6px; background: transparent; color: var(--text-3); }
  .row:hover .disconnect { display: grid; }
  .disconnect:hover { background: var(--active); color: var(--text); }
  .row.current .open-state { color: var(--accent-text); }
  .line { position: absolute; left: 14px; right: 14px; height: 2px; border-radius: 1px; background: var(--accent-hover); pointer-events: none; }
</style>
