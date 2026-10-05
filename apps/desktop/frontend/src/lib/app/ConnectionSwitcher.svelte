<script lang="ts">
  import { dismiss } from '../ui/dismiss';
  import { app } from './app.svelte';
  import { engine } from '../engines';
  import type { Connection } from '../api/wire';
  import { connectionTarget } from '../ui/format';
  import ConnAvatar from '../connections/ConnAvatar.svelte';
  import EnvBadge from '../connections/EnvBadge.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let query = $state('');
  let index = $state(0);
  let list = $state<HTMLUListElement>();

  const items = $derived.by(() => {
    const q = query.trim().toLowerCase();
    const match = (c: Connection) =>
      !q || [c.name, c.host, c.database, c.file, c.env, engine(c.driver).name].some(v => v?.toLowerCase().includes(q));
    const open = app.connections.filter(c => app.workspaceFor(c.id) && match(c));
    const rest = app.connections.filter(c => !app.workspaceFor(c.id) && match(c));
    return [...open, ...rest];
  });

  $effect(() => {
    void query;
    index = 0;
  });

  function close() {
    app.switcherOpen = false;
  }

  function choose(c: Connection) {
    close();
    app.open(c);
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
    } else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      if (items.length === 0) return;
      index = (index + (e.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length;
      list?.children[index]?.scrollIntoView({ block: 'nearest' });
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (items[index]) choose(items[index]);
    } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'n') {
      e.preventDefault();
      close();
      app.editing = null;
    } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'o') {
      e.preventDefault();
      close();
      app.openSQLiteFile();
    }
  }
</script>

<div class="backdrop" use:dismiss={close}>
  <div class="palette" role="dialog" aria-modal="true" aria-label="Switch connection">
    <div class="search">
      <Icon name="search" size={15} />
      <!-- svelte-ignore a11y_autofocus -->
      <input bind:value={query} {onkeydown} placeholder="Switch to connection…" spellcheck="false" autofocus />
    </div>

    <ul bind:this={list} role="listbox" aria-label="Connections">
      {#each items as c, i (c.id)}
        {@const isOpen = !!app.workspaceFor(c.id)}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <li
          role="option"
          aria-selected={i === index}
          class:selected={i === index}
          onclick={() => choose(c)}
          onmousemove={() => (index = i)}
        >
          <ConnAvatar connection={c} size={28} />
          <div class="meta">
            <div class="name">
              {c.name} <EnvBadge env={c.env} />
              {#if c.readOnly}<span class="marker ro-mark" title="Read-only"><Icon name="lock" size={11} /></span>{/if}
              {#if c.ssh?.enabled && !engine(c.driver).file}<span class="marker" title="Through SSH {c.ssh.user}@{c.ssh.host}"><Icon name="tunnel" size={12} /></span>{/if}
            </div>
            <div class="target">{connectionTarget(c)}</div>
          </div>
          {#if app.connectingId === c.id}
            <span class="state" role="status"><Spinner size={10} />Connecting…</span>
          {:else if isOpen}
            <span class="state open"><span class="dot"></span>{app.active?.connection.id === c.id ? 'Current' : 'Open'}</span>
          {/if}
        </li>
      {:else}
        <li class="empty">
          {app.connections.length === 0 ? 'No saved connections yet.' : `Nothing matches “${query}”.`}
        </li>
      {/each}
    </ul>

    <div class="actions">
      <button class="action" onclick={() => { close(); app.editing = null; }}>
        <Icon name="plus" size={15} /><span>New connection…</span><span class="kbd">⌘N</span>
      </button>
      <button class="action" onclick={() => { close(); app.openSQLiteFile(); }}>
        <Icon name="folder" size={15} /><span>Open SQLite file…</span><span class="kbd">⌘O</span>
      </button>
    </div>

    <footer>
      <span><span class="kbd">↑↓</span> choose</span>
      <span><span class="kbd">↵</span> open</span>
      <span style="flex:1"></span>
      <button class="link" onclick={() => { close(); app.goHome(); }}>All connections</button>
    </footer>
  </div>
</div>

<style>
  .marker { display: flex; color: var(--text-3); }
  .ro-mark { color: var(--text-3); }
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 60;
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding-top: 14vh;
    background: rgba(0, 0, 0, 0.32);
    animation: fade 0.1s ease-out;
  }
  .palette {
    width: 560px;
    max-width: calc(100vw - 32px);
    border-radius: 12px;
    background: var(--bg);
    box-shadow: var(--shadow-modal);
    overflow: hidden;
    animation: pop 0.12s ease-out;
  }
  .search {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 16px;
    border-bottom: 1px solid var(--border-subtle);
    color: var(--text-3);
  }
  .search input {
    flex: 1;
    height: 48px;
    border: 0;
    outline: none;
    background: transparent;
    font-size: 15px;
    color: var(--text);
  }
  .search input::placeholder { color: var(--text-3); }
  ul { list-style: none; margin: 0; padding: 6px; max-height: 360px; overflow-y: auto; }
  li[role='option'] {
    display: flex;
    align-items: center;
    gap: 11px;
    height: 46px;
    padding: 0 10px;
    border-radius: 7px;
  }
  li.selected { background: var(--accent-dim); }
  .meta { flex: 1; min-width: 0; }
  .name { display: flex; align-items: center; gap: 8px; font-weight: 550; }
  .target {
    font-family: var(--font-mono);
    font-size: 11.5px;
    color: var(--text-3);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .state { font-size: 11.5px; color: var(--text-3); display: flex; align-items: center; gap: 6px; }
  .state.open { color: var(--ok); }
  .dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
  .empty { padding: 24px; text-align: center; color: var(--text-3); }
  .actions {
    display: flex;
    flex-direction: column;
    padding: 4px 6px 6px;
    border-top: 1px solid var(--border-subtle);
  }
  .action {
    display: flex;
    align-items: center;
    gap: 11px;
    height: 34px;
    padding: 0 10px 0 14px;
    border: 0;
    border-radius: 7px;
    background: transparent;
    color: var(--text-2);
    font-size: 13px;
    text-align: left;
  }
  .action span:nth-child(2) { flex: 1; }
  .action:hover { background: var(--hover); color: var(--text); }
  .action :global(.icon) { color: var(--accent); }
  footer {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 8px 14px;
    border-top: 1px solid var(--border-subtle);
    background: var(--surface);
    font-size: 11.5px;
    color: var(--text-3);
  }
  .link { border: 0; background: transparent; color: var(--accent); font-size: 12px; padding: 0; }
  .link:hover { text-decoration: underline; }
  @keyframes fade { from { opacity: 0; } }
  @keyframes pop { from { opacity: 0; transform: translateY(-6px) scale(0.985); } }
</style>
