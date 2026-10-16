<script lang="ts">
  import { untrack } from 'svelte';
  import type { Workspace } from '../app/workspace.svelte';
  import { app } from '../app/app.svelte';
  import { formatBytes, formatCount } from '../ui/format';
  import Modal from '../ui/Modal.svelte';
  import SegmentedControl from '../ui/SegmentedControl.svelte';
  import ToggleRow from '../ui/ToggleRow.svelte';
  import Progress from './Progress.svelte';
  import { ExportJob } from './job.svelte';
  import { dumpDetail, dumpName, dumpOptions, dumpShare, dumpStep, type DumpWhat } from './plan';

  let { ws, onclose }: { ws: Workspace; onclose: () => void } = $props();

  const all = $derived(ws.session.schemas);
  let what = $state<DumpWhat>('all');
  let picked = $state<string[]>(untrack(() => [...ws.session.schemas]));
  let clean = $state(false);
  let inserts = $state(false);
  let owners = $state(false);
  const formats = $derived(ws.session.engine.dumpFormats ?? []);
  let format = $state(untrack(() => ws.session.engine.dumpFormats?.[0]?.id ?? 'sql'));
  const chosen = $derived(formats.find(f => f.id === format));
  const job = new ExportJob();

  function toggle(s: string) {
    picked = picked.includes(s) ? picked.filter(x => x !== s) : all.filter(x => x === s || picked.includes(x));
  }

  async function start() {
    const res = await job.run(ws.session.sessionId, dumpName(ws.connection.database, ws.connection.name, chosen?.ext ?? '.sql'), dumpOptions(what, picked, all, clean, inserts, owners, format));
    if (res?.path) app.notify(`Dump saved to ${res.path}`, 'info');
    else if (job.error && job.error.code !== 'canceled') app.notify(job.error);
  }

  function close() {
    if (job.running) job.stop();
    else onclose();
  }

  const p = $derived(job.progress);
</script>

<Modal title="Export dump" width={460} onclose={close}>
  {#if job.running}
    <Progress share={dumpShare(p)} label={dumpStep(p)} detail={dumpDetail(p)} />
  {:else if job.result}
    <div class="done">
      <p>Saved <strong>{formatBytes(job.result.bytes)}</strong>{job.result.rows > 0 ? ` with ${formatCount(job.result.rows, 'row')} from ${formatCount(job.result.tables, 'table')}` : ''}.</p>
      <p class="path mono">{job.result.path}</p>
    </div>
  {:else}
    <div class="form">
      {#if formats.length > 1}
        <SegmentedControl label="Format" bind:value={format} options={formats.map(f => ({ value: f.id, label: f.label }))} />
        {#if chosen?.hint}<p class="hint faint">{chosen.hint}</p>{/if}
      {/if}
      <SegmentedControl label="What to export" bind:value={what} options={[
        { value: 'all', label: 'Structure and data' },
        { value: 'structure', label: 'Structure only' },
        { value: 'data', label: 'Data only' },
      ]} />
      {#if all.length > 1}
        <span class="label">Schemas</span>
        <div class="schemas" role="group" aria-label="Schemas">
          {#each all as s (s)}
            <label class="schema"><input type="checkbox" checked={picked.includes(s)} onchange={() => toggle(s)} /><span class="mono">{s}</span></label>
          {/each}
        </div>
      {/if}
      {#if what !== 'data'}
        <ToggleRow bind:checked={clean}>
          {#snippet title()}Replace what’s there{/snippet}
          Drops each table, view, function and type before creating it, so the file can load into a database that already has them.
        </ToggleRow>
      {/if}
      {#if what !== 'data'}
        <ToggleRow bind:checked={owners}>
          {#snippet title()}Owners and privileges{/snippet}
          <code>OWNER TO</code>, <code>GRANT</code> and default privileges. The roles have to exist where the file is loaded.
        </ToggleRow>
      {/if}
      {#if what !== 'structure'}
        <ToggleRow bind:checked={inserts}>
          {#snippet title()}Rows as INSERT statements{/snippet}
          Slower to load than <code>COPY</code>, but any SQL tool can run the file.
        </ToggleRow>
      {/if}
      {#if what === 'data'}
        <p class="hint faint">Rows only, for tables that already exist. Foreign keys and triggers are switched off while the rows load and back on after.</p>
      {/if}
      {#if job.error && job.error.code !== 'canceled'}<p class="error">{job.error.message}</p>{/if}
    </div>
  {/if}
  {#snippet footer()}
    <span style="flex:1"></span>
    {#if job.running}
      <button class="btn" onclick={() => job.stop()} disabled={job.stopping}>{job.stopping ? 'Stopping…' : 'Stop'}</button>
    {:else if job.result}
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn primary" autofocus onclick={onclose}>Done</button>
    {:else}
      <button class="btn" onclick={onclose}>Cancel</button>
      <button class="btn primary" onclick={start} disabled={picked.length === 0}>Export…</button>
    {/if}
  {/snippet}
</Modal>

<style>
  .form { display: flex; flex-direction: column; gap: 10px; }
  .label { margin-top: 2px; color: var(--text-2); font-size: 12px; }
  .schemas { max-height: 160px; overflow-y: auto; margin-top: -4px; padding: 4px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface); }
  .schema { display: flex; align-items: center; gap: 8px; height: 26px; padding: 0 6px; border-radius: 5px; font-size: 12.5px; }
  .schema:hover { background: var(--hover); }
  .hint { margin: 0; font-size: 11.5px; line-height: 1.45; }
  .error { margin: 0; color: var(--danger); font-size: 12px; }
  .done p { margin: 0; color: var(--text-2); line-height: 1.5; }
  .done strong { color: var(--text); }
  .path { margin-top: 6px !important; font-size: 11.5px; word-break: break-all; user-select: text; -webkit-user-select: text; }
</style>
