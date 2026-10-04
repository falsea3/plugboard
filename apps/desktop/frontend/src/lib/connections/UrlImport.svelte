<script lang="ts">
  import type { Connection } from '../api/backend';
  import { parseConnectionUrl } from './url';
  import Icon from '../ui/Icon.svelte';

  let { onimport }: { onimport: (parsed: Connection) => void } = $props();

  let url = $state('');
  let error = $state('');

  function importUrl(text = url) {
    error = '';
    if (!text.trim()) return;
    try {
      onimport(parseConnectionUrl(text));
      url = '';
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  function onpaste(e: ClipboardEvent) {
    const text = e.clipboardData?.getData('text') ?? '';
    if (/^\s*(jdbc:)?[a-z0-9+]+:\/\//i.test(text) || /^\s*[A-Z_]+\s*=/.test(text)) {
      e.preventDefault();
      importUrl(text);
    }
  }
</script>

<div class="url-row">
  <Icon name="link" size={14} />
  <input
    class="url-input mono"
    bind:value={url}
    {onpaste}
    onkeydown={e => { if (e.key === 'Enter') { e.preventDefault(); importUrl(); } }}
    placeholder="Paste a URL — postgres://user:pass@host:5432/db"
    spellcheck="false"
    autocomplete="off"
    aria-label="Import from connection URL"
  />
  <button type="button" class="btn sm" onclick={() => importUrl()} disabled={!url.trim()}>Import</button>
</div>
{#if error}<div class="url-error">{error}</div>{/if}

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
</style>
