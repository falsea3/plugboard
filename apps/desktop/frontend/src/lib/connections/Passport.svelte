<script lang="ts">
  import { app } from '../app/app.svelte';
  import { api, type Connection, type TestResult } from '../api/backend';
  import { engine } from '../engines';
  import { connectionUrl, sshRoute } from './home';
  import ConnAvatar from './ConnAvatar.svelte';
  import EnvBadge from './EnvBadge.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let { connection: c, ondelete }: { connection: Connection; ondelete: () => void } = $props();

  let checks = $state<Record<string, TestResult | 'running'>>({});
  let saved = $state<{ id: string; titles: string[] }>({ id: '', titles: [] });

  const ws = $derived(app.workspaceFor(c.id));
  const url = $derived(connectionUrl(c));
  const ssh = $derived(sshRoute(c));
  const check = $derived(checks[c.id]);
  const server = $derived(ws?.session.serverVersion || (typeof check === 'object' && check.ok ? check.serverVersion : '') || engine(c.driver).name);
  const recent = $derived(
    ws ? ws.tabs.map(t => ({ id: t.id, title: t.kind === 'table' ? t.table : t.title, kind: t.kind })) : saved.id === c.id ? saved.titles.map(title => ({ id: '', title, kind: 'query' as const })) : [],
  );

  $effect(() => {
    const id = c.id;
    if (app.workspaceFor(id)) return;
    api.queryTabs(id).then(tabs => (saved = { id, titles: tabs.map(t => t.title) })).catch(() => (saved = { id, titles: [] }));
  });

  async function runCheck() {
    const id = c.id;
    checks[id] = 'running';
    checks[id] = await api.testConnection(c).catch(err => ({ ok: false, error: String(err?.message ?? err) }) as TestResult);
  }

  function resume(tabId: string) {
    if (ws && tabId) ws.activeTabId = tabId;
    app.open(c);
  }
</script>

<aside class="passport" aria-label={c.name}>
  <div class="head">
    <ConnAvatar connection={c} size={44} />
    <div class="who">
      <div class="title"><span class="text">{c.name}</span><EnvBadge env={c.env} /></div>
      <div class="sub">{server} · {c.group || 'Ungrouped'}</div>
    </div>
  </div>

  <div class="url">
    <div><span class="dim">{url.scheme}</span>{url.address}<span class="db">{url.database}</span></div>
    {#if ssh}<div><span class="dim">{'via ssh '}</span>{ssh}</div>{/if}
  </div>

  <dl class="stats">
    <div>
      <dt>Latency</dt>
      <dd>
        {#if check === 'running'}<Spinner size={11} label="Checking" />
        {:else if check?.ok}<span class="ok">●</span> {Math.round(check.latencyMs)} ms
        {:else if check}<span class="bad" title={check.error}>● failed</span>
        {:else}<button class="link" onclick={runCheck}>Check</button>{/if}
      </dd>
    </div>
    <div><dt>Schemas</dt><dd>{ws ? ws.session.schemas.length : '—'}</dd></div>
    <div><dt>Tables</dt><dd>{ws ? ws.tables.length : '—'}</dd></div>
  </dl>
  {#if check && check !== 'running' && !check.ok}<p class="error">{check.error}</p>{/if}

  {#if ws && ws.session.schemas.length > 1}
    <div class="heading">Schemas</div>
    <ul class="schemas">
      {#each ws.session.schemas.slice(0, 6) as s (s)}
        <li class:current={s === ws.schema}><span>{s}</span>{#if s === ws.schema}<span class="dim">{ws.tables.length} tables</span>{/if}</li>
      {/each}
    </ul>
  {/if}

  <div class="heading">{ws ? 'Open tabs' : 'Pick up where you left'}</div>
  <ul class="recent">
    {#each recent.slice(0, 6) as r, i (r.id || i)}
      <li><button onclick={() => resume(r.id)}><Icon name={r.kind === 'table' ? 'table' : 'code'} size={14} /><span>{r.title}</span></button></li>
    {:else}
      <li class="none">Nothing open yet</li>
    {/each}
  </ul>

  <div class="actions">
    <button class="btn primary connect" onclick={() => app.open(c)} disabled={app.connectingId === c.id}>
      {#if app.connectingId === c.id}<Spinner size={12} />Connecting…{:else}{ws ? 'Switch to' : 'Connect'}<span class="kbd">↵</span>{/if}
    </button>
    <button class="btn" onclick={() => (app.editing = c)}>Edit</button>
    <button class="btn icon" onclick={ondelete} title="Delete connection" aria-label="Delete connection"><Icon name="trash" size={14} /></button>
  </div>
</aside>

<style>
  .passport { width: 340px; flex: none; display: flex; flex-direction: column; padding: 18px 18px 0; overflow-y: auto; border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--elevated); }
  .head { --avatar-ring: var(--elevated); display: flex; align-items: center; gap: 12px; }
  .who { flex: 1; min-width: 0; }
  .title { display: flex; align-items: center; gap: 6px; min-width: 0; font-size: 16px; font-weight: 600; }
  .text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sub { margin-top: 2px; font-size: 12px; color: var(--text-3); }
  .url { margin-top: 14px; padding: 10px 12px; border: 1px solid var(--border-subtle); border-radius: var(--radius); background: var(--card); font-family: var(--font-mono); font-size: 12px; line-height: 1.7; color: var(--text-2); overflow-wrap: anywhere; user-select: text; -webkit-user-select: text; }
  .dim { color: var(--text-4); }
  .db { color: var(--text); }
  .stats { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; margin: 14px 0 0; }
  .stats div { padding: 10px; border-radius: var(--radius); background: var(--hover); }
  dt { font-size: 11px; color: var(--text-3); }
  dd { display: flex; align-items: center; gap: 4px; height: 18px; margin: 4px 0 0; font-family: var(--font-mono); }
  .ok { color: var(--ok); }
  .bad { color: var(--danger); font-family: var(--font-ui, inherit); font-size: 12px; }
  .error { margin: 8px 0 0; font-size: 12px; color: var(--danger); overflow-wrap: anywhere; }
  .link { padding: 0; border: 0; background: transparent; color: var(--accent-text); font: inherit; font-size: 12px; }
  .link:hover { text-decoration: underline; }
  .heading { padding: 18px 0 6px; font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; color: var(--text-3); }
  ul { list-style: none; margin: 0; padding: 0; }
  .schemas li { display: flex; justify-content: space-between; height: 26px; align-items: center; font-family: var(--font-mono); font-size: 12px; color: var(--text-2); }
  .schemas li.current { color: var(--text); }
  .recent button { width: 100%; display: flex; align-items: center; gap: 8px; height: 30px; margin: 0 -8px; padding: 0 8px; border: 0; border-radius: 8px; background: transparent; color: var(--text); font-family: var(--font-mono); font-size: 12px; text-align: left; box-sizing: content-box; }
  .recent button:hover { background: var(--hover); }
  .recent button span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .recent :global(.icon) { color: var(--accent-text); }
  .none { font-size: 12px; color: var(--text-4); }
  .actions { position: sticky; bottom: 0; display: flex; gap: 8px; margin-top: auto; padding: 14px 0 18px; background: var(--elevated); }
  .connect { flex: 1; gap: 8px; }
  .connect .kbd { background: transparent; color: inherit; opacity: 0.75; }
</style>
