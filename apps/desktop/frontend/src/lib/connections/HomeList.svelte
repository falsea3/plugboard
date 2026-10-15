<script lang="ts">
  import { app } from '../app/app.svelte';
  import { engine } from '../engines';
  import type { Connection } from '../api/wire';
  import type { Section } from './groups';
  import { GroupDrag } from './groupDrag.svelte';
  import { connectionUrl, usedAgo } from './home';
  import DriverMark from './DriverMark.svelte';
  import EnvBadge from './EnvBadge.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let {
    secs,
    selected = $bindable(),
    arranging,
    onkeydown,
    onrename,
  }: {
    secs: Section[];
    selected: string;
    arranging: boolean;
    onkeydown: (e: KeyboardEvent) => void;
    onrename: (group: string) => void;
  } = $props();

  let root = $state<HTMLDivElement>();
  const drag = new GroupDrag({ root: () => root, sections: () => app.groups.sections, apply: s => app.groups.apply(s) });
  const heads = $derived(secs.length > 1 || !!secs[0]?.name);

  function server(c: Connection) {
    return app.workspaceFor(c.id)?.session.serverVersion || engine(c.driver).name;
  }

  export function scrollTo(id: string) {
    root?.querySelector(`[data-conn="${CSS.escape(id)}"]`)?.scrollIntoView({ block: 'nearest' });
  }
</script>

<svelte:window onpointermove={drag.pointermove} onpointerup={drag.pointerup} onpointercancel={drag.pointercancel} />

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div class="list" role="tree" aria-label="Connections" tabindex="0" bind:this={root} {onkeydown} class:dragging={!!drag.dragging}>
  <div class="cols" aria-hidden="true"><span></span><span>Name</span><span>Connection</span><span>Server</span><span class="right">Last used</span></div>
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
          <Icon name={s.collapsed ? 'chevron-right' : 'chevron-down'} size={10} />
          <span class="gname">{s.name || 'Ungrouped'}</span>
          <span class="count">{s.items.length}</span>
          {#if open}<span class="open">{open} open</span>{/if}
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
          {@const live = !!app.workspaceFor(c.id)}
          {@const url = connectionUrl(c)}
          <div
            class="row"
            class:selected={c.id === selected}
            class:lifted={drag.dragging?.kind === 'conn' && drag.dragging.id === c.id}
            data-conn={c.id}
            data-in={s.name}
            role="treeitem"
            aria-selected={c.id === selected}
            tabindex="-1"
            onpointerdown={e => arranging && drag.down(e, 'conn', c.id)}
            onclick={e => drag.clicked(e, 'conn', c.id) && (selected = c.id)}
            ondblclick={() => app.open(c)}
            onkeydown={() => {}}
          >
            <span class="avatar">{#if arranging}<span class="grip"><Icon name="grip" size={12} /></span>{/if}<DriverMark driver={c.driver} size={24} /></span>
            <span class="name"><span class="live" class:on={live}></span><span class="text">{c.name}</span><EnvBadge env={c.env} />{#if c.readOnly}<span class="mark" title="Read-only"><Icon name="lock" size={10} /></span>{/if}</span>
            <span class="url">{url.address}{url.database}</span>
            <span class="server">{server(c)}</span>
            <span class="state" class:on={live}>
              {#if app.connectingId === c.id}<Spinner size={11} label="Connecting" />{:else if live}open{:else}{usedAgo(c.lastUsed)}{/if}
            </span>
          </div>
        {/each}
      {/if}
    </div>
  {/each}
  {#if drag.line !== null}<div class="line" style:top="{drag.line - 1}px"></div>{/if}
</div>

<style>
  .list { position: relative; flex: 1; min-width: 0; overflow-y: auto; padding-bottom: 8px; border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--card); outline: none; }
  .list.dragging { cursor: grabbing; }
  .cols, .row { display: grid; grid-template-columns: 40px minmax(0, 1.2fr) minmax(0, 1.6fr) 110px 90px; align-items: center; }
  .cols { position: sticky; top: 0; z-index: 1; height: 36px; padding: 0 8px; border-bottom: 1px solid var(--border-subtle); background: var(--card); font-size: 11.5px; color: var(--text-3); }
  .right { text-align: right; padding-right: 8px; }
  .head { display: flex; align-items: center; gap: 8px; height: 34px; padding: 0 16px; color: var(--text-3); font-size: 11.5px; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase; }
  .head > :global(.icon) { color: var(--text-4); }
  .gname { color: var(--text-3); }
  .count { color: var(--text-4); font-weight: 400; }
  .open { padding: 0 6px; border-radius: 5px; background: var(--accent-bg); color: var(--accent-text); font-size: 11px; font-weight: 500; letter-spacing: 0; text-transform: none; }
  .tools { display: none; align-items: center; gap: 2px; margin-left: auto; }
  .head:hover .tools { display: flex; }
  .tools button { display: grid; place-items: center; width: 22px; height: 22px; border: 0; border-radius: 6px; background: transparent; color: var(--text-3); }
  .tools button:hover { background: var(--active); color: var(--text); }
  .grip { display: flex; color: var(--text-4); cursor: grab; }
  .row { height: 40px; margin: 0 6px; padding: 0 2px; border-radius: 9px; cursor: default; }
  .row:hover { background: var(--hover); }
  .row.selected { background: var(--tree-selected); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--accent) 45%, transparent); }
  .lifted { opacity: 0.4; }
  .avatar { position: relative; display: flex; justify-content: center; }
  .avatar :global(.mark) { border-radius: 7px; }
  .avatar .grip { position: absolute; left: -4px; top: 50%; transform: translateY(-50%); visibility: hidden; }
  .row:hover .avatar .grip { visibility: visible; }
  .name { display: flex; align-items: center; gap: 7px; min-width: 0; }
  .text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
  .live { width: 7px; height: 7px; flex: none; border-radius: 50%; }
  .live.on { background: var(--accent-text); }
  .mark { display: flex; color: var(--text-3); }
  .url { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding-right: 12px; font-family: var(--font-mono); font-size: 12px; color: var(--text-2); }
  .server { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; color: var(--text-3); }
  .state { display: flex; justify-content: flex-end; padding-right: 8px; font-size: 12px; color: var(--text-4); }
  .state.on { color: var(--accent-text); }
  .line { position: absolute; left: 14px; right: 14px; height: 2px; border-radius: 1px; background: var(--accent-hover); pointer-events: none; z-index: 2; }
</style>
