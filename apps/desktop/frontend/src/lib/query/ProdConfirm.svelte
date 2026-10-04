<script lang="ts">
  import Modal from '../ui/Modal.svelte';

  let { name, writes, onrun, onclose }: { name: string; writes: string[]; onrun: () => void; onclose: () => void } = $props();
</script>

<Modal title="Run on Production?" width={500} {onclose}>
  <p class="confirm-text">
    <strong>{name}</strong> is tagged Production. {writes.length === 1 ? 'This statement can change' : `These ${writes.length} statements can change`} data, schema or settings:
  </p>
  <div class="confirm-list">
    {#each writes.slice(0, 4) as w, i (i)}
      <pre>{w.length > 400 ? w.slice(0, 400) + '…' : w}</pre>
    {/each}
    {#if writes.length > 4}<div class="faint">…and {writes.length - 4} more</div>{/if}
  </div>
  {#snippet footer()}
    <span class="faint confirm-hint">Turn this off in Settings ▸ General.</span>
    <span style="flex:1"></span>
    <!-- svelte-ignore a11y_autofocus -->
    <button class="btn" autofocus onclick={onclose}>Cancel</button>
    <button class="btn primary danger-fill" onclick={onrun}>Run on Production</button>
  {/snippet}
</Modal>

<style>
  .confirm-text { margin: 0 0 12px; color: var(--text-2); line-height: 1.5; }
  .confirm-text strong { color: var(--text); }
  .confirm-list { display: flex; flex-direction: column; gap: 6px; max-height: 240px; overflow: auto; }
  .confirm-list pre {
    margin: 0;
    padding: 8px 10px;
    border-radius: 6px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    border-left: 3px solid var(--env-prod);
    font-family: var(--font-mono);
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-word;
    user-select: text;
    -webkit-user-select: text;
  }
  .confirm-hint { font-size: 11.5px; }
</style>
