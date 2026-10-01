<script lang="ts">
  import { parseNotes } from '../releaseNotes';

  let { markdown }: { markdown: string } = $props();
  const blocks = $derived(parseNotes(markdown));
</script>

<div class="notes">
  {#each blocks as block, i (i)}
    {#snippet spans()}
      {#each block.spans as span, j (j)}
        {#if span.code}<code>{span.text}</code>{:else if span.strong}<strong>{span.text}</strong>{:else}{span.text}{/if}
      {/each}
    {/snippet}
    {#if block.kind === 'heading'}
      <h4>{@render spans()}</h4>
    {:else if block.kind === 'item'}
      <p class="item">{@render spans()}</p>
    {:else}
      <p>{@render spans()}</p>
    {/if}
  {:else}
    <p class="empty">No notes for this release.</p>
  {/each}
</div>

<style>
  .notes { max-height: 360px; overflow: auto; font-size: 12.5px; line-height: 1.55; color: var(--text-2); user-select: text; -webkit-user-select: text; }
  h4 { margin: 12px 0 4px; font-size: 12px; color: var(--text); }
  h4:first-child { margin-top: 0; }
  p { margin: 0 0 6px; }
  .item { position: relative; padding-left: 14px; }
  .item::before { content: '•'; position: absolute; left: 2px; color: var(--text-3); }
  strong { color: var(--text); font-weight: 600; }
  code { font-family: var(--font-mono); font-size: 11.5px; padding: 0 3px; border-radius: 3px; background: var(--surface); }
  .empty { color: var(--text-3); }
</style>
