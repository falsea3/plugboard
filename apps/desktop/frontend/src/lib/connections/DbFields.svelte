<script lang="ts">
  import { api, type Connection, type Env } from '../api/backend';
  import { engine, nameFromFile } from '../engines';
  import Icon from '../ui/Icon.svelte';
  import Select from '../ui/Select.svelte';
  import ToggleRow from '../ui/ToggleRow.svelte';

  let {
    form = $bindable(),
    passwordHint,
    onenv,
    onreadonly,
    onerror,
  }: { form: Connection; passwordHint: string; onenv: () => void; onreadonly: () => void; onerror: (message: string) => void } = $props();

  const envs: { value: Env; label: string }[] = [
    { value: '', label: 'None' },
    { value: 'local', label: 'Local' },
    { value: 'dev', label: 'Development' },
    { value: 'staging', label: 'Staging' },
    { value: 'prod', label: 'Production' },
  ];
  const sslModes = [
    { value: '', label: 'Preferred', hint: 'encrypt if the server can' },
    { value: 'disable', label: 'Disabled' },
    { value: 'require', label: 'Required', hint: 'encrypt, trust any certificate' },
    { value: 'verify-full', label: 'Verify certificate' },
  ];
  const e = $derived(engine(form.driver));

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
  <div class="form-row">
    <input id="cf-name" class="input" bind:value={form.name} placeholder={e.file ? 'Optional — named after the file' : 'Optional — e.g. Production replica'} autocomplete="off" spellcheck="false" />
    <div class="env"><Select bind:value={form.env} options={envs} onchange={onenv} aria-label="Environment tag" /></div>
  </div>

  {#if e.file}
    <label for="cf-file">File</label>
    <div class="form-row">
      <input id="cf-file" class="input mono" bind:value={form.file} placeholder="/path/to/database.db" spellcheck="false" />
      <button type="button" class="btn" onclick={chooseFile}><Icon name="folder" />Choose…</button>
    </div>
  {:else}
    <label for="cf-host">Host</label>
    <div class="form-row">
      <input id="cf-host" class="input" bind:value={form.host} placeholder="127.0.0.1" spellcheck="false" autocomplete="off" />
      <input class="input port" type="number" bind:value={form.port} min="1" max="65535" aria-label="Port" placeholder={String(e.defaultPort)} />
    </div>

    <label for="cf-user">User</label>
    <input id="cf-user" class="input" bind:value={form.user} spellcheck="false" autocomplete="off" />

    <label for="cf-password">Password</label>
    <div class="form-row">
      <input id="cf-password" class="input" type="password" bind:value={form.password} placeholder={passwordHint} autocomplete="off" />
      <label class="check"><input type="checkbox" bind:checked={form.savePassword} />Save in Keychain</label>
    </div>

    <label for="cf-db">Database</label>
    <input id="cf-db" class="input" bind:value={form.database} placeholder={e.defaultDatabase ? `Optional — defaults to ${e.defaultDatabase}` : 'Optional'} spellcheck="false" autocomplete="off" />

    <label for="cf-ssl">SSL</label>
    <Select id="cf-ssl" bind:value={form.sslMode} options={sslModes} aria-label="SSL" />
  {/if}

  <span></span>
  <ToggleRow bind:checked={form.readOnly} onchange={onreadonly}>
    {#snippet title()}<Icon name="lock" size={13} />Read-only{/snippet}
    The session can only read. Writes are refused by Plugboard and by the server. You can switch it per session from the sidebar.
  </ToggleRow>
</div>

<style>
  .env { width: 132px; flex: none; }
  .check {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: none;
    color: var(--text-2);
    font-size: 12px;
    white-space: nowrap;
  }
</style>
