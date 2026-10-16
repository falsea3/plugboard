<script lang="ts">
  import { app } from '../app/app.svelte';
  import { api, AppError } from '../api/backend';
  import type { ImportScan } from '../api/wire';
  import { connectionTarget, formatCount } from '../ui/format';
  import Modal from '../ui/Modal.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import DriverMark from './DriverMark.svelte';

  let { onclose }: { onclose: () => void } = $props();

  let scan = $state<ImportScan | null>(null);
  let error = $state('');
  let picked = $state<string[]>([]);
  let busy = $state(false);

  function show(s: ImportScan) {
    if (!s.root && scan) return;
    scan = s;
    error = '';
    picked = s.candidates.filter(c => !c.skip && !c.exists).map(c => c.key);
  }

  function fail(err: unknown) {
    error = err instanceof AppError || err instanceof Error ? err.message : String(err);
  }

  api.findDBeaver().then(show, fail);

  function toggle(key: string) {
    picked = picked.includes(key) ? picked.filter(k => k !== key) : [...picked, key];
  }

  async function run() {
    if (!scan) return;
    busy = true;
    try {
      const done = await api.importDBeaver(scan.root, picked);
      await app.loadConnections();
      for (const f of done.failed) app.notify(f);
      if (done.added) app.notify(`${formatCount(done.added, 'connection')} imported from DBeaver`, 'info');
      onclose();
    } catch (err) {
      fail(err);
    } finally {
      busy = false;
    }
  }
</script>

<Modal title="Import from DBeaver" width={560} {onclose}>
  {#if !scan && !error}
    <p class="faint"><Spinner size={11} /> Looking for DBeaver’s connections…</p>
  {:else if scan && scan.candidates.length > 0}
    <p class="from faint">From <span class="mono">{scan.root}</span>. Passwords DBeaver saved go to the system keychain.</p>
    <div class="list" role="group" aria-label="DBeaver connections">
      {#each scan.candidates as c (c.key)}
        <label class="row" class:off={!!c.skip}>
          <input type="checkbox" checked={picked.includes(c.key)} disabled={!!c.skip} onchange={() => toggle(c.key)} />
          {#if c.skip}<span class="mark-none"></span>{:else}<DriverMark driver={c.connection.driver} size={22} />{/if}
          <span class="main">
            <span class="name">{c.connection.name}{#if c.connection.group}<span class="group faint"> · {c.connection.group}</span>{/if}</span>
            <span class="sub faint">{c.skip || c.note || connectionTarget(c.connection)}</span>
          </span>
          {#if c.exists}<span class="tag">already here</span>{:else if !c.skip && !c.password && !c.connection.file}<span class="tag">no saved password</span>{/if}
        </label>
      {/each}
    </div>
  {:else if scan}
    <p class="faint">No DBeaver connections were found{scan.root ? ` in ${scan.root}` : ' where DBeaver keeps them'}. Choose DBeaver’s workspace folder if it lives somewhere else.</p>
  {/if}
  {#if error}<p class="error">{error}</p>{/if}
  {#snippet footer()}
    <button class="btn ghost" onclick={() => api.chooseDBeaverFolder().then(show, fail)}>Choose folder…</button>
    <span style="flex:1"></span>
    <button class="btn" onclick={onclose}>Cancel</button>
    <button class="btn primary" onclick={run} disabled={busy || picked.length === 0}>{busy ? 'Importing…' : `Import ${picked.length || ''}`.trim()}</button>
  {/snippet}
</Modal>

<style>
  .from { margin: 0 0 10px; font-size: 12px; line-height: 1.45; word-break: break-all; }
  .list { max-height: 360px; overflow-y: auto; padding: 4px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface); }
  .row { display: flex; align-items: center; gap: 10px; padding: 6px 8px; border-radius: 6px; }
  .row:hover { background: var(--hover); }
  .row.off { opacity: 0.55; }
  .mark-none { width: 22px; flex: none; }
  .main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
  .name, .sub { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
  .name { font-size: 13px; color: var(--text); }
  .sub { font-size: 11.5px; }
  .tag { flex: none; padding: 0 6px; border: 1px solid var(--border); border-radius: 5px; color: var(--text-3); font-size: 11px; }
  .error { margin: 10px 0 0; color: var(--danger); font-size: 12px; }
</style>
