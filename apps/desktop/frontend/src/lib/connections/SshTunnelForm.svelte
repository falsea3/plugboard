<script lang="ts">
  import { api, type Connection, type SSHAuth } from '../api/backend';
  import Icon from '../ui/Icon.svelte';
  import Switch from '../ui/Switch.svelte';

  let {
    form = $bindable(),
    defaultPort,
    passwordHint,
    passphraseHint,
    onerror,
  }: { form: Connection; defaultPort: number; passwordHint: string; passphraseHint: string; onerror: (message: string) => void } = $props();

  const auths: { value: SSHAuth; label: string }[] = [
    { value: 'password', label: 'Password' },
    { value: 'key', label: 'Private key' },
    { value: 'agent', label: 'SSH agent' },
  ];
  const viaDbHost = $derived(!form.ssh.host.trim());
  const summary = $derived.by(() => {
    if (!form.ssh.enabled) return 'off';
    const host = form.ssh.host.trim() || form.host || 'database host';
    const port = Number(form.ssh.port) && Number(form.ssh.port) !== 22 ? `:${form.ssh.port}` : '';
    const auth = auths.find(a => a.value === form.ssh.auth)?.label.toLowerCase() ?? '';
    return `${form.ssh.user ? form.ssh.user + '@' : ''}${host}${port} · ${auth}`;
  });

  async function chooseKey() {
    try {
      const path = await api.chooseSSHKeyFile();
      if (path) form.ssh.keyFile = path;
    } catch (err) {
      onerror(err instanceof Error ? err.message : String(err));
    }
  }
</script>

<section class="card" class:on={form.ssh.enabled}>
  <div class="card-head">
    <span class="what">
      <span class="title">SSH tunnel</span>
      <span class="summary">{summary}</span>
    </span>
    <Switch bind:checked={form.ssh.enabled} label="SSH tunnel" />
  </div>
  {#if form.ssh.enabled}
  <p class="hint route">
    {#if viaDbHost}
      Signs in to <strong>{form.host || 'the database host'}</strong> over SSH and reaches the database there on <code>127.0.0.1:{form.port || defaultPort}</code>. Set an SSH host to go through a separate bastion.
    {:else}
      Signs in to <strong>{form.ssh.host}</strong> and connects from there to <code>{form.host || '127.0.0.1'}:{form.port || defaultPort}</code> — the database host as that server sees it.
    {/if}
  </p>
  <div class="form-grid">
    <label for="cf-ssh-host">SSH host</label>
    <div class="form-row">
      <input id="cf-ssh-host" class="input" bind:value={form.ssh.host} placeholder={form.host ? `Optional — same as database host (${form.host})` : 'Optional — same as database host'} spellcheck="false" autocomplete="off" />
      <input class="input port" type="number" bind:value={form.ssh.port} min="1" max="65535" aria-label="SSH port" placeholder="22" />
    </div>

    <label for="cf-ssh-user">SSH user</label>
    <input id="cf-ssh-user" class="input" bind:value={form.ssh.user} placeholder="e.g. root or ubuntu" spellcheck="false" autocomplete="off" />

    <span class="label">Sign in with</span>
    <div class="segmented" role="radiogroup" aria-label="SSH authentication">
      {#each auths as a (a.value)}
        <button type="button" role="radio" aria-checked={form.ssh.auth === a.value} class:on={form.ssh.auth === a.value} onclick={() => (form.ssh.auth = a.value)}>{a.label}</button>
      {/each}
    </div>

    {#if form.ssh.auth === 'password'}
      <label for="cf-ssh-pw">Password</label>
      <input id="cf-ssh-pw" class="input" type="password" bind:value={form.ssh.password} placeholder={passwordHint} autocomplete="off" />
    {:else if form.ssh.auth === 'key'}
      <label for="cf-ssh-key">Key file</label>
      <div class="form-row">
        <input id="cf-ssh-key" class="input mono" bind:value={form.ssh.keyFile} placeholder="~/.ssh/id_ed25519" spellcheck="false" />
        <button type="button" class="btn" onclick={chooseKey}><Icon name="folder" />Choose…</button>
      </div>
      <label for="cf-ssh-pass">Passphrase</label>
      <input id="cf-ssh-pass" class="input" type="password" bind:value={form.ssh.passphrase} placeholder={passphraseHint} autocomplete="off" />
    {:else}
      <span></span>
      <p class="hint">Uses the keys loaded in your running ssh-agent (<code>ssh-add -l</code>), including 1Password and Secretive agents.</p>
    {/if}

    <span></span>
    <p class="hint">Secrets follow “Save passwords in the Keychain”. On first connect the server’s host key is remembered; if it ever changes, Plugboard refuses to connect.</p>
  </div>
  {/if}
</section>

<style>
  .card { margin-top: 18px; padding: 12px 14px; border: 1px solid var(--border-subtle); border-radius: 12px; background: var(--surface); }
  .card.on .form-grid { margin-top: 12px; grid-template-columns: 89px minmax(0, 1fr); }
  .card-head { display: flex; align-items: center; gap: 10px; }
  .what { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .title { font-weight: 500; }
  .summary { color: var(--text-3); }
  .summary::before { content: ' · '; }
  .route { margin-top: 8px; }
  .route strong { color: var(--text-2); font-weight: 550; }
  .hint { margin: 0; color: var(--text-3); font-size: 11.5px; line-height: 1.45; }
  code { font-family: var(--font-mono); font-size: 11px; }
  .segmented { display: flex; padding: 3px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--card); }
  .segmented button { flex: 1; height: 26px; border: 0; border-radius: var(--radius-sm); background: transparent; color: var(--text-2); font-size: 12.5px; }
  .segmented button:hover { color: var(--text); }
  .segmented button.on { background: var(--active); color: var(--text); font-weight: 500; }
</style>
