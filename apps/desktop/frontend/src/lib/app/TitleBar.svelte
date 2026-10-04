<script lang="ts">
  import { tick } from 'svelte';
  import { app } from './app.svelte';
  import { connectionTarget } from '../ui/format';
  import Icon from '../ui/Icon.svelte';

  const envColor = $derived(app.active?.connection.env ? `var(--env-${app.active.connection.env})` : 'transparent');

  let sessions = $state<HTMLElement>();
  let moreLeft = $state(false);
  let moreRight = $state(false);

  function measure() {
    if (!sessions) return;
    moreLeft = sessions.scrollLeft > 1;
    moreRight = sessions.scrollLeft + sessions.clientWidth < sessions.scrollWidth - 1;
  }

  $effect(() => {
    if (!sessions) return;
    const observer = new ResizeObserver(measure);
    observer.observe(sessions);
    for (const child of sessions.children) observer.observe(child);
    return () => observer.disconnect();
  });

  $effect(() => {
    app.workspaces.length;
    tick().then(measure);
  });

  $effect(() => {
    const id = app.active?.id;
    if (!id) return;
    tick().then(() => sessions?.querySelector(`[data-ws="${id}"]`)?.scrollIntoView({ block: 'nearest', inline: 'nearest' }));
  });

  function onwheel(e: WheelEvent) {
    if (!sessions || Math.abs(e.deltaY) <= Math.abs(e.deltaX)) return;
    sessions.scrollLeft += e.deltaY;
    e.preventDefault();
  }
</script>

<header class="titlebar drag" style:--stripe={envColor}>
  <div class="traffic-space"></div>

  <button
    class="home no-drag"
    class:on={!app.active}
    onclick={() => (app.active ? app.goHome() : app.leaveHome())}
    title="All connections (⇧⌘H)"
    aria-label="All connections"
  >
    <Icon name="database" size={14} />
  </button>

  <nav class="sessions no-drag" class:more-left={moreLeft} class:more-right={moreRight} aria-label="Open connections" bind:this={sessions} {onwheel} onscroll={measure}>
    {#each app.workspaces as ws (ws.id)}
      {@const c = ws.connection}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="pill env-{c.env || 'none'}"
        class:active={app.active === ws}
        data-ws={ws.id}
        onmousedown={e => e.button === 1 && e.preventDefault()}
        onauxclick={e => e.button === 1 && app.requestClose(ws)}
      >
        <button class="pill-main" onclick={() => app.activate(ws)} title="{c.name} — {connectionTarget(c)}">
          <span class="dot" class:tunnel-down={ws.tunnel !== 'ok'} title={ws.tunnel !== 'ok' ? 'SSH connection lost — reconnecting' : undefined}></span>
          <span class="name">{c.name}</span>
          {#if ws.tabs.length > 0}<span class="count" title="{ws.tabs.length} open {ws.tabs.length === 1 ? 'tab' : 'tabs'}">{ws.tabs.length}</span>{/if}
          {#if ws.readOnly}<span class="pill-lock" title="Read-only"><Icon name="lock" size={10} /></span>{/if}
        </button>
        <button class="pill-close" onclick={() => app.requestClose(ws)} title="Close connection (⇧⌘W or middle-click)" aria-label="Close {c.name}">
          <Icon name="x" size={10} />
        </button>
      </div>
    {/each}
  </nav>
  <button class="add no-drag" onclick={() => (app.switcherOpen = true)} title="Switch connection (⌘K)" aria-label="Switch connection">
    <Icon name="plus" size={13} />
  </button>

  <span class="spacer"></span>

  <button class="switch no-drag" onclick={() => (app.switcherOpen = true)} title="Switch connection">
    <Icon name="search" size={12} />
    <span>Switch connection</span>
    <span class="kbd">⌘K</span>
  </button>
  <button class="icon-btn no-drag" onclick={() => (app.settingsOpen = 'general')} title="Settings (⌘,)" aria-label="Settings">
    <Icon name="settings" size={15} />
  </button>
</header>

<style>
  .titlebar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    height: var(--titlebar-h);
    padding: 0 8px 0 10px;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
    box-shadow: inset 0 2px 0 var(--stripe);
    min-width: 0;
  }
  .traffic-space { flex: none; width: 0; }
  :global([data-platform='darwin']) .traffic-space { width: 64px; }

  .home, .icon-btn, .add {
    flex: none;
    display: grid;
    place-items: center;
    width: 28px;
    height: 26px;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: var(--text-2);
  }
  .home:hover, .icon-btn:hover, .add:hover { background: var(--hover); color: var(--text); }
  .home.on { background: var(--accent-dim); color: var(--accent); }
  .add { width: 24px; height: 24px; color: var(--text-3); }

  .sessions {
    display: flex;
    align-items: center;
    gap: 4px;
    min-width: 0;
    padding: 0 2px;
    overflow-x: auto;
    scroll-padding-inline: 28px;
    scrollbar-width: none;
  }
  .sessions::-webkit-scrollbar { display: none; }
  .sessions.more-left { -webkit-mask-image: linear-gradient(to right, transparent, #000 32px); mask-image: linear-gradient(to right, transparent, #000 32px); }
  .sessions.more-right { -webkit-mask-image: linear-gradient(to left, transparent, #000 32px); mask-image: linear-gradient(to left, transparent, #000 32px); }
  .sessions.more-left.more-right {
    -webkit-mask-image: linear-gradient(to right, transparent, #000 32px, #000 calc(100% - 32px), transparent);
    mask-image: linear-gradient(to right, transparent, #000 32px, #000 calc(100% - 32px), transparent);
  }

  .pill {
    flex: 0 1 auto;
    display: flex;
    align-items: center;
    height: 26px;
    min-width: 96px;
    max-width: 220px;
    border-radius: 6px;
    color: var(--text-2);
  }
  .pill:hover { background: var(--hover); }
  .pill.active {
    flex-shrink: 0;
    background: var(--elevated);
    color: var(--text);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.12);
  }
  .pill-main {
    flex: 1 1 auto;
    display: flex;
    align-items: center;
    gap: 7px;
    min-width: 0;
    height: 100%;
    padding: 0 2px 0 10px;
    border: 0;
    background: transparent;
    color: inherit;
    font-size: 12.5px;
    font-weight: 500;
  }
  .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .pill-lock { display: flex; color: var(--text-3); }
  .count { flex: none; min-width: 16px; padding: 0 4px; border-radius: 8px; background: var(--hover); color: var(--text-3); font-size: 10.5px; line-height: 15px; text-align: center; }
  .pill.active .count { background: var(--active); color: var(--text-2); }
  .dot {
    flex: none;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--env-color, var(--ok));
  }
  .dot.tunnel-down { background: var(--warn); animation: blink 1s ease-in-out infinite; }
  @keyframes blink { 50% { opacity: 0.25; } }
  .env-local { --env-color: var(--env-local); }
  .env-dev { --env-color: var(--env-dev); }
  .env-staging { --env-color: var(--env-staging); }
  .env-prod { --env-color: var(--env-prod); }
  .pill-close {
    flex: none;
    display: grid;
    place-items: center;
    width: 18px;
    height: 18px;
    margin: 0 4px 0 2px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--text-3);
    opacity: 0;
  }
  .pill:hover .pill-close, .pill.active .pill-close { opacity: 1; }
  .pill-close:hover { background: var(--hover); color: var(--text); }

  .spacer { flex: 1; min-width: 24px; align-self: stretch; }

  .switch {
    flex: none;
    display: flex;
    align-items: center;
    gap: 7px;
    height: 26px;
    padding: 0 6px 0 9px;
    border: 0;
    border-radius: 6px;
    background: var(--elevated);
    color: var(--text-3);
    font-size: 12px;
  }
  .switch:hover { color: var(--text-2); background: var(--hover); }
  @media (max-width: 1100px) {
    .switch span:not(.kbd) { display: none; }
  }
</style>
