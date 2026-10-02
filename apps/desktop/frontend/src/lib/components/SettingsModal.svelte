<script lang="ts">
  import { api, type AppInfo, type ThemeMode } from '../backend';
  import { app, type SettingsSection } from '../stores/app.svelte';
  import Icon, { type IconName } from './Icon.svelte';
  import Select from './Select.svelte';
  import mark from '../assets/relay-db-mark.png';

  let info = $state<AppInfo | null>(null);
  api.appInfo().then(i => (info = i)).catch(() => {});

  const section = $derived(app.settingsOpen ?? 'general');
  const s = $derived(app.settings);

  const sections: { id: SettingsSection; label: string; icon: IconName }[] = [
    { id: 'general', label: 'General', icon: 'settings' },
    { id: 'editor', label: 'SQL Editor', icon: 'code' },
    { id: 'about', label: 'About', icon: 'info' },
  ];

  const themes: { id: ThemeMode; label: string; icon: IconName }[] = [
    { id: 'system', label: 'Match System', icon: 'monitor' },
    { id: 'light', label: 'Light', icon: 'sun' },
    { id: 'dark', label: 'Dark', icon: 'moon' },
  ];

  const pageSizes = [100, 300, 500, 1000, 2000];

  function close() {
    app.settingsOpen = null;
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      close();
    }
  }

  let checking = $state(false);
  let checkError = $state('');
  let upToDate = $state(false);

  async function checkForUpdate() {
    checking = true;
    checkError = '';
    const error = await app.checkForUpdate();
    checking = false;
    checkError = error;
    upToDate = !error && !app.update;
  }

  const folderLabel = $derived(
    info?.platform.startsWith('darwin') ? 'Show in Finder' : info?.platform.startsWith('windows') ? 'Show in Explorer' : 'Open folder',
  );

  async function openDataFolder() {
    try {
      await api.openDataFolder();
    } catch (err) {
      app.notify(err);
    }
  }
</script>

<svelte:window {onkeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="backdrop" onclick={e => e.target === e.currentTarget && close()}>
  <div class="window" role="dialog" aria-modal="true" aria-label="Settings">
    <nav>
      <div class="nav-title">Settings</div>
      {#each sections as item (item.id)}
        <button class:on={section === item.id} onclick={() => (app.settingsOpen = item.id)}>
          <Icon name={item.icon} size={14} />{item.label}
        </button>
      {/each}
    </nav>

    <section class="content">
      <button class="close" onclick={close} aria-label="Close settings"><Icon name="x" size={14} /></button>

      {#if section === 'general'}
        <h2>General</h2>

        <div class="field">
          <div class="label">Appearance</div>
          <div class="themes" role="radiogroup" aria-label="Appearance">
            {#each themes as t (t.id)}
              <button role="radio" aria-checked={s.theme === t.id} aria-label={t.label} class="theme" class:on={s.theme === t.id} onclick={() => app.updateSettings({ theme: t.id })}>
                <span class="preview {t.id}">
                  <span class="pv-side"></span>
                  <span class="pv-main"><span></span><span></span><span></span></span>
                </span>
                <span class="theme-label"><Icon name={t.icon} size={13} />{t.label}</span>
              </button>
            {/each}
          </div>
          <p class="hint">Also in View ▸ Theme.</p>
        </div>

        <div class="field row">
          <div>
            <div class="label">Rows per page</div>
            <p class="hint">How many rows a table tab loads at a time.</p>
          </div>
          <div class="narrow">
            <Select value={s.pageSize} options={pageSizes.map(n => ({ value: n, label: n.toLocaleString('en-US') }))} onchange={n => app.updateSettings({ pageSize: n })} aria-label="Rows per page" />
          </div>
        </div>

        <div class="field row">
          <div>
            <div class="label"><Icon name="shield" size={13} />Production safety</div>
            <p class="hint">Ask before running INSERT, UPDATE, DELETE, DDL and other writes on connections tagged Production.</p>
          </div>
          <label class="switch">
            <input type="checkbox" checked={s.confirmProdWrites} onchange={e => app.updateSettings({ confirmProdWrites: e.currentTarget.checked })} />
            <span class="track"></span>
          </label>
        </div>

        <div class="field row">
          <div>
            <div class="label"><Icon name="update" size={13} />Install updates automatically</div>
            <p class="hint">Relay DB checks GitHub for a new version at launch. On, it installs it right away and switches to it the next time you open the app; off, it asks first.</p>
          </div>
          <label class="switch">
            <input type="checkbox" checked={s.autoUpdate} onchange={e => app.updateSettings({ autoUpdate: e.currentTarget.checked })} />
            <span class="track"></span>
          </label>
        </div>
      {:else if section === 'editor'}
        <h2>SQL Editor</h2>

        <div class="field row">
          <div>
            <div class="label">Font size</div>
            <p class="hint">Applies to the SQL editor.</p>
          </div>
          <div class="stepper">
            <button class="btn icon sm" disabled={s.editorFontSize <= 10} onclick={() => app.updateSettings({ editorFontSize: s.editorFontSize - 1 })} aria-label="Smaller">−</button>
            <span class="mono">{s.editorFontSize}px</span>
            <button class="btn icon sm" disabled={s.editorFontSize >= 24} onclick={() => app.updateSettings({ editorFontSize: s.editorFontSize + 1 })} aria-label="Larger">+</button>
          </div>
        </div>
        <pre class="sample" style:font-size="{s.editorFontSize}px"><span class="kw">SELECT</span> id, email <span class="kw">FROM</span> customers
<span class="kw">WHERE</span> created_at &gt; <span class="str">'2026-01-01'</span>
<span class="kw">LIMIT</span> <span class="num">50</span>;</pre>

        <div class="field">
          <div class="label">Shortcuts</div>
          <dl class="keys">
            <dt><span class="kbd">⌘↵</span></dt><dd>Run the statement under the cursor (or the selection)</dd>
            <dt><span class="kbd">⇧⌘↵</span></dt><dd>Run the whole script</dd>
            <dt><span class="kbd">⌘T</span></dt><dd>New query tab</dd>
            <dt><span class="kbd">⌘K</span></dt><dd>Switch connection</dd>
          </dl>
        </div>
      {:else}
        <div class="about">
          <img src={mark} alt="" width="80" height="80" draggable="false" />
          <h2>Relay DB</h2>
          <p class="version">Version {info?.version ?? '—'}</p>
          <div class="update-row">
            {#if app.update?.status === 'available'}
              <span>Version {app.update.info.version} is available.</span>
              <button class="btn sm primary" onclick={() => app.installUpdate()}>Install</button>
            {:else if app.update?.status === 'installing'}
              <span>Installing {app.update.info.version}…</span>
            {:else if app.update?.status === 'installed'}
              <span>Version {app.update.info.version} is installed.</span>
              <button class="btn sm primary" onclick={() => app.restartToUpdate()}>Restart now</button>
            {:else if upToDate}
              <span class="up-to-date"><Icon name="check" size={13} />Relay DB is up to date</span>
            {:else}
              {#if app.update?.status === 'failed'}<span class="update-error" title={app.update.error}>{app.update.error}</span>{:else if checkError}<span class="update-error" title={checkError}>{checkError}</span>{/if}
              <button class="btn sm" onclick={checkForUpdate} disabled={checking}>{checking ? 'Checking…' : 'Check for updates'}</button>
            {/if}
          </div>
          <p class="blurb">
            A native database client for PostgreSQL, MySQL and SQLite. Connections and credentials stay on this
            computer — passwords are kept in the system keychain, and Relay DB has no accounts, no cloud sync
            and no telemetry.
          </p>
          <dl class="details">
            <dt>Version</dt><dd class="mono">{info?.version ?? '—'}</dd>
            <dt>Platform</dt><dd class="mono">{info?.platform ?? '—'}</dd>
            <dt>Runtime</dt><dd class="mono">{info?.goVersion ?? '—'}</dd>
            <dt>Data folder</dt>
            <dd class="data-dir">
              <span class="mono" title={info?.dataDir}>{info?.dataDir ?? '—'}</span>
              <button class="btn sm" onclick={openDataFolder}>{folderLabel}</button>
            </dd>
          </dl>
          <p class="copyright">{info?.copyright ?? ''} · Part of the Relay family of developer tools.</p>
        </div>
      {/if}
    </section>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 70;
    display: grid;
    place-items: center;
    background: rgba(0, 0, 0, 0.42);
    animation: fade 0.12s ease-out;
  }
  .window {
    display: grid;
    grid-template-columns: 188px 1fr;
    width: 720px;
    height: 520px;
    max-width: calc(100vw - 32px);
    max-height: calc(100vh - 48px);
    border-radius: 12px;
    background: var(--bg);
    box-shadow: var(--shadow-modal);
    overflow: hidden;
    animation: pop 0.14s ease-out;
  }
  nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 14px 10px;
    background: var(--surface);
    border-right: 1px solid var(--border-subtle);
  }
  .nav-title { padding: 2px 10px 10px; font-size: 11px; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase; color: var(--text-3); }
  nav button {
    display: flex;
    align-items: center;
    gap: 9px;
    height: 30px;
    padding: 0 10px;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: var(--text-2);
    font-size: 13px;
    text-align: left;
  }
  nav button:hover { background: var(--hover); color: var(--text); }
  nav button.on { background: var(--accent-dim); color: var(--text); }
  nav button.on :global(.icon) { color: var(--accent); }

  .content { position: relative; padding: 22px 28px; overflow-y: auto; }
  .close {
    position: absolute;
    top: 14px;
    right: 14px;
    display: grid;
    place-items: center;
    width: 26px;
    height: 26px;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: var(--text-3);
  }
  .close:hover { background: var(--hover); color: var(--text); }
  h2 { margin: 0 0 18px; font-size: 17px; font-weight: 650; }

  .field { padding: 14px 0; border-top: 1px solid var(--border-subtle); }
  .field.row { display: flex; align-items: center; justify-content: space-between; gap: 24px; }
  .label { display: flex; align-items: center; gap: 6px; font-weight: 550; margin-bottom: 2px; }
  .hint { margin: 2px 0 0; font-size: 12px; color: var(--text-3); max-width: 380px; }
  .narrow { width: 110px; flex: none; }

  .themes { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-top: 10px; }
  .theme {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--text-2);
    text-align: left;
  }
  .theme.on { color: var(--text); }
  .preview {
    display: flex;
    height: 74px;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid var(--border);
    outline: 2px solid transparent;
    outline-offset: 2px;
    transition: outline-color 0.12s;
  }
  .theme.on .preview { outline-color: var(--accent); }
  .theme:hover:not(.on) .preview { outline-color: var(--border); }
  .pv-side { width: 28%; }
  .pv-main { flex: 1; display: flex; flex-direction: column; gap: 6px; padding: 10px; }
  .pv-main span { height: 6px; border-radius: 3px; }
  .pv-main span:nth-child(1) { width: 70%; }
  .pv-main span:nth-child(2) { width: 90%; }
  .pv-main span:nth-child(3) { width: 50%; background: #5865f2 !important; }
  .preview.light { background: #fff; }
  .preview.light .pv-side { background: #f1f1f3; }
  .preview.light .pv-main span { background: #dfdfe4; }
  .preview.dark { background: #111113; }
  .preview.dark .pv-side { background: #19191c; }
  .preview.dark .pv-main span { background: #2d2d32; }
  .preview.system { background: linear-gradient(115deg, #fff 50%, #111113 50%); }
  .preview.system .pv-side { background: #f1f1f3; }
  .preview.system .pv-main span { background: #8a8a93; }
  .theme-label { display: flex; align-items: center; gap: 6px; font-size: 12.5px; }

  .switch { position: relative; flex: none; width: 34px; height: 20px; }
  .switch input { position: absolute; opacity: 0; inset: 0; margin: 0; }
  .track {
    position: absolute;
    inset: 0;
    border-radius: 999px;
    background: var(--active);
    transition: background 0.15s;
    pointer-events: none;
  }
  .track::after {
    content: '';
    position: absolute;
    top: 2px;
    left: 2px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
    transition: transform 0.15s;
  }
  .switch input:checked + .track { background: var(--accent); }
  .switch input:checked + .track::after { transform: translateX(14px); }
  .switch input:focus-visible + .track { outline: 2px solid var(--accent); outline-offset: 2px; }

  .stepper { display: flex; align-items: center; gap: 10px; }
  .stepper .mono { min-width: 36px; text-align: center; }
  .sample {
    margin: 0 0 14px;
    padding: 12px 14px;
    border-radius: 8px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    font-family: var(--font-mono);
    line-height: 1.6;
    white-space: pre;
    overflow-x: auto;
  }
  .sample .kw { color: var(--syn-kw); font-weight: 500; }
  .sample .str { color: var(--syn-str); }
  .sample .num { color: var(--syn-num); }

  .keys { display: grid; grid-template-columns: 64px 1fr; gap: 8px 12px; margin: 10px 0 0; font-size: 12.5px; }
  .keys dt { text-align: right; }
  .keys dd { margin: 0; color: var(--text-2); }

  .about { display: flex; flex-direction: column; align-items: center; text-align: center; padding-top: 8px; }
  .about img { border-radius: 18px; box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18); }
  .about h2 { margin: 14px 0 2px; font-size: 20px; }
  .update-row { display: flex; align-items: center; justify-content: center; gap: 8px; margin: 6px 0 2px; font-size: 12px; color: var(--text-2); min-height: 24px; }
  .up-to-date { display: inline-flex; align-items: center; gap: 6px; color: var(--ok); }
  .update-error { max-width: 320px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--danger); }
  .version { margin: 0; font-size: 12.5px; color: var(--text-3); }
  .blurb { max-width: 430px; margin: 16px 0 18px; color: var(--text-2); line-height: 1.55; }
  .details {
    display: grid;
    grid-template-columns: 96px 1fr;
    gap: 8px 14px;
    width: 100%;
    max-width: 440px;
    margin: 0;
    padding: 14px 16px;
    border-radius: 8px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    text-align: left;
    font-size: 12.5px;
  }
  .details dt { color: var(--text-3); }
  .details dd { margin: 0; min-width: 0; user-select: text; -webkit-user-select: text; }
  .details .mono { font-size: 11.5px; }
  .data-dir { display: flex; align-items: center; gap: 8px; }
  .data-dir .mono { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .copyright { margin: 16px 0 0; font-size: 11.5px; color: var(--text-3); }

  @keyframes fade { from { opacity: 0; } }
  @keyframes pop { from { opacity: 0; transform: translateY(6px) scale(0.985); } }
</style>
