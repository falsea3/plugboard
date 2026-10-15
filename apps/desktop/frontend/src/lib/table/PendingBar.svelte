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
  <span class="chip danger" title={error}><Icon name="alert" size={11} />{error}</span>
{:else}
  <span class="chip warn">{#if saving}<Spinner size={10} />Saving…{:else}{formatCount(count, 'change')} · ⌘S to save{/if}</span>
{/if}
{#if saving && onstop}<button class="chip link" onclick={onstop}>Stop</button>{/if}
<button class="chip link" onclick={onpreview} disabled={saving}>Preview SQL</button>
<button class="chip link" onclick={ondiscard} disabled={saving}>Discard</button>

<style>
  .chip { flex: none; display: inline-flex; align-items: center; gap: 5px; max-width: 420px; overflow: hidden; padding: 2px 8px; border: 0; border-radius: 6px; background: var(--surface); color: var(--text-3); font-size: 12px; white-space: nowrap; text-overflow: ellipsis; }
  .link { color: var(--accent-text); }
  .link:hover:not(:disabled) { background: var(--hover); }
  .link:disabled { color: var(--text-4); }
  .warn { background: var(--cell-edited); color: var(--warn); }
  .danger { flex-shrink: 1; min-width: 0; background: color-mix(in srgb, var(--danger) 14%, transparent); color: var(--danger); user-select: text; -webkit-user-select: text; }
</style>
