<script lang="ts">
  import { untrack } from 'svelte';
  import {
    api, DEFAULT_PORTS, DRIVER_LABELS, emptyConnection,
    type Connection, type ConnectSecrets, type Driver, type Env, type SSHAuth, type TestResult,
  } from '../backend';
  import { app } from '../stores/app.svelte';
  import { parseConnectionUrl } from '../connectionUrl';
  import Modal from './Modal.svelte';
  import DriverMark from './DriverMark.svelte';
  import Icon from './Icon.svelte';
  import Select from './Select.svelte';
  import { formatDuration, sqliteName } from '../format';

  let {
    initial,
    onclose,
    onconnect,
  }: {
    initial: Connection | null;
    onclose: () => void;
    /** secrets is undefined when the profile saves them (use the stored ones). */
    onconnect: (c: Connection, secrets?: Partial<ConnectSecrets>) => void;
  } = $props();

  // The form edits a snapshot; later changes to `initial` don't reset it.
  const start = untrack(() => initial);
  const editing = start !== null && start.id !== '';
  const blank = emptyConnection();
  let form = $state<Connection>(
    start
      ? {
          ...blank,
          ...start,
          password: '',
          ssh: {
            ...blank.ssh,
            ...start.ssh,
            port: start.ssh?.port || 22,
            auth: start.ssh?.auth || 'password',
            password: '',
            passphrase: '',
          },
        }
      : blank,
  );
  let tab = $state<'general' | 'ssh'>('general');
  let testing = $state(false);
  let saving = $state(false);
  let test = $state<TestResult | null>(null);
  let error = $state('');
  let url = $state('');
  let urlError = $state('');
  let readOnlyTouched = editing;

  const drivers: Driver[] = ['postgres', 'mysql', 'sqlite'];
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
  const sshAuths: { value: SSHAuth; label: string }[] = [
    { value: 'password', label: 'Password' },
    { value: 'key', label: 'Private key' },
    { value: 'agent', label: 'SSH agent' },
  ];
  const hasStored = editing && start?.savePassword;
  // A saved secret is only reused while what it unlocks stays the same (see store.secretFields).
  const sameText = (a = '', b = '') => a.trim().toLowerCase() === b.trim().toLowerCase();
  const dbMoved = $derived(
    !!start && (form.driver !== start.driver || !sameText(form.host, start.host) || Number(form.port) !== start.port || form.user !== start.user),
  );
  const sshMoved = $derived(
    !!start &&
      (!sameText(form.ssh.host || form.host, start.ssh?.host || start.host) || Number(form.ssh.port) !== start.ssh?.port || form.ssh.user !== start.ssh?.user),
  );
  const keyMoved = $derived(!!start && form.ssh.keyFile !== start.ssh?.keyFile);
  function secretHint(what: string, moved: boolean, otherwise = 'Optional'): string {
    if (!hasStored) return otherwise;
    return moved ? `Enter it again — the saved ${what} doesn't carry over to this change` : `Optional — keeps the saved ${what}`;
  }
  const sshViaDbHost = $derived(!form.ssh.host.trim());

  function setDriver(d: Driver) {
    if (d === form.driver) return;
    const fresh = emptyConnection(d);
    // Keep what isn't driver-specific.
    form = { ...fresh, id: form.id, name: form.name, env: form.env, password: form.password, savePassword: form.savePassword, readOnly: form.readOnly, ssh: form.ssh };
    if (d === 'sqlite') tab = 'general';
    test = null;
  }

  function onEnvChange() {
    // New Production connections start read-only; an explicit choice wins.
    if (!readOnlyTouched && form.env === 'prod') form.readOnly = true;
  }

  function importUrl(text = url) {
    urlError = '';
    if (!text.trim()) return;
    try {
      const parsed = parseConnectionUrl(text);
      form = {
        ...parsed,
        id: form.id,
        name: form.name || parsed.name,
        env: form.env,
        readOnly: form.readOnly,
        savePassword: form.savePassword,
        ssh: form.ssh,
      };
      url = '';
      test = null;
      tab = 'general';
    } catch (err) {
      urlError = err instanceof Error ? err.message : String(err);
    }
  }

  function onUrlPaste(e: ClipboardEvent) {
    const text = e.clipboardData?.getData('text') ?? '';
    if (/^\s*(jdbc:)?[a-z0-9+]+:\/\//i.test(text) || /^\s*[A-Z_]+\s*=/.test(text)) {
      e.preventDefault();
      importUrl(text);
    }
  }

  async function chooseFile() {
    try {
      const path = await api.chooseSQLiteFile();
      if (path) {
        form.file = path;
        if (!form.name) form.name = sqliteName(path);
      }
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  async function chooseKey() {
    try {
      const path = await api.chooseSSHKeyFile();
      if (path) form.ssh.keyFile = path;
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  function normalized(): Connection {
    const c = $state.snapshot(form) as Connection;
    c.name = c.name.trim();
    if (!c.name) {
      c.name = c.driver === 'sqlite' ? sqliteName(c.file) : `${c.host}${c.database ? '/' + c.database : ''}`;
    }
    c.port = Number(c.port) || DEFAULT_PORTS[c.driver];
    c.ssh.port = Number(c.ssh.port) || 22;
    if (c.driver === 'sqlite') c.ssh.enabled = false;
    return c;
  }

  async function runTest() {
    testing = true;
    test = null;
    error = '';
    try {
      test = await api.testConnection(normalized());
    } catch (err) {
      test = { ok: false, error: err instanceof Error ? err.message : String(err), latencyMs: 0 };
    } finally {
      testing = false;
    }
  }

  async function save(andConnect: boolean) {
    saving = true;
    error = '';
    try {
      const c = normalized();
      const typed: Partial<ConnectSecrets> = {
        password: c.password ?? '',
        sshPassword: c.ssh.password ?? '',
        sshPassphrase: c.ssh.passphrase ?? '',
      };
      const saved = await app.saveConnection(c);
      onclose();
      if (andConnect) onconnect(saved, saved.savePassword ? undefined : typed);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      saving = false;
    }
  }

  function onsubmit(e: SubmitEvent) {
    e.preventDefault();
    save(true);
  }
</script>

<Modal title={editing ? 'Edit connection' : 'New connection'} width={560} {onclose}>
  <form id="connection-form" {onsubmit}>
    <div class="url-row">
      <Icon name="link" size={14} />
      <input
        class="url-input mono"
        bind:value={url}
        onpaste={onUrlPaste}
        onkeydown={e => { if (e.key === 'Enter') { e.preventDefault(); importUrl(); } }}
        placeholder="Paste a URL — postgres://user:pass@host:5432/db"
        spellcheck="false"
        autocomplete="off"
        aria-label="Import from connection URL"
      />
      <button type="button" class="btn sm" onclick={() => importUrl()} disabled={!url.trim()}>Import</button>
    </div>
    {#if urlError}<div class="url-error">{urlError}</div>{/if}

    <div class="drivers" role="radiogroup" aria-label="Database">
      {#each drivers as d (d)}
        <button type="button" role="radio" aria-checked={form.driver === d} aria-label={DRIVER_LABELS[d]} class="driver" class:selected={form.driver === d} onclick={() => setDriver(d)}>
          <DriverMark driver={d} size={26} />
          <span>{DRIVER_LABELS[d]}</span>
        </button>
      {/each}
    </div>

    {#if form.driver !== 'sqlite'}
      <div class="tabs" role="tablist">
        <button type="button" role="tab" aria-selected={tab === 'general'} class:on={tab === 'general'} onclick={() => (tab = 'general')}>General</button>
        <button type="button" role="tab" aria-selected={tab === 'ssh'} class:on={tab === 'ssh'} onclick={() => (tab = 'ssh')}>
          SSH Tunnel
          {#if form.ssh.enabled}<span class="on-dot" title="SSH tunnel is on"></span>{/if}
        </button>
      </div>
    {/if}

    {#if tab === 'general'}
      <div class="grid">
        <label for="cf-name">Name</label>
        <div class="row">
          <input id="cf-name" class="input" bind:value={form.name} placeholder={form.driver === 'sqlite' ? 'Optional — named after the file' : 'Optional — e.g. Production replica'} autocomplete="off" spellcheck="false" />
          <div class="env"><Select bind:value={form.env} options={envs} onchange={onEnvChange} aria-label="Environment tag" /></div>
        </div>

        {#if form.driver === 'sqlite'}
          <label for="cf-file">File</label>
          <div class="row">
            <input id="cf-file" class="input mono" bind:value={form.file} placeholder="/path/to/database.db" spellcheck="false" />
            <button type="button" class="btn" onclick={chooseFile}><Icon name="folder" />Choose…</button>
          </div>
        {:else}
          <label for="cf-host">Host</label>
          <div class="row">
            <input id="cf-host" class="input" bind:value={form.host} placeholder="127.0.0.1" spellcheck="false" autocomplete="off" />
            <input class="input port" type="number" bind:value={form.port} min="1" max="65535" aria-label="Port" placeholder={String(DEFAULT_PORTS[form.driver])} />
          </div>

          <label for="cf-user">User</label>
          <input id="cf-user" class="input" bind:value={form.user} spellcheck="false" autocomplete="off" />

          <label for="cf-password">Password</label>
          <div class="row">
            <input id="cf-password" class="input" type="password" bind:value={form.password} placeholder={secretHint('password', dbMoved)} autocomplete="off" />
            <label class="check"><input type="checkbox" bind:checked={form.savePassword} />Save in Keychain</label>
          </div>

          <label for="cf-db">Database</label>
          <input id="cf-db" class="input" bind:value={form.database} placeholder={form.driver === 'postgres' ? 'Optional — defaults to postgres' : 'Optional'} spellcheck="false" autocomplete="off" />

          <label for="cf-ssl">SSL</label>
          <Select id="cf-ssl" bind:value={form.sslMode} options={sslModes} aria-label="SSL" />
        {/if}

        <span></span>
        <label class="toggle-row">
          <span class="switch">
            <input type="checkbox" bind:checked={form.readOnly} onchange={() => (readOnlyTouched = true)} />
            <span class="track"></span>
          </span>
          <span>
            <span class="toggle-title"><Icon name="lock" size={13} />Read-only</span>
            <span class="toggle-hint">The session can only read. Writes are refused by Relay DB and by the server. You can switch it per session from the sidebar.</span>
          </span>
        </label>
      </div>
    {:else}
      <div class="grid">
        <span></span>
        <label class="toggle-row">
          <span class="switch">
            <input type="checkbox" bind:checked={form.ssh.enabled} />
            <span class="track"></span>
          </span>
          <span>
            <span class="toggle-title">Connect through an SSH server</span>
            <span class="toggle-hint">
              {#if sshViaDbHost}
                Relay DB signs in to <strong>{form.host || 'the database host'}</strong> over SSH and reaches the database there on <code>127.0.0.1:{form.port || DEFAULT_PORTS[form.driver]}</code>. Set an SSH host to go through a separate bastion instead.
              {:else}
                Relay DB signs in to <strong>{form.ssh.host}</strong> and connects from there to <code>{form.host || '127.0.0.1'}:{form.port || DEFAULT_PORTS[form.driver]}</code> — the database host as that server sees it.
              {/if}
            </span>
          </span>
        </label>

        {#if form.ssh.enabled}
          <label for="cf-ssh-host">SSH host</label>
          <div class="row">
            <input id="cf-ssh-host" class="input" bind:value={form.ssh.host} placeholder={form.host ? `Optional — same as database host (${form.host})` : 'Optional — same as database host'} spellcheck="false" autocomplete="off" />
            <input class="input port" type="number" bind:value={form.ssh.port} min="1" max="65535" aria-label="SSH port" placeholder="22" />
          </div>

          <label for="cf-ssh-user">SSH user</label>
          <input id="cf-ssh-user" class="input" bind:value={form.ssh.user} placeholder="e.g. root or ubuntu" spellcheck="false" autocomplete="off" />

          <span class="label">Sign in with</span>
          <div class="segmented" role="radiogroup" aria-label="SSH authentication">
            {#each sshAuths as a (a.value)}
              <button type="button" role="radio" aria-checked={form.ssh.auth === a.value} class:on={form.ssh.auth === a.value} onclick={() => (form.ssh.auth = a.value)}>{a.label}</button>
            {/each}
          </div>

          {#if form.ssh.auth === 'password'}
            <label for="cf-ssh-pw">Password</label>
            <input id="cf-ssh-pw" class="input" type="password" bind:value={form.ssh.password} placeholder={secretHint('SSH password', sshMoved, '')} autocomplete="off" />
          {:else if form.ssh.auth === 'key'}
            <label for="cf-ssh-key">Key file</label>
            <div class="row">
              <input id="cf-ssh-key" class="input mono" bind:value={form.ssh.keyFile} placeholder="~/.ssh/id_ed25519" spellcheck="false" />
              <button type="button" class="btn" onclick={chooseKey}><Icon name="folder" />Choose…</button>
            </div>
            <label for="cf-ssh-pass">Passphrase</label>
            <input id="cf-ssh-pass" class="input" type="password" bind:value={form.ssh.passphrase} placeholder={secretHint('passphrase', keyMoved, 'Optional — only if the key is encrypted')} autocomplete="off" />
          {:else}
            <span></span>
            <p class="toggle-hint agent-hint">Uses the keys loaded in your running ssh-agent (<code>ssh-add -l</code>), including 1Password and Secretive agents.</p>
          {/if}

          <span></span>
          <p class="toggle-hint">Secrets follow “Save in Keychain” on the General tab. On first connect the server’s host key is remembered; if it ever changes, Relay DB refuses to connect.</p>
        {/if}
      </div>
    {/if}

    {#if test}
      <div class="result" class:ok={test.ok} role="status">
        <Icon name={test.ok ? 'check' : 'alert'} />
        {#if test.ok}
          <span>Connected{form.ssh.enabled && form.driver !== 'sqlite' ? ' via SSH' : ''} · {test.serverVersion} · {formatDuration(test.latencyMs)}</span>
        {:else}
          <span>{test.error}</span>
        {/if}
      </div>
    {/if}
    {#if error}
      <div class="result" role="alert"><Icon name="alert" /><span>{error}</span></div>
    {/if}
  </form>

  {#snippet footer()}
    <button type="button" class="btn" onclick={runTest} disabled={testing}>{testing ? 'Testing…' : 'Test'}</button>
    <span style="flex:1"></span>
    <button type="button" class="btn" onclick={onclose}>Cancel</button>
    <button type="button" class="btn" onclick={() => save(false)} disabled={saving}>Save</button>
    <button type="submit" form="connection-form" class="btn primary" disabled={saving}>Connect</button>
  {/snippet}
</Modal>

<style>
  .url-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 4px 0 10px;
    height: 32px;
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text-3);
  }
  .url-row:focus-within { border-style: solid; border-color: var(--accent); }
  .url-input {
    flex: 1;
    min-width: 0;
    height: 100%;
    border: 0;
    outline: none;
    background: transparent;
    color: var(--text);
    font-size: 12px;
  }
  .url-input::placeholder { color: var(--text-3); font-family: var(--font-ui); }
  .url-error { margin: 6px 2px 0; font-size: 12px; color: var(--danger); }

  .drivers {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    margin: 14px 0;
  }
  .driver {
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 8px 10px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    font-weight: 500;
    text-align: left;
  }
  .driver:hover { background: var(--hover); }
  .driver.selected {
    border-color: var(--accent);
    background: var(--accent-dim);
    box-shadow: 0 0 0 1px var(--accent);
  }

  .tabs {
    display: flex;
    gap: 2px;
    margin-bottom: 14px;
    border-bottom: 1px solid var(--border-subtle);
  }
  .tabs button {
    position: relative;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 30px;
    padding: 0 12px;
    border: 0;
    background: transparent;
    color: var(--text-2);
    font-size: 12.5px;
    font-weight: 500;
  }
  .tabs button:hover { color: var(--text); }
  .tabs button.on { color: var(--text); }
  .tabs button.on::after {
    content: '';
    position: absolute;
    left: 8px;
    right: 8px;
    bottom: -1px;
    height: 2px;
    border-radius: 2px;
    background: var(--accent);
  }
  .on-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--ok); }

  .grid {
    display: grid;
    grid-template-columns: 84px 1fr;
    align-items: center;
    gap: 9px 12px;
  }
  .grid > label, .grid > .label {
    color: var(--text-2);
    text-align: right;
    font-size: 12.5px;
  }
  .row { display: flex; gap: 8px; align-items: center; }
  .port { width: 86px; flex: none; }
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

  .segmented {
    display: flex;
    padding: 2px;
    border-radius: 7px;
    background: var(--elevated);
    border: 1px solid var(--border-subtle);
  }
  .segmented button {
    flex: 1;
    height: 24px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text-2);
    font-size: 12px;
    font-weight: 500;
  }
  .segmented button.on { background: var(--bg); color: var(--text); box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2); }

  .toggle-row { display: flex; align-items: flex-start; gap: 10px; padding: 4px 0; text-align: left !important; }
  .toggle-title { display: flex; align-items: center; gap: 6px; color: var(--text); font-weight: 550; font-size: 12.5px; }
  .toggle-hint { display: block; margin: 2px 0 0; color: var(--text-3); font-size: 11.5px; line-height: 1.45; }
  .agent-hint { margin: 0; }
  code { font-family: var(--font-mono); font-size: 11px; }
  .toggle-hint strong { color: var(--text-2); font-weight: 550; }

  .switch { position: relative; flex: none; width: 30px; height: 18px; margin-top: 1px; }
  .switch input { position: absolute; inset: 0; opacity: 0; margin: 0; }
  .track { position: absolute; inset: 0; border-radius: 999px; background: var(--active); transition: background 0.15s; pointer-events: none; }
  .track::after {
    content: '';
    position: absolute;
    top: 2px;
    left: 2px;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
    transition: transform 0.15s;
  }
  .switch input:checked + .track { background: var(--accent); }
  .switch input:checked + .track::after { transform: translateX(12px); }
  .switch input:focus-visible + .track { outline: 2px solid var(--accent); outline-offset: 2px; }

  .result {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    margin-top: 14px;
    padding: 8px 10px;
    border-radius: var(--radius);
    font-size: 12px;
    color: var(--danger);
    background: color-mix(in srgb, var(--danger) 10%, transparent);
    user-select: text;
    -webkit-user-select: text;
    word-break: break-word;
  }
  .result :global(.icon) { margin-top: 1px; }
  .result.ok {
    color: var(--ok);
    background: color-mix(in srgb, var(--ok) 10%, transparent);
  }
</style>
