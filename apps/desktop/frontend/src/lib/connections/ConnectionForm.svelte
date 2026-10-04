<script lang="ts">
  import { untrack } from 'svelte';
  import {
    api,
    type Connection, type ConnectSecrets, type Driver, type TestResult,
  } from '../api/backend';
  import { emptyConnection, engine, nameFromFile } from '../engines';
  import { app } from '../app/app.svelte';
  import Modal from '../ui/Modal.svelte';
  import DriverPicker from './DriverPicker.svelte';
  import DbFields from './DbFields.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import UrlImport from './UrlImport.svelte';
  import SshTunnelForm from './SshTunnelForm.svelte';
  import { formatDuration } from '../ui/format';

  let {
    initial,
    onclose,
    onconnect,
  }: {
    initial: Connection | null;
    onclose: () => void;
    onconnect: (c: Connection, secrets?: Partial<ConnectSecrets>) => void;
  } = $props();

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
  let readOnlyTouched = editing;

  const hasStored = editing && start?.savePassword;
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
  const formEngine = $derived(engine(form.driver));

  function setDriver(d: Driver) {
    if (d === form.driver) return;
    const fresh = emptyConnection(engine(d));
    form = { ...fresh, id: form.id, name: form.name, env: form.env, password: form.password, savePassword: form.savePassword, readOnly: form.readOnly, ssh: form.ssh };
    if (engine(d).file) tab = 'general';
    test = null;
  }

  function onEnvChange() {
    if (!readOnlyTouched && form.env === 'prod') form.readOnly = true;
  }

  function importParsed(parsed: Connection) {
    form = {
      ...parsed,
      id: form.id,
      name: form.name || parsed.name,
      env: form.env,
      readOnly: form.readOnly,
      savePassword: form.savePassword,
      ssh: form.ssh,
    };
    test = null;
    tab = 'general';
  }

  function normalized(): Connection {
    const c = $state.snapshot(form) as Connection;
    const e = engine(c.driver);
    c.name = c.name.trim();
    if (!c.name) {
      c.name = e.file ? nameFromFile(e, c.file) : `${c.host}${c.database ? '/' + c.database : ''}`;
    }
    c.port = Number(c.port) || e.defaultPort;
    c.ssh.port = Number(c.ssh.port) || 22;
    if (e.file) c.ssh.enabled = false;
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
    <UrlImport onimport={importParsed} />

    <DriverPicker driver={form.driver} onpick={setDriver} />

    {#if !formEngine.file}
      <div class="tabs" role="tablist">
        <button type="button" role="tab" aria-selected={tab === 'general'} class:on={tab === 'general'} onclick={() => (tab = 'general')}>General</button>
        <button type="button" role="tab" aria-selected={tab === 'ssh'} class:on={tab === 'ssh'} onclick={() => (tab = 'ssh')}>
          SSH Tunnel
          {#if form.ssh.enabled}<span class="on-dot" title="SSH tunnel is on"></span>{/if}
        </button>
      </div>
    {/if}

    {#if tab === 'general'}
      <DbFields
        bind:form
        passwordHint={secretHint('password', dbMoved)}
        onenv={onEnvChange}
        onreadonly={() => (readOnlyTouched = true)}
        onerror={m => (error = m)}
      />
    {:else}
      <SshTunnelForm
        bind:form
        defaultPort={formEngine.defaultPort}
        passwordHint={secretHint('SSH password', sshMoved, '')}
        passphraseHint={secretHint('passphrase', keyMoved, 'Optional — only if the key is encrypted')}
        onerror={m => (error = m)}
      />
    {/if}

    {#if test}
      <div class="result" class:ok={test.ok} role="status">
        <Icon name={test.ok ? 'check' : 'alert'} />
        {#if test.ok}
          <span>Connected{form.ssh.enabled && !formEngine.file ? ' via SSH' : ''} · {test.serverVersion} · {formatDuration(test.latencyMs)}</span>
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
    <button type="button" class="btn" onclick={runTest} disabled={testing}>{#if testing}<Spinner size={11} />Testing…{:else}Test{/if}</button>
    <span style="flex:1"></span>
    <button type="button" class="btn" onclick={onclose}>Cancel</button>
    <button type="button" class="btn" onclick={() => save(false)} disabled={saving}>Save</button>
    <button type="submit" form="connection-form" class="btn primary" disabled={saving}>Connect</button>
  {/snippet}
</Modal>

<style>
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
