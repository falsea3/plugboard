<script lang="ts">
  import { api, type Connection } from '../api/backend';
  import { app } from '../app/app.svelte';
  import { engine, nameFromFile } from '../engines';
  import Icon from '../ui/Icon.svelte';
  import Select from '../ui/Select.svelte';
  import Autocomplete from '../ui/Autocomplete.svelte';
  import EnvPicker from './EnvPicker.svelte';

  let {
    form = $bindable(),
    passwordHint,
    onenv,
    onerror,
  }: { form: Connection; passwordHint: string; onenv: () => void; onerror: (message: string) => void } = $props();

  const sslModes = [
    { value: '', label: 'SSL preferred', hint: 'encrypt if the server can' },
    { value: 'disable', label: 'SSL off' },
    { value: 'require', label: 'SSL required', hint: 'encrypt, trust any certificate' },
    { value: 'verify-full', label: 'SSL verified' },
  ];
  const e = $derived(engine(form.driver));
  const sslOptions = $derived(e.canPreferSsl ? sslModes : [{ value: '', label: 'SSL off' }, ...sslModes.slice(2)]);
  const groups = $derived(app.groups.sections.map(s => s.name).filter(Boolean));
  async function chooseFile() {
    try {
      const path = await api.chooseSQLiteFile();
      if (path) {
        form.file = path;
        if (!form.name) form.name = nameFromFile(e, path);
      }
    } catch (err) {
      onerror(err instanceof Error ? err.message : String(err));
    }
  }
</script>

<div class="form-grid">
  <label for="cf-name">Name</label>
  <input id="cf-name" class="input" bind:value={form.name} placeholder={e.file ? 'Optional — named after the file' : 'Optional — e.g. Production replica'} autocomplete="off" spellcheck="false" />

  <label for="cf-group">Group</label>
  <Autocomplete id="cf-group" bind:value={form.group} suggestions={groups} mono={false} placeholder="Optional — e.g. Shop" />

  <span class="label">Environment</span>
  <EnvPicker bind:value={form.env} onchange={onenv} />

  {#if e.file}
    <label for="cf-file">File</label>
    <div class="form-row">
      <input id="cf-file" class="input mono" bind:value={form.file} placeholder="/path/to/database.db" spellcheck="false" />
      <button type="button" class="btn" onclick={chooseFile}><Icon name="folder" />Choose…</button>
    </div>
  {:else}
    <label for="cf-host">Host</label>
    <div class="form-row">
      <input id="cf-host" class="input mono" bind:value={form.host} placeholder="127.0.0.1" spellcheck="false" autocomplete="off" />
      <input class="input mono port" type="number" bind:value={form.port} min="1" max="65535" aria-label="Port" placeholder={String(e.defaultPort)} />
    </div>

    <label for="cf-user">User</label>
    <div class="form-row">
      <input id="cf-user" class="input" bind:value={form.user} placeholder={e.userHint} spellcheck="false" autocomplete="off" />
      <input class="input" type="password" bind:value={form.password} placeholder={passwordHint === 'Optional' ? 'Password' : passwordHint} aria-label="Password" autocomplete="off" />
    </div>

    <span></span>
    <label class="check"><input type="checkbox" bind:checked={form.savePassword} />Save passwords in the Keychain</label>

    <label for="cf-db">Database</label>
    <div class="form-row">
      <input id="cf-db" class="input" bind:value={form.database} placeholder={e.databaseHint} spellcheck="false" autocomplete="off" />
      <div class="ssl"><Select id="cf-ssl" bind:value={form.sslMode} options={sslOptions} aria-label="SSL" /></div>
    </div>
  {/if}
</div>

<style>
  .ssl { width: 150px; flex: none; }
  .check { display: flex; align-items: center; gap: 7px; margin-top: -4px; color: var(--text-3); font-size: 12px; }
</style>
