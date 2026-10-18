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
  import ConnAvatar from './ConnAvatar.svelte';
  import { pictures, squarePicture } from './pictures.svelte';
  import ToggleCard from '../ui/ToggleCard.svelte';
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
  let showUrl = $state(false);
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
    return moved ? `Enter the ${what} again` : `Saved — leave empty to keep`;
  }
  const formEngine = $derived(engine(form.driver));

  function setDriver(d: Driver) {
    if (d === form.driver) return;
    const fresh = emptyConnection(engine(d));
    form = { ...fresh, id: form.id, name: form.name, group: form.group, env: form.env, password: form.password, savePassword: form.savePassword, readOnly: form.readOnly, ssh: form.ssh };
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
      group: form.group,
      env: form.env,
      readOnly: form.readOnly,
      savePassword: form.savePassword,
      ssh: form.ssh,
    };
    test = null;
    showUrl = false;
  }

  function normalized(): Connection {
    const c = $state.snapshot(form) as Connection;
    const e = engine(c.driver);
    c.name = c.name.trim();
    c.group = (c.group ?? '').trim();
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

  let picture = $state<string | undefined>(undefined);
  let pictureInput = $state<HTMLInputElement>();
  const shownPicture = $derived(picture ?? pictures.byId[form.id] ?? '');

  async function pickPicture(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) return;
    try {
      picture = await squarePicture(file);
      error = '';
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
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
      if (picture !== undefined) await pictures.set(saved.id, picture);
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

<Modal title={editing ? 'Edit connection' : 'New connection'} width={640} {onclose}>
  {#snippet header()}
    <div class="head">
      <span class="pic">
        <button type="button" class="pic-pick" onclick={() => pictureInput?.click()} title="Choose a picture" aria-label="Choose a picture">
          <ConnAvatar connection={{ ...form, name: form.name || formEngine.name }} size={40} showDriver={false} picture={shownPicture} />
          <span class="pic-hover"><Icon name="pencil" size={13} /></span>
        </button>
        {#if shownPicture}<button type="button" class="pic-remove" onclick={() => (picture = '')} title="Remove the picture" aria-label="Remove the picture"><Icon name="x" size={9} /></button>{/if}
        <input bind:this={pictureInput} type="file" accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml,.svg" hidden onchange={pickPicture} />
      </span>
      <div class="head-text">
        <div class="head-title">{editing ? 'Edit connection' : 'New connection'}</div>
        <div class="head-sub">or <button type="button" class="link" onclick={() => (showUrl = !showUrl)}>paste a URL</button> — postgres://, mysql://, redis://…</div>
      </div>
      <button type="button" class="btn ghost icon close" onclick={onclose} aria-label="Close"><Icon name="x" size={15} /></button>
    </div>
  {/snippet}

  <form id="connection-form" {onsubmit}>
    {#if showUrl}<div class="url"><UrlImport onimport={importParsed} /></div>{/if}

    <DriverPicker driver={form.driver} onpick={setDriver} />

    <div class="fields">
      <DbFields bind:form passwordHint={secretHint('password', dbMoved)} onenv={onEnvChange} onerror={m => (error = m)} />
    </div>

    {#if !formEngine.file}
      <SshTunnelForm
        bind:form
        defaultPort={formEngine.defaultPort}
        passwordHint={secretHint('SSH password', sshMoved, '')}
        passphraseHint={secretHint('passphrase', keyMoved, 'Optional — only if the key is encrypted')}
        onerror={m => (error = m)}
      />
    {/if}

    <div class="toggles">
      <ToggleCard bind:checked={form.readOnly} onchange={() => (readOnlyTouched = true)} title="Read only" hint="Writes are refused" />
      <ToggleCard bind:checked={form.aiAccess} title="AI agents (MCP)" hint="Open to MCP clients" />
    </div>

    {#if test && !test.ok}
      <div class="result" role="status"><Icon name="alert" /><span>{test.error}</span></div>
    {/if}
    {#if error}
      <div class="result" role="alert"><Icon name="alert" /><span>{error}</span></div>
    {/if}
  </form>

  {#snippet footer()}
    <button type="button" class="btn outline" onclick={runTest} disabled={testing}>{#if testing}<Spinner size={11} />Testing…{:else}Test{/if}</button>
    {#if test?.ok}
      <span class="tested" role="status">
        <span class="dot"></span>Connected{form.ssh.enabled && !formEngine.file ? ' via SSH' : ''} · {test.serverVersion} · {formatDuration(test.latencyMs)}
      </span>
    {/if}
    <span style="flex:1"></span>
    <button type="button" class="btn ghost" onclick={onclose}>Cancel</button>
    <button type="button" class="btn" onclick={() => save(false)} disabled={saving}>Save</button>
    <button type="submit" form="connection-form" class="btn primary" disabled={saving}>Save &amp; connect</button>
  {/snippet}
</Modal>

<style>
  .head { display: flex; align-items: center; gap: 12px; font-weight: 400; }
  .head-text { flex: 1; min-width: 0; }
  .pic { position: relative; flex: none; }
  .pic-pick { position: relative; display: block; padding: 0; border: 0; border-radius: 24%; background: transparent; }
  .pic-hover { position: absolute; inset: 0; display: grid; place-items: center; border-radius: 24%; background: rgba(0, 0, 0, 0.45); color: #fff; opacity: 0; transition: opacity 0.12s; }
  .pic-pick:hover .pic-hover, .pic-pick:focus-visible .pic-hover { opacity: 1; }
  .pic-remove { position: absolute; top: -5px; right: -5px; display: grid; place-items: center; width: 16px; height: 16px; padding: 0; border: 1px solid var(--border); border-radius: 50%; background: var(--elevated); color: var(--text-2); opacity: 0; }
  .pic:hover .pic-remove, .pic-remove:focus-visible { opacity: 1; }
  .head-title { font-size: 16px; font-weight: 600; }
  .head-sub { color: var(--text-3); font-size: 12px; }
  .link { padding: 0; border: 0; background: transparent; color: var(--accent-text); font: inherit; }
  .link:hover { text-decoration: underline; }
  .close { margin: -6px -6px 0 0; align-self: flex-start; }
  .url { margin-bottom: 14px; }
  .fields { margin-top: 20px; }
  .toggles { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin-top: 10px; }
  .tested { min-width: 0; display: flex; align-items: center; gap: 6px; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; color: var(--ok); font-size: 12.5px; }
  .dot { width: 6px; height: 6px; flex: none; border-radius: 50%; background: currentColor; }
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
</style>
