<script lang="ts">
  import { app } from '../app/app.svelte';
  import { api, type Connection, type QueryTabState, type TestResult } from '../api/backend';
  import type { Tab } from '../app/workspace.svelte';
  import { engine } from '../engines';
  import { isDirty } from '../scripts/names';
  import { formatBytes } from '../ui/format';
  import { dismiss } from '../ui/dismiss';
  import { connectionUrl, sshRoute } from './home';
  import ConnAvatar from './ConnAvatar.svelte';
  import EnvBadge from './EnvBadge.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let { connection: c, ondelete }: { connection: Connection; ondelete: () => void } = $props();

  type Item = { id: string; title: string; table: boolean; note: string };

  let checks = $state<Record<string, TestResult | 'running'>>({});
  let sizes = $state<Record<string, number>>({});
  let saved = $state<{ id: string; tabs: QueryTabState[] }>({ id: '', tabs: [] });
  let menu = $state(false);

  const ws = $derived(app.workspaceFor(c.id));
  const url = $derived(connectionUrl(c));
  const ssh = $derived(sshRoute(c));
  const check = $derived(checks[c.id]);
  const size = $derived(sizes[c.id]);
  const server = $derived(ws?.session.serverVersion || (typeof check === 'object' && check.ok ? check.serverVersion : '') || engine(c.driver).name);
  const recent = $derived<Item[]>(ws ? ws.tabs.map(fromTab) : saved.id === c.id ? saved.tabs.map(t => ({ id: '', title: t.title, table: false, note: t.script ? 'script' : 'query' })) : []);

  function fromTab(t: Tab): Item {
    if (t.kind === 'table') return { id: t.id, title: t.table, table: true, note: t.dirty ? 'unsaved edits' : t.tableKind };
    if (t.kind === 'query') return { id: t.id, title: t.title, table: false, note: isDirty(t) ? 'edited' : t.script ? 'script' : 'query' };
    return { id: t.id, title: t.title, table: false, note: t.kind };
  }

  $effect(() => {
    const id = c.id;
    if (app.workspaceFor(id)) return;
    api.queryTabs(id).then(tabs => (saved = { id, tabs })).catch(() => (saved = { id, tabs: [] }));
  });

  $effect(() => {
    const id = c.id;
    const sessionId = ws?.session.sessionId;
    if (!sessionId || sizes[id] !== undefined) return;
    api.databaseSize(sessionId).then(n => (sizes[id] = n)).catch(() => (sizes[id] = -1));
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

  async function toggleReadOnly() {
    try {
      await app.saveConnection({ ...c, readOnly: !c.readOnly });
    } catch (err) {
      app.notify(err);
    }
  }
</script>

<aside class="passport" aria-label={c.name}>
  <div class="top">
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
          {:else if check?.ok}<span class="ok">●</span>{Math.round(check.latencyMs)} ms
          {:else if check}<span class="bad" title={check.error}>● failed</span>
          {:else}<button class="link" onclick={runCheck}>Check</button>{/if}
        </dd>
      </div>
      <div><dt>Size</dt><dd>{size === undefined || size < 0 ? '—' : formatBytes(size)}</dd></div>
      <div><dt>Tables</dt><dd>{ws ? ws.tables.length : '—'}</dd></div>
    </dl>
    {#if check && check !== 'running' && !check.ok}<p class="error">{check.error}</p>{/if}
  </div>

  {#if ws && ws.session.schemas.length > 1}
    <div class="section">
      <div class="heading">Schemas</div>
      {#each ws.session.schemas.slice(0, 5) as s (s)}
        <div class="schema" class:current={s === ws.schema}><span class="name">{s}</span>{#if s === ws.schema}<span class="dim">{ws.tables.length} tables</span><span class="bar"></span>{/if}</div>
      {/each}
    </div>
  {/if}

  <div class="section grow">
    <div class="heading">Pick up where you left</div>
    {#each recent.slice(0, 6) as r, i (r.id || i)}
      <button class="recent" onclick={() => resume(r.id)}>
        <span class:tbl={r.table} class="kind"><Icon name={r.table ? 'table' : 'code'} size={14} /></span>
        <span class="name">{r.title}</span>
        <span class="note">{r.note}</span>
      </button>
    {:else}
      <p class="none">Nothing open yet</p>
    {/each}
  </div>

  <div class="actions">
    <button class="btn primary connect" onclick={() => app.open(c)} disabled={app.connectingId === c.id}>
      {#if app.connectingId === c.id}<Spinner size={12} />Connecting…{:else}{ws ? 'Switch to' : 'Connect'}<span class="kbd">↵</span>{/if}
    </button>
    <button class="btn ro" class:on={c.readOnly} aria-pressed={c.readOnly} onclick={toggleReadOnly} title="Open this connection read-only">Read-only</button>
    <span class="more-wrap">
      <button class="btn icon more" onclick={() => (menu = !menu)} aria-label="More" aria-haspopup="menu" aria-expanded={menu}>⋯</button>
      {#if menu}
        <div class="menu-backdrop" use:dismiss={() => (menu = false)}></div>
        <div class="menu" role="menu">
          <button role="menuitem" onclick={() => ((menu = false), (app.editing = c))}><Icon name="pencil" size={13} />Edit…</button>
          {#if ws}<button role="menuitem" onclick={() => ((menu = false), ws && app.requestClose(ws))}><Icon name="unplug" size={13} />Disconnect</button>{/if}
          <button role="menuitem" class="danger" onclick={() => ((menu = false), ondelete())}><Icon name="trash" size={13} />Delete…</button>
        </div>
      {/if}
    </span>
  </div>
</aside>

<style>
  .passport { width: 372px; flex: none; display: flex; flex-direction: column; overflow-y: auto; border: 1px solid var(--hover); border-radius: var(--radius-lg); background: var(--elevated); }
  .top { padding: 18px 18px 0; }
  .head { display: flex; align-items: center; gap: 12px; }
  .head { --avatar-ring: var(--elevated); }
  .who { flex: 1; min-width: 0; }
  .title { display: flex; align-items: center; gap: 6px; min-width: 0; font-size: 16px; font-weight: 600; }
  .text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sub { font-size: 12px; color: var(--text-3); }
  .url { margin-top: 14px; padding: 10px 12px; border: 1px solid var(--hover); border-radius: var(--radius); background: var(--well); font-family: var(--font-mono); font-size: 12px; line-height: 1.7; color: var(--text-soft); overflow-wrap: anywhere; user-select: text; -webkit-user-select: text; }
  .dim { color: var(--text-4); }
  .db { color: var(--text); }
  .stats { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; margin: 14px 0 0; }
  .stats div { padding: 10px; border-radius: var(--radius); background: var(--well); }
  dt { font-size: 11px; color: var(--text-3); }
  dd { display: flex; align-items: center; gap: 4px; height: 18px; margin: 4px 0 0; font-family: var(--font-mono); }
  .ok { color: var(--ok); }
  .bad { color: var(--danger); font-size: 12px; }
  .error { margin: 8px 0 0; font-size: 12px; color: var(--danger); overflow-wrap: anywhere; }
  .link { padding: 0; border: 0; background: transparent; color: var(--accent-text); font: inherit; font-size: 12px; }
  .link:hover { text-decoration: underline; }
  .section { padding: 18px 18px 0; }
  .section.grow { flex: 1; padding-top: 16px; }
  .heading { font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; color: var(--text-3); }
  .schema { display: flex; align-items: center; gap: 8px; height: 28px; font-family: var(--font-mono); font-size: 12px; color: var(--text-2); }
  .schema.current { color: var(--text); }
  .schema .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .bar { width: 70px; height: 6px; border-radius: 3px; background: var(--accent); }
  .recent { width: calc(100% + 16px); display: flex; align-items: center; gap: 8px; height: 32px; margin: 0 -8px; padding: 0 8px; border: 0; border-radius: 8px; background: transparent; color: var(--text); text-align: left; }
  .recent:hover { background: var(--hover); }
  .kind { display: flex; color: var(--syn-str); }
  .kind.tbl { color: var(--accent-text); }
  .recent .name { flex: 1; min-width: 0; overflow: hidden; font-family: var(--font-mono); font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
  .note { flex: none; color: var(--text-4); font-size: 11.5px; }
  .none { margin: 8px 0 0; font-size: 12px; color: var(--text-4); }
  .actions { display: flex; gap: 8px; padding: 14px 18px 18px; }
  .actions .btn { height: 38px; border-radius: 11px; }
  .connect { flex: 1; gap: 8px; }
  .connect .kbd { padding: 0; background: transparent; color: inherit; opacity: 0.8; }
  .ro { padding: 0 12px; }
  .ro.on { background: color-mix(in srgb, var(--warn) 16%, var(--elevated)); color: var(--warn); }
  .more-wrap { position: relative; }
  .more { width: 38px; color: var(--text-2); font-size: 16px; }
  .menu-backdrop { position: fixed; inset: 0; z-index: 20; }
  .menu { position: absolute; right: 0; bottom: 44px; z-index: 21; min-width: 150px; padding: 4px; border-radius: var(--radius); background: var(--elevated); box-shadow: var(--shadow-pop); }
  .menu button { width: 100%; display: flex; align-items: center; gap: 8px; height: 30px; padding: 0 10px; border: 0; border-radius: 7px; background: transparent; color: var(--text); text-align: left; }
  .menu button:hover { background: var(--hover); }
  .menu .danger { color: var(--danger); }
</style>
