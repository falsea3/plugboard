<script lang="ts">
  import { dismiss } from '../ui/dismiss';
  import { api, type AppInfo } from '../api/backend';
  import { app, type SettingsSection } from '../app/app.svelte';
  import Icon, { type IconName } from '../ui/Icon.svelte';
  import SettingsGeneral from './SettingsGeneral.svelte';
  import SettingsEditor from './SettingsEditor.svelte';
  import SettingsAbout from './SettingsAbout.svelte';
  import SettingsAgents from './SettingsAgents.svelte';

  let info = $state<AppInfo | null>(null);
  api.appInfo().then(i => (info = i)).catch(() => {});

  const section = $derived(app.settingsOpen ?? 'general');

  const sections: { id: SettingsSection; label: string; icon: IconName }[] = [
    { id: 'general', label: 'General', icon: 'settings' },
    { id: 'editor', label: 'SQL Editor', icon: 'code' },
    { id: 'agents', label: 'AI agents', icon: 'terminal' },
    { id: 'about', label: 'About', icon: 'info' },
  ];

  function close() {
    app.settingsOpen = null;
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      close();
    }
  }
</script>

<svelte:window {onkeydown} />

<div class="backdrop" use:dismiss={close}>
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
        <SettingsGeneral />
      {:else if section === 'editor'}
        <SettingsEditor />
      {:else if section === 'agents'}
        <SettingsAgents />
      {:else}
        <SettingsAbout {info} />
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
    grid-template-columns: 176px 1fr;
    width: 660px;
    min-height: 340px;
    max-width: calc(100vw - 32px);
    max-height: calc(100vh - 48px);
    border-radius: calc(var(--radius-xl) + 2px);
    background: var(--elevated);
    box-shadow: var(--shadow-modal);
    overflow: hidden;
    animation: pop 0.14s ease-out;
  }
  nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 14px 10px;
    background: transparent;
    border-right: 1px solid var(--hover);
  }
  .nav-title { padding: 2px 10px 10px; font-size: 11px; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase; color: var(--text-3); }
  nav button {
    display: flex;
    align-items: center;
    gap: 9px;
    height: 30px;
    padding: 0 10px;
    border: 0;
    border-radius: 8px;
    background: transparent;
    color: var(--text-2);
    font-size: 13px;
    text-align: left;
  }
  nav button:hover { background: var(--hover); color: var(--text); }
  nav button.on { background: var(--tree-selected); color: var(--accent-text); font-weight: 500; }
  nav button.on :global(.icon) { color: var(--accent-text); }

  .content { position: relative; padding: 20px 24px; overflow-y: auto; }
  .close {
    position: absolute;
    top: 14px;
    right: 14px;
    display: grid;
    place-items: center;
    width: 26px;
    height: 26px;
    border: 0;
    border-radius: 8px;
    background: transparent;
    color: var(--text-3);
  }
  .close:hover { background: var(--hover); color: var(--text); }
  .content :global(h2) { margin: 0 0 12px; font-size: 16px; font-weight: 600; }
  .content :global(.field) { padding: 12px 0; border-top: 1px solid var(--hover); }
  .content :global(.field.row) { display: flex; align-items: center; justify-content: space-between; gap: 24px; }
  .content :global(.label) { display: flex; align-items: center; gap: 6px; font-weight: 550; margin-bottom: 2px; }
  .content :global(.hint) { margin: 2px 0 0; font-size: 12px; color: var(--text-3); max-width: 380px; }

  @keyframes fade { from { opacity: 0; } }
  @keyframes pop { from { opacity: 0; transform: translateY(6px) scale(0.985); } }
</style>
