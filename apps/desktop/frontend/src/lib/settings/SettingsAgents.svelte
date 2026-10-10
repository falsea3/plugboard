<script lang="ts">
  import { api } from '../api/backend';
  import { app } from '../app/app.svelte';
  import Icon from '../ui/Icon.svelte';
  import ToggleRow from '../ui/ToggleRow.svelte';
  import { copyToClipboard } from '../ui/clipboard';
  import { CLIENTS, agentSetup, mcpArgs, type AgentClient } from './agents';

  let exe = $state('');
  let log = $state('');
  let client = $state<AgentClient>('claude');
  let write = $state(false);
  let create = $state(true);
  api.mcpExecutable().then(e => (exe = e), () => {});
  api.mcpLog().then(l => (log = l), () => {});

  const setup = $derived(exe ? agentSetup(client, exe, mcpArgs({ write, create })) : null);
  const open = $derived(app.connections.filter(c => c.aiAccess));
</script>

<h2>AI agents</h2>

<p class="hint lead">
  Plugboard can be an MCP server, so Claude Code, Codex, Cursor and other AI agents can look at your databases without seeing a password: they ask Plugboard, and
  Plugboard connects with the connections you already have — Keychain, SSH tunnels and read-only rules included.
</p>

<div class="field">
  <div class="label spaced">Add Plugboard to</div>
  <div class="clients" role="tablist" aria-label="Agent">
    {#each CLIENTS as c (c.id)}
      <button role="tab" aria-selected={client === c.id} class:on={client === c.id} onclick={() => (client = c.id)}>{c.label}</button>
    {/each}
  </div>
  <div class="snippet">
    <pre class="mono" aria-label="Setup for {CLIENTS.find(c => c.id === client)?.label}">{#each (setup?.text ?? '…').split(/(\s+)/) as part, i (i)}{#if part.trim()}<span class="word">{part}</span>{:else}{part}{/if}{/each}</pre>
    <button class="btn icon sm ghost" onclick={() => setup && copyToClipboard(setup.text)} disabled={!setup} title="Copy" aria-label="Copy setup"><Icon name="copy" size={13} /></button>
  </div>
  {#if setup}<p class="hint setup-hint">{setup.hint}</p>{/if}
  <div class="toggles">
    <ToggleRow bind:checked={write}>
      {#snippet title()}Let agents change data{/snippet}
      Adds <span class="mono">--write</span>: agents may write to connections that aren’t read-only. Production stays read-only for agents.
    </ToggleRow>
    <ToggleRow bind:checked={create}>
      {#snippet title()}Let agents add connections{/snippet}
      Adds <span class="mono">--create</span>: an agent can save a new connection, open to agents and read-only unless it asks otherwise. Plugboard connects first and saves it only if that works.
    </ToggleRow>
    <p class="hint">These change the setup above — add Plugboard again after switching them. <span class="mono">--env staging,dev</span> limits agents to connections with those tags.</p>
  </div>
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
  .spaced { margin-bottom: 8px; }
  .clients { display: inline-flex; gap: 2px; margin-bottom: 8px; padding: 2px; border-radius: 7px; background: var(--elevated); }
  .clients button { padding: 3px 10px; border: 0; border-radius: 5px; background: transparent; color: var(--text-2); font-size: 12px; font-weight: 500; }
  .clients button:hover { color: var(--text); }
  .clients button.on { background: var(--bg); color: var(--text); box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2); }
  .snippet { display: flex; align-items: flex-start; gap: 6px; padding: 6px 6px 6px 10px; border: 1px solid var(--border); border-radius: 7px; background: var(--bg); }
  .snippet pre { flex: 1; min-width: 0; margin: 2px 0; font-size: 12px; line-height: 1.5; white-space: pre-wrap; overflow-x: auto; user-select: text; -webkit-user-select: text; }
  .snippet .word { white-space: nowrap; }
  .setup-hint { margin-top: 6px; }
  .toggles { display: flex; flex-direction: column; gap: 6px; margin-top: 16px; }
  .open { margin: 4px 0 0; padding-left: 18px; color: var(--text-2); font-size: 12.5px; line-height: 1.7; }
</style>
