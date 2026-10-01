<script lang="ts">
  import { app } from '../stores/app.svelte';
  import { connectionTarget } from '../format';
  import Icon from './Icon.svelte';

  const envColor = $derived(app.active?.connection.env ? `var(--env-${app.active.connection.env})` : 'transparent');
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

  <nav class="sessions no-drag" aria-label="Open connections">
    {#each app.workspaces as ws (ws.id)}
      {@const c = ws.connection}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="pill env-{c.env || 'none'}"
        class:active={app.active === ws}
        onmousedown={e => e.button === 1 && e.preventDefault()}
        onauxclick={e => e.button === 1 && app.requestClose(ws)}
      >
        <button class="pill-main" onclick={() => app.activate(ws)} title="{c.name} — {connectionTarget(c)}">
          <span class="dot" class:tunnel-down={ws.tunnel !== 'ok'} title={ws.tunnel !== 'ok' ? 'SSH connection lost — reconnecting' : undefined}></span>
          <span class="name">{c.name}</span>
          {#if ws.readOnly}<span class="pill-lock" title="Read-only"><Icon name="lock" size={10} /></span>{/if}
        </button>
        <button class="pill-close" onclick={() => app.requestClose(ws)} title="Close connection (⇧⌘W or middle-click)" aria-label="Close {c.name}">
          <Icon name="x" size={10} />
        </button>
      </div>
    {/each}
    <button class="add" onclick={() => (app.switcherOpen = true)} title="Switch connection (⌘K)" aria-label="Switch connection">
      <Icon name="plus" size={13} />
    </button>
  </nav>

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
    overflow-x: auto;
    scrollbar-width: none;
  }
  .sessions::-webkit-scrollbar { display: none; }

  .pill {
    flex: none;
    display: flex;
    align-items: center;
    height: 26px;
    max-width: 200px;
    border: 1px solid transparent;
    border-radius: 6px;
    color: var(--text-2);
  }
  .pill:hover { background: var(--hover); }
  .pill.active {
    background: var(--bg);
    border-color: var(--border);
    color: var(--text);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.12);
  }
  .pill-main {
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
