<script lang="ts">
  import { api } from '../api/backend';
  import { app } from '../app/app.svelte';
  import Icon from '../ui/Icon.svelte';
  import { copyToClipboard } from '../ui/clipboard';

  let command = $state('');
  let log = $state('');
  api.mcpCommand().then(c => (command = c), () => {});
  api.mcpLog().then(l => (log = l), () => {});

  const open = $derived(app.connections.filter(c => c.aiAccess));
</script>

<h2>AI agents</h2>

<p class="hint lead">
  Plugboard can be an MCP server, so Claude Code, Cursor and other AI agents can look at your databases without seeing a password: they ask Plugboard, and Plugboard
  connects with the connections you already have — Keychain, SSH tunnels and read-only rules included.
</p>

<div class="field">
  <div class="label">Add it to Claude Code</div>
  <div class="command">
    <code class="mono">{command || '…'}</code>
    <button class="btn icon sm ghost" onclick={() => copyToClipboard(command)} disabled={!command} title="Copy command" aria-label="Copy command"><Icon name="copy" size={13} /></button>
  </div>
  <p class="hint">
    Other agents take the same command: run <span class="mono">plugboard mcp</span> over stdio. Add <span class="mono">--env staging,dev</span> to show only connections with those tags,
    and <span class="mono">--write</span> to let agents change data on connections that aren’t read-only. Production is always read-only for agents.
  </p>
</div>

<div class="field">
  <div class="label">Connections open to agents</div>
  {#if open.length === 0}
    <p class="hint">None yet. Turn on “AI agents (MCP)” in a connection’s settings — agents see nothing until you do.</p>
  {:else}
    <ul class="open">
      {#each open as c (c.id)}
        <li>{c.name}<span class="faint">{c.env ? ` · ${c.env}` : ''}{c.readOnly || c.env === 'prod' ? ' · read-only' : ''}</span></li>
      {/each}
    </ul>
  {/if}
</div>

<div class="field">
  <div class="label">What agents did</div>
  <p class="hint">Every call is written to <span class="mono">{log}</span>. Rows an agent reads are sent to its model provider.</p>
  <button class="btn sm" onclick={() => api.openLogs().catch(err => app.notify(err))}><Icon name="folder" size={13} />Open logs folder</button>
</div>

<style>
  .lead { margin: 0 0 16px; line-height: 1.55; }
  .command { display: flex; align-items: center; gap: 6px; padding: 6px 6px 6px 10px; border: 1px solid var(--border); border-radius: 7px; background: var(--bg); }
  .command code { flex: 1; min-width: 0; overflow-x: auto; white-space: nowrap; font-size: 12px; user-select: text; -webkit-user-select: text; }
  .open { margin: 4px 0 0; padding-left: 18px; color: var(--text-2); font-size: 12.5px; line-height: 1.7; }
</style>
