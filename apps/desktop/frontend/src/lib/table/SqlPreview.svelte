<script lang="ts">
  import type { Snippet } from 'svelte';
  import { formatCount } from '../ui/format';
  import Modal from '../ui/Modal.svelte';

  let {
    title,
    sql,
    confirm,
    confirmLabel,
    onapply,
    oncommit,
    onclose,
    children,
  }: {
    title: string;
    sql: string[];
    confirm: boolean;
    confirmLabel: string;
    onapply: () => void;
    oncommit: () => void;
    onclose: () => void;
    children: Snippet;
  } = $props();
</script>

<Modal {title} width={620} {onclose}>
  <p class="confirm-text">{@render children()}</p>
  <div class="sql-list" class:prod={confirm}>
    {#each sql as stmt, i (i)}<pre>{stmt};</pre>{/each}
  </div>
  {#snippet footer()}
    <span class="faint" style="font-size:11.5px">{formatCount(sql.length, 'statement')}</span>
    <span style="flex:1"></span>
    {#if confirm}
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={onclose}>Cancel</button>
      <button class="btn primary danger-fill" onclick={onapply}>{confirmLabel}</button>
    {:else}
      <button class="btn" onclick={onclose}>Close</button>
      <button class="btn primary" onclick={oncommit}>Commit</button>
    {/if}
  {/snippet}
</Modal>

<style>
  .confirm-text { margin: 0 0 12px; color: var(--text-2); line-height: 1.5; }
  .confirm-text :global(strong) { color: var(--text); }
  .sql-list { display: flex; flex-direction: column; gap: 6px; max-height: 320px; overflow: auto; }
  .sql-list pre {
    margin: 0;
    padding: 8px 10px;
    border-radius: 6px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    border-left: 3px solid var(--accent);
    font-family: var(--font-mono);
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-word;
    user-select: text;
    -webkit-user-select: text;
  }
  .sql-list.prod pre { border-left-color: var(--env-prod); }
</style>
