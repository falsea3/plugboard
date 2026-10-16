<script lang="ts">
  import type { Workspace } from '../app/workspace.svelte';
  import { app } from '../app/app.svelte';
  import { journal } from '../app/journal.svelte';
  import { formatBytes, formatCount } from '../ui/format';
  import Icon from '../ui/Icon.svelte';
  import Modal from '../ui/Modal.svelte';
  import ToggleRow from '../ui/ToggleRow.svelte';
  import Progress from './Progress.svelte';
  import ImportReport from './ImportReport.svelte';
  import { ImportJob } from './job.svelte';
  import { importShare } from './plan';

  let { ws, onclose }: { ws: Workspace; onclose: () => void } = $props();

  const canTx = $derived(ws.session.engine.transactionalDDL);
  const prod = $derived(ws.connection.env === 'prod');
  let oneTransaction = $state(true);
  let stopOnError = $state(true);
  let sure = $state(false);
  const job = new ImportJob();

  async function start() {
    const res = await job.run(ws.session.sessionId, { oneTransaction: canTx && oneTransaction, stopOnError });
    if (!res) {
      if (job.error && job.error.code !== 'canceled') app.notify(job.error);
      return;
    }
    for (const e of res.errors.slice(0, 50)) journal.add('error', `Import, statement ${e.index}: ${e.message}`, e.statement);
    if (!res.rolledBack && res.statements > 0) ws.loadTables();
  }

  function close() {
    if (job.running) job.stop();
    else onclose();
  }

  const p = $derived(job.progress);
</script>

<Modal title="Import dump" width={480} onclose={close}>
  {#if job.running}
    <Progress share={importShare(p)} label={p?.current || 'Reading the file…'} detail={p ? `${formatBytes(p.bytes)} of ${formatBytes(p.total)} · ${formatCount(p.statements, 'statement')}${p.errors ? ` · ${formatCount(p.errors, 'error')}` : ''}` : ''} />
  {:else if job.result}
    <ImportReport result={job.result} />
  {:else}
    <div class="form">
      <button class="file" onclick={() => job.pick()}>
        <Icon name="upload" size={16} />
        {#if job.file}
          <span class="name">{job.file.name}</span><span class="size faint">{formatBytes(job.file.size)}</span>
        {:else}
          <span class="name">Choose a dump file…</span>
        {/if}
      </button>
      <p class="hint faint">Runs the file statement by statement against <strong>{ws.connection.name}</strong>.</p>
      {#if canTx}
        <ToggleRow bind:checked={oneTransaction}>
          {#snippet title()}All or nothing{/snippet}
          Runs everything in one transaction: the first error rolls the whole import back.
        </ToggleRow>
      {/if}
      {#if !canTx || !oneTransaction}
        <ToggleRow bind:checked={stopOnError}>
          {#snippet title()}Stop at the first error{/snippet}
          Otherwise failed statements are listed and the rest still run.{#if !canTx} What ran before an error stays.{/if}
        </ToggleRow>
      {/if}
      {#if prod}
        <label class="sure"><input type="checkbox" bind:checked={sure} />This is <strong>Production</strong> — import anyway</label>
      {/if}
      {#if ws.readOnly}<p class="warn">Read-only is on for this connection. Turn it off in the sidebar to import.</p>{/if}
      {#if job.error && job.error.code !== 'canceled'}<p class="error">{job.error.message}</p>{/if}
    </div>
  {/if}
  {#snippet footer()}
    <span style="flex:1"></span>
    {#if job.running}
      <button class="btn" onclick={() => job.stop()} disabled={job.stopping}>{job.stopping ? 'Stopping…' : 'Stop'}</button>
    {:else if job.result}
      <button class="btn" onclick={() => (job.result = null)}>Back</button>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn primary" autofocus onclick={onclose}>Done</button>
    {:else}
      <button class="btn" onclick={onclose}>Cancel</button>
      <button class="btn primary" class:danger-fill={prod} onclick={start} disabled={!job.file || ws.readOnly || (prod && !sure)}>Import</button>
    {/if}
  {/snippet}
</Modal>

<style>
  .form { display: flex; flex-direction: column; gap: 10px; }
  .file { display: flex; align-items: center; gap: 10px; height: 44px; padding: 0 14px; border: 1px dashed var(--border); border-radius: var(--radius); background: var(--surface); color: var(--text-2); font-size: 13px; text-align: left; }
  .file:hover { border-color: var(--focus-border); color: var(--text); }
  .name { flex: 1; min-width: 0; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; color: var(--text); }
  .size { font-size: 12px; }
  .hint { margin: 0; font-size: 11.5px; line-height: 1.45; }
  .hint strong { color: var(--text-2); font-weight: 550; }
  .sure { display: flex; align-items: center; gap: 8px; font-size: 12.5px; color: var(--text-2); }
  .sure strong { color: var(--danger); }
  .warn { margin: 0; color: var(--warn); font-size: 12px; }
  .error { margin: 0; color: var(--danger); font-size: 12px; }
</style>
