<script lang="ts">
  import type { Snippet } from 'svelte';

  let { tone = 'neutral', children }: { tone?: 'neutral' | 'accent' | 'warn' | 'ok' | 'danger'; children: Snippet } = $props();
</script>

<span class="pill {tone}">{@render children()}</span>

<style>
  .pill { display: inline-flex; align-items: center; gap: 5px; padding: 1px 8px; border-radius: 6px; background: var(--surface); color: var(--text-3); font-size: 12px; white-space: nowrap; }
  .accent { background: var(--accent-bg); color: var(--accent-text); }
  .warn { background: color-mix(in srgb, var(--warn) 14%, transparent); color: var(--warn); }
  .ok { background: color-mix(in srgb, var(--ok) 14%, transparent); color: var(--ok); }
  .danger { background: color-mix(in srgb, var(--danger) 14%, transparent); color: var(--danger); }
</style>
