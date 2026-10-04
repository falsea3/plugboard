<script lang="ts">
  import { app } from '../app/app.svelte';
  import { api, type Connection } from '../api/backend';
  import { engine } from '../engines';
  import { connectionTarget } from '../ui/format';
  import ConnAvatar from './ConnAvatar.svelte';
  import EnvBadge from './EnvBadge.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import Modal from '../ui/Modal.svelte';
  import mark from '../assets/plugboard-mark.png';

  let query = $state('');
  let selectedId = $state('');
  let deleting = $state<Connection | null>(null);
  let version = $state('');

  api.appInfo().then(i => (version = i.version)).catch(() => {});

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return app.connections;
    return app.connections.filter(c =>
      [c.name, c.host, c.database, c.file, c.env, engine(c.driver).name].some(v => v?.toLowerCase().includes(q)),
    );
  });

  function onListKey(e: KeyboardEvent) {
    const i = filtered.findIndex(c => c.id === selectedId);
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      const next = filtered[Math.max(0, Math.min(filtered.length - 1, i + (e.key === 'ArrowDown' ? 1 : -1)))];
      if (next) selectedId = next.id;
    } else if (e.key === 'Enter' && i >= 0) {
      app.open(filtered[i]);
    }
  }
</script>

<div class="home">
  <div class="panel">
    <aside class="brand">
      <img src={mark} alt="" width="72" height="72" draggable="false" />
      <h1>Plugboard</h1>
      <p class="faint">{version ? `Version ${version}` : ' '}</p>

      <div class="actions">
        <button class="btn primary" onclick={() => (app.editing = null)}><Icon name="plus" />New connection<span class="kbd on-accent">⌘N</span></button>
        <button class="btn" onclick={() => app.openSQLiteFile()}><Icon name="folder" />Open SQLite file…</button>
      </div>
      <p class="hint faint">PostgreSQL · MySQL · SQLite</p>
    </aside>

    <section class="list-pane">
      <div class="search">
        <Icon name="search" />
        <input class="input" placeholder="Search connections" bind:value={query} onkeydown={onListKey} spellcheck="false" />
      </div>

      <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
      <ul class="list" role="listbox" aria-label="Connections" tabindex="0" onkeydown={onListKey}>
        {#each filtered as c (c.id)}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <li
            role="option"
            aria-selected={selectedId === c.id}
            class:selected={selectedId === c.id}
            class="env-{c.env || 'none'}"
            onclick={() => (selectedId = c.id)}
            ondblclick={() => app.open(c)}
          >
            <ConnAvatar connection={c} size={34} />
            <div class="meta">
              <div class="name">
              {c.name} <EnvBadge env={c.env} />
              {#if c.readOnly}<span class="marker ro-mark" title="Read-only"><Icon name="lock" size={11} /></span>{/if}
              {#if c.ssh?.enabled && !engine(c.driver).file}<span class="marker" title="Through SSH {c.ssh.user}@{c.ssh.host}"><Icon name="tunnel" size={12} /></span>{/if}
            </div>
              <div class="target faint">{connectionTarget(c)}</div>
            </div>
            {#if app.connectingId === c.id}
              <span class="faint connecting" role="status"><Spinner size={11} />Connecting…</span>
            {:else}
              {@const isOpen = !!app.workspaceFor(c.id)}
              {#if isOpen}<span class="open-badge"><span class="dot"></span>Open</span>{/if}
              <div class="row-actions">
                <button class="btn icon sm ghost" title="Edit" onclick={e => { e.stopPropagation(); app.editing = c; }}><Icon name="pencil" size={13} /></button>
                <button class="btn icon sm ghost" title="Delete" onclick={e => { e.stopPropagation(); deleting = c; }}><Icon name="trash" size={13} /></button>
                <button class="btn sm" onclick={e => { e.stopPropagation(); app.open(c); }}>{isOpen ? 'Switch to' : 'Connect'}</button>
              </div>
            {/if}
          </li>
        {:else}
          <li class="empty">
            {#if !app.connectionsLoaded}
              <span class="faint loading"><Spinner size={11} />Loading…</span>
            {:else if query}
              <span class="faint">No connections match “{query}”.</span>
            {:else}
              <Icon name="database" size={28} />
              <span>No connections yet</span>
              <button class="btn sm" onclick={() => (app.editing = null)}>Create the first one</button>
            {/if}
          </li>
        {/each}
      </ul>
    </section>
  </div>
</div>

{#if deleting}
  {@const target = deleting}
  <Modal title="Delete connection?" width={400} onclose={() => (deleting = null)}>
    <p class="muted" style="margin:0">“{target.name}” and its saved password will be removed. The database itself is not touched.</p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <button class="btn" onclick={() => (deleting = null)}>Cancel</button>
      <button class="btn primary" onclick={() => { app.deleteConnection(target.id); deleting = null; }}>Delete</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .marker { display: flex; color: var(--text-3); }
  .ro-mark { color: var(--text-3); }
  .home {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--surface);
  }
  .panel {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: 300px 1fr;
    overflow: hidden;
    background: var(--bg);
  }
  .brand {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 56px 28px 24px;
    background: var(--surface);
  }
  .brand img { border-radius: 16px; box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18); }
  h1 { margin: 16px 0 2px; font-size: 20px; font-weight: 650; letter-spacing: -0.01em; }
  .brand p { margin: 0; font-size: 12px; }
  .actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;
    margin-top: 32px;
  }
  .actions .btn { height: 32px; justify-content: flex-start; padding: 0 12px; }
  .hint { margin-top: auto !important; }

  .list-pane { display: flex; flex-direction: column; min-width: 0; }
  .search {
    position: relative;
    padding: 14px 14px 10px;
    color: var(--text-3);
  }
  .search :global(.icon) { position: absolute; left: 24px; top: 21px; }
  .search .input { padding-left: 30px; }

  .list {
    flex: 1;
    overflow: auto;
    margin: 0;
    padding: 0 8px 12px;
    list-style: none;
    outline: none;
  }
  li[role='option'] {
    position: relative;
    display: flex;
    align-items: center;
    gap: 12px;
    height: 52px;
    padding: 0 10px 0 14px;
    border-radius: var(--radius);
  }
  li[role='option']::before {
    content: '';
    position: absolute;
    left: 4px;
    top: 12px;
    bottom: 12px;
    width: 3px;
    border-radius: 3px;
    background: var(--env-color, transparent);
  }
  .env-local { --env-color: var(--env-local); }
  .env-dev { --env-color: var(--env-dev); }
  .env-staging { --env-color: var(--env-staging); }
  .env-prod { --env-color: var(--env-prod); }
  li[role='option']:hover { background: var(--hover); }
  li.selected { background: var(--accent-dim); }
  .meta { flex: 1; min-width: 0; }
  .name { display: flex; align-items: center; gap: 8px; font-weight: 550; }
  .target {
    margin-top: 1px;
    font-size: 12px;
    font-family: var(--font-mono);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row-actions { display: flex; gap: 2px; align-items: center; opacity: 0; transition: opacity 0.1s; }
  li:hover .row-actions, li.selected .row-actions { opacity: 1; }
  .row-actions .btn:last-child { margin-left: 6px; }
  .connecting, .loading { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; }
  .open-badge { display: flex; align-items: center; gap: 6px; font-size: 11.5px; color: var(--ok); margin-right: 4px; }
  .open-badge .dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
  .kbd.on-accent { margin-left: auto; border-color: rgba(255, 255, 255, 0.35); color: rgba(255, 255, 255, 0.8); }

  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 72px 0;
    color: var(--text-2);
  }
  .empty :global(.icon) { color: var(--text-3); }
</style>
