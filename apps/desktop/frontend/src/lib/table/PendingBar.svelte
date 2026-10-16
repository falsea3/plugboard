<script lang="ts">
  import { keys } from '../ui/keys';
  import { clip, formatCount } from '../ui/format';
  import Icon from '../ui/Icon.svelte';
  import { journal } from '../app/journal.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let {
    count,
    error,
    saving,
    onstop,
    ondiscard,
    onpreview,
  }: {
    count: number;
    error: string;
    saving: boolean;
    onstop?: () => void;
    ondiscard: () => void;
    onpreview: () => void;
  } = $props();
</script>

{#if error}
  <button class="chip danger" title="Show the whole error in the journal (⇧⌘J)" onclick={() => journal.show()}><Icon name="alert" size={11} /><span class="msg">{clip(error, 48)}</span><span class="open"><Icon name="rows" size={12} /></span></button>
{:else}
  <span class="chip warn">{#if saving}<Spinner size={10} />Saving…{:else}{formatCount(count, 'change')} · {keys('⌘S')} to save{/if}</span>
{/if}
{#if saving && onstop}<button class="btn sm outline act" onclick={onstop}><Icon name="stop" size={11} />Stop</button>{/if}
<button class="btn sm outline act" onclick={onpreview} disabled={saving}><Icon name="code" size={13} />Preview SQL</button>
<button class="btn sm outline act" onclick={ondiscard} disabled={saving}><Icon name="restore" size={13} />Discard</button>

<style>
  .chip { flex: 0 1 auto; min-width: 0; display: inline-flex; align-items: center; gap: 5px; padding: 2px 8px; border: 0; border-radius: 6px; background: var(--surface); color: var(--text-3); font-size: 12px; white-space: nowrap; }
  .chip .msg { overflow: hidden; text-overflow: ellipsis; }
  .chip :global(.icon) { flex: none; }
  .chip .open { display: inline-flex; margin-left: 3px; padding-left: 6px; border-left: 1px solid color-mix(in srgb, currentColor 30%, transparent); opacity: 0.8; }
  .chip:hover .open { opacity: 1; }
  .act { flex: none; height: 26px; gap: 6px; padding: 0 10px; border-radius: 8px; font-size: 12px; }
  .warn { background: var(--cell-edited); color: var(--warn); }
  .danger { flex-shrink: 1; min-width: 0; cursor: pointer; background: color-mix(in srgb, var(--danger) 14%, transparent); color: var(--danger); user-select: text; -webkit-user-select: text; }
</style>
