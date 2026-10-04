<script lang="ts">
  import type { ThemeMode } from '../api/backend';
  import { app } from '../app/app.svelte';
  import Icon, { type IconName } from '../ui/Icon.svelte';
  import Select from '../ui/Select.svelte';
  import Switch from '../ui/Switch.svelte';

  const s = $derived(app.settings);
  const themes: { id: ThemeMode; label: string; icon: IconName }[] = [
    { id: 'system', label: 'Match System', icon: 'monitor' },
    { id: 'light', label: 'Light', icon: 'sun' },
    { id: 'dark', label: 'Dark', icon: 'moon' },
  ];
  const pageSizes = [100, 300, 500, 1000, 2000];
</script>

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
  <Switch checked={s.confirmProdWrites} onchange={on => app.updateSettings({ confirmProdWrites: on })} label="Production safety" />
</div>

<div class="field row">
  <div>
    <div class="label"><Icon name="update" size={13} />Install updates automatically</div>
    <p class="hint">Plugboard checks GitHub for a new version at launch. On, it installs it right away and switches to it the next time you open the app; off, it asks first.</p>
  </div>
  <Switch checked={s.autoUpdate} onchange={on => app.updateSettings({ autoUpdate: on })} label="Install updates automatically" />
</div>

<style>
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
</style>
