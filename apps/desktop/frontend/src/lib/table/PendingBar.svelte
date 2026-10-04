<script lang="ts">
  import { formatCount } from '../ui/format';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let {
    count,
    error,
    saving,
    onstop,
    ondiscard,
    onpreview,
    oncommit,
  }: {
    count: number;
    error: string;
    saving: boolean;
    onstop?: () => void;
    ondiscard: () => void;
    onpreview: () => void;
    oncommit: () => void;
  } = $props();
</script>

<div class="footer pending" class:has-error={!!error}>
  <span class="dot"></span>
  <span class="small strong">{formatCount(count, 'change')}</span>
  {#if error}
    <span class="save-error" title={error}><Icon name="alert" size={12} />{error}</span>
  {:else}
    <span class="small faint">not saved</span>
  {/if}
  <span style="flex:1"></span>
  {#if saving && onstop}
    <button class="btn sm" onclick={onstop}><Icon name="stop" size={11} />Stop</button>
  {/if}
  <button class="btn sm ghost" onclick={ondiscard} disabled={saving}>Discard</button>
  <button class="btn sm" onclick={onpreview} disabled={saving}><Icon name="code" size={12} />Preview SQL</button>
  <button class="btn sm primary" onclick={oncommit} disabled={saving}>{#if saving}<Spinner size={11} />Saving…{:else}Commit{/if}<span class="kbd on-accent">⌘S</span></button>
</div>

<style>
  .footer {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 37px;
    min-width: 0;
    padding: 0 10px;
    border-top: 1px solid color-mix(in srgb, var(--warn) 40%, var(--border));
    background: color-mix(in srgb, var(--warn) 9%, var(--surface));
  }
  .footer.has-error {
    background: color-mix(in srgb, var(--danger) 9%, var(--surface));
    border-top-color: color-mix(in srgb, var(--danger) 40%, var(--border));
  }
  .small { font-size: 12px; padding: 0 4px; white-space: nowrap; }
  .strong { color: var(--text); font-weight: 600; }
  .dot { width: 7px; height: 7px; margin-left: 4px; border-radius: 50%; background: var(--warn); }
  .has-error .dot { background: var(--danger); }
  .save-error {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    min-width: 0;
    overflow: hidden;
    color: var(--danger);
    font-size: 12px;
    white-space: nowrap;
    text-overflow: ellipsis;
    user-select: text;
    -webkit-user-select: text;
  }
  .kbd.on-accent { border-color: rgba(255, 255, 255, 0.35); color: rgba(255, 255, 255, 0.85); }
</style>
