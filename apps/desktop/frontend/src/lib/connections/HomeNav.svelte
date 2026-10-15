<script lang="ts">
  import { app } from '../app/app.svelte';
  import { layout } from '../app/layout.svelte';
  import { ENGINES } from '../engines';
  import { GROUP_TONES, type Scope } from './home';
  import DriverMark from './DriverMark.svelte';
  import Icon from '../ui/Icon.svelte';

  let { scope = $bindable(), onnewgroup }: { scope: Scope; onnewgroup: () => void } = $props();

  const groups = $derived(app.groups.sections.filter(s => s.name || app.groups.sections.length > 1));
  const engines = $derived(
    ENGINES.map(e => ({ e, count: app.connections.filter(c => c.driver === e.driver).length })).filter(x => x.count > 0),
  );
  const back = $derived(app.workspaces.find(w => w.id === app.activeId) ?? app.workspaces[0]);

  function same(a: Scope, b: Scope) {
    return JSON.stringify(a) === JSON.stringify(b);
  }
</script>

<aside class="nav" aria-label="Connection groups">
  <div class="top drag" class:full={layout.fullscreen} class:empty={layout.fullscreen && !back}>
    <span class="lights"></span>
    {#if back}
      <button class="back no-drag" onclick={() => app.activate(back)} title="Back to {back.connection.name} (⇧⌘H)">
        <Icon name="chevron-left" size={12} />{back.connection.name}
      </button>
    {/if}
  </div>
  <h1>Connections</h1>

  <nav aria-label="Groups">
    <button class="item" class:on={scope.kind === 'all'} aria-current={scope.kind === 'all'} onclick={() => (scope = { kind: 'all' })}>
      <span class="dot none"></span><span class="label">All</span><span class="count">{app.connections.length}</span>
    </button>
    {#each groups as g, i (g.name)}
      {@const s = { kind: 'group', name: g.name } as Scope}
      <button class="item" class:on={same(scope, s)} aria-current={same(scope, s)} onclick={() => (scope = s)}>
        <span class="dot" style:background={g.name ? GROUP_TONES[i % GROUP_TONES.length] : 'var(--text-4)'}></span><span class="label">{g.name || 'Ungrouped'}</span><span class="count">{g.items.length}</span>
      </button>
    {/each}

    {#if engines.length > 1}
      <div class="heading">Engines</div>
      {#each engines as { e, count } (e.driver)}
        {@const s = { kind: 'engine', driver: e.driver } as Scope}
        <button class="item engine" class:on={same(scope, s)} aria-current={same(scope, s)} onclick={() => (scope = s)}>
          <DriverMark driver={e.driver} size={20} /><span class="label">{e.name}</span><span class="count">{count}</span>
        </button>
      {/each}
    {/if}
  </nav>

  <div class="bottom">
    <button class="btn new-group" onclick={onnewgroup}>New group</button>
    <button class="btn icon" onclick={() => (app.settingsOpen = 'general')} title="Settings (⌘,)" aria-label="Settings"><Icon name="settings" size={15} /></button>
  </div>
</aside>

<style>
  .nav { width: 232px; flex: none; display: flex; flex-direction: column; margin: 8px 0 8px 8px; border: 1px solid var(--hover); border-radius: var(--radius-xl); background: var(--elevated); }
  .top { display: flex; align-items: center; gap: 8px; height: 33px; padding: 0 10px 0 0; }
  .lights { width: 64px; flex: none; }
  .top.full .lights { display: none; }
  .top.empty { height: 8px; }
  .back { min-width: 0; display: flex; align-items: center; gap: 4px; height: 26px; margin-left: auto; padding: 0 8px; border: 0; border-radius: 8px; background: transparent; color: var(--text-3); font-size: 12px; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
  .back:hover { background: var(--hover); color: var(--text); }
  h1 { margin: 0; padding: 22px 16px 0; font-size: 18px; font-weight: 600; letter-spacing: -0.01em; }
  nav { flex: 1; min-height: 0; overflow-y: auto; padding: 14px 8px 0; }
  .item { width: 100%; display: flex; align-items: center; gap: 8px; height: 30px; padding: 0 8px; border: 0; border-radius: 8px; background: transparent; color: var(--text-soft); text-align: left; }
  .item:hover { background: var(--hover); color: var(--text); }
  .item.on { background: var(--accent-bg); color: var(--tab-active); }
  .label { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .count { font-size: 11.5px; color: var(--text-4); }
  .item.on .count { color: var(--accent-text); }
  .dot { width: 8px; height: 8px; flex: none; border-radius: 3px; }
  .dot.none { background: transparent; }
  .engine { color: var(--text-2); height: 28px; }
  .engine .count { color: var(--gutter); }
  .item :global(.mark) { border-radius: 6px; }
  .heading { padding: 18px 8px 8px; font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; color: var(--text-3); }
  .bottom { display: flex; gap: 6px; padding: 10px 12px 12px; }
  .new-group { flex: 1; }
</style>
