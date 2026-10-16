<script lang="ts">
  import type { ImportResult } from '../api/wire';
  import { formatCount } from '../ui/format';
  import Icon from '../ui/Icon.svelte';

  let { result }: { result: ImportResult } = $props();

  const ok = $derived(result.errors.length === 0 && !result.stopped);
</script>

<div class="report">
  <p class="head" class:ok class:bad={!ok}>
    <Icon name={ok ? 'check' : 'alert'} size={16} />
    {#if ok}
      Imported: {formatCount(result.statements, 'statement')}{#if result.rows > 0}, {formatCount(result.rows, 'row')} loaded{/if}.
    {:else if result.rolledBack}
      Nothing was imported — an error rolled everything back.
    {:else if result.stopped}
      Stopped after {formatCount(result.statements, 'statement')}. What ran before stays.
    {:else}
      Done with {formatCount(result.errors.length, 'error')}: {formatCount(result.statements - result.errors.length, 'statement')} ran.
    {/if}
  </p>
  {#if result.errors.length > 0}
    <ul class="errors">
      {#each result.errors as e (e.index)}
        <li>
          <span class="msg">{#if e.index}<span class="faint">#{e.index}</span> {/if}{e.message}</span>
          {#if e.statement}<pre class="mono">{e.statement}</pre>{/if}
        </li>
      {/each}
    </ul>
    <p class="hint faint">Errors are also in the journal (⇧⌘J).</p>
  {/if}
  {#if result.skipped.length > 0}
    <p class="hint faint">Skipped {formatCount(result.skipped.length, 'psql command')} (<code>{result.skipped[0]}</code>{result.skipped.length > 1 ? ', …' : ''}) — they only mean something to psql.</p>
  {/if}
</div>

<style>
  .report { display: flex; flex-direction: column; gap: 10px; }
  .head { display: flex; align-items: flex-start; gap: 8px; margin: 0; color: var(--text); line-height: 1.45; }
  .head.ok :global(.icon) { color: var(--ok); flex: none; margin-top: 2px; }
  .head.bad :global(.icon) { color: var(--danger); flex: none; margin-top: 2px; }
  .errors { max-height: 240px; overflow-y: auto; margin: 0; padding: 0; list-style: none; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface); }
  .errors li { padding: 8px 10px; border-top: 1px solid var(--border-subtle); }
  .errors li:first-child { border-top: 0; }
  .msg { display: block; color: var(--danger); font-size: 12px; line-height: 1.4; user-select: text; -webkit-user-select: text; }
  pre { margin: 6px 0 0; max-height: 80px; overflow: hidden; color: var(--text-3); font-size: 11px; white-space: pre-wrap; word-break: break-all; user-select: text; -webkit-user-select: text; }
  .hint { margin: 0; font-size: 11.5px; line-height: 1.45; }
  .hint code { font-family: var(--font-mono); font-size: 11px; }
</style>
