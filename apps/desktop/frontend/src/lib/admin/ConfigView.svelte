<script lang="ts">
  import { untrack } from 'svelte';
  import { api, type ConfigFile, type ConfigSaved, type ConfigText } from '../api/backend';
  import { app } from '../app/app.svelte';
  import type { Workspace } from '../app/workspace.svelte';
  import Icon from '../ui/Icon.svelte';
  import LoadBar from '../ui/LoadBar.svelte';
  import Modal from '../ui/Modal.svelte';
  import Select from '../ui/Select.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import ConfEditor from './ConfEditor.svelte';
  import { viaLabel } from './conf';

  let { ws, ondirty }: { ws: Workspace; ondirty: (dirty: boolean) => void } = $props();

  const OTHER = '\u0000other';
  let files = $state<ConfigFile[]>([]);
  let picked = $state('');
  let customPath = $state('');
  let loaded = $state<ConfigText | null>(null);
  let text = $state('');
  let loading = $state(false);
  let saving = $state(false);
  let error = $state('');
  let saved = $state<ConfigSaved | null>(null);
  let generation = $state(0);
  let askDiscard = $state<(() => void) | null>(null);
  let askProd = $state(false);
  let editor = $state<ConfEditor>();
  let pickerKey = $state(0);

  const dirty = $derived(!!loaded && text !== loaded.text);
  const readOnly = $derived(ws.readOnly);
  const options = $derived([...files.map(f => ({ value: f.path, label: f.name, hint: f.path })), { value: OTHER, label: 'Other file…' }]);
  const via = $derived(loaded ? viaLabel(loaded.via) : null);

  $effect(() => ondirty(dirty));

  function message(err: unknown) {
    return err instanceof Error ? err.message : String(err);
  }

  async function start() {
    loading = true;
    try {
      files = await api.configFiles(ws.session.sessionId);
      picked = files[0]?.path ?? OTHER;
    } catch (err) {
      error = `${message(err)} — enter the file's path instead.`;
      picked = OTHER;
    } finally {
      loading = false;
    }
    if (picked !== OTHER) await open(picked);
  }

  function guarded(run: () => void) {
    if (dirty) askDiscard = run;
    else run();
  }

  async function open(path: string) {
    if (!path.trim()) return;
    loading = true;
    error = '';
    saved = null;
    try {
      loaded = await api.readConfig(ws.session.sessionId, path);
      text = loaded.text;
      generation++;
    } catch (err) {
      error = message(err);
    } finally {
      loading = false;
    }
  }

  function pick(value: string) {
    if (value === OTHER) {
      customPath = loaded?.path ?? '';
      picked = OTHER;
      return;
    }
    if (dirty) {
      askDiscard = () => ((picked = value), open(value));
      pickerKey++;
      return;
    }
    picked = value;
    void open(value);
  }

  function save() {
    if (!loaded || !dirty || saving || readOnly) return;
    if (ws.connection.env === 'prod') askProd = true;
    else void write();
  }

  async function write() {
    if (!loaded) return;
    askProd = false;
    saving = true;
    const content = text;
    try {
      saved = await api.writeConfig(ws.session.sessionId, loaded.path, content);
      loaded = { ...loaded, text: content, via: saved.via };
      if (saved.reloaded) app.notify(saved.restart.length ? `Saved and reloaded — ${saved.restart.join(', ')} ${saved.restart.length === 1 ? 'needs' : 'need'} a restart` : 'Saved and reloaded', 'info');
    } catch (err) {
      app.notify(err);
    } finally {
      saving = false;
    }
  }

  untrack(start);
</script>

<div class="config">
  <div class="bar">
    {#key pickerKey}<Select value={picked} {options} onchange={pick} aria-label="Configuration file" class="file-select" />{/key}
    {#if picked === OTHER}
      <form class="custom" onsubmit={e => (e.preventDefault(), guarded(() => open(customPath)))}>
        <input class="input mono" bind:value={customPath} placeholder={files[0]?.path ?? "/path/to/server.conf"} spellcheck="false" aria-label="File path" />
        <button class="btn sm" disabled={!customPath.trim()}>Open</button>
      </form>
    {:else if loaded}
      <span class="path mono faint" title={loaded.path}>{loaded.path}</span>
    {/if}
    <span style="flex:1"></span>
    {#if via}<span class="via" title={via.hint}>{via.label}</span>{/if}
    <button class="btn icon sm ghost" onclick={() => loaded && guarded(() => open(loaded!.path))} disabled={!loaded || loading} title="Reload from the server" aria-label="Reload"><Icon name="refresh" size={13} /></button>
    <button class="btn sm primary" onclick={save} disabled={!dirty || saving || readOnly} title={readOnly ? 'Read-only is on — turn it off to save' : 'Save and reload the server configuration (⌘S)'}>
      {#if saving}<Spinner size={12} />{/if}Save &amp; reload
    </button>
  </div>
  {#if saved && !saved.reloaded}
    <div class="problems" role="alert">
      <strong><Icon name="alert" size={13} />Saved, but not reloaded — the server found {saved.problems.length === 1 ? 'a problem' : 'problems'}. Fix and save again; until then the server keeps its old settings, and a restart would fail.</strong>
      {#each saved.problems as p, i (i)}
        <button class="problem" onclick={() => editor?.goto(p.line)}><span class="mono">line {p.line}</span>{p.message}</button>
      {/each}
    </div>
  {/if}
  <div class="body">
    {#if loading}<LoadBar label="Loading the file" />{/if}
    {#if error}<div class="error" role="alert"><Icon name="alert" />{error}</div>{/if}
    {#if loaded}
      {#key generation}
        <ConfEditor bind:this={editor} value={loaded.text} {readOnly} problems={saved?.problems ?? []} onchange={t => (text = t)} onsave={save} />
      {/key}
    {/if}
  </div>
</div>

{#if askDiscard}
  <Modal title="Discard your changes?" width={400} onclose={() => (askDiscard = null)}>
    <p class="text"><strong>{loaded?.path}</strong> has changes that aren’t saved.</p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (askDiscard = null)}>Keep editing</button>
      <button class="btn primary danger-fill" onclick={() => { const run = askDiscard; askDiscard = null; run?.(); }}>Discard</button>
    {/snippet}
  </Modal>
{/if}

{#if askProd}
  <Modal title="Change Production’s configuration?" width={420} onclose={() => (askProd = false)}>
    <p class="text"><strong>{ws.connection.name}</strong> is tagged Production. Saving writes <span class="mono">{loaded?.path}</span> and reloads the server’s configuration.</p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (askProd = false)}>Cancel</button>
      <button class="btn primary danger-fill" onclick={write}>Save on Production</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .config { height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .bar { flex: none; display: flex; align-items: center; gap: 8px; height: 40px; padding: 0 10px; border-bottom: 1px solid var(--border); background: var(--surface); }
  .bar :global(.file-select) { width: 190px; flex: none; }
  .custom { display: flex; gap: 6px; flex: 1; min-width: 0; max-width: 520px; }
  .custom .input { flex: 1; min-width: 0; height: 26px; font-size: 12px; }
  .path { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; }
  .via { flex: none; padding: 0 6px; border: 1px solid var(--border); border-radius: 4px; color: var(--text-3); font-size: 11px; }
  .body { position: relative; flex: 1; min-height: 0; display: flex; flex-direction: column; }
  .problems { flex: none; display: flex; flex-direction: column; gap: 2px; padding: 8px 12px; border-bottom: 1px solid var(--border); background: color-mix(in srgb, var(--danger) 8%, var(--surface)); font-size: 12px; }
  .problems strong { display: flex; align-items: center; gap: 6px; margin-bottom: 4px; color: var(--danger); font-weight: 500; }
  .problem { display: flex; gap: 10px; padding: 2px 0; border: 0; background: none; color: var(--text-2); font-size: 12px; text-align: left; cursor: pointer; }
  .problem:hover { color: var(--text); }
  .problem .mono { flex: none; color: var(--text-3); }
  .error { display: flex; gap: 8px; margin: 12px; padding: 10px 12px; border-radius: var(--radius); color: var(--danger); background: color-mix(in srgb, var(--danger) 10%, transparent); font-size: 12px; user-select: text; -webkit-user-select: text; }
  .text { margin: 0; color: var(--text-2); line-height: 1.5; }
  .text strong { color: var(--text); }
</style>
