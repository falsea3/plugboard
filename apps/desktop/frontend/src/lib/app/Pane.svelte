<script lang="ts">
  import type { Snippet } from 'svelte';
  import { setPaneVisible } from './pane';

  let { visible, children }: { visible: boolean; children: Snippet } = $props();
  setPaneVisible(() => visible);
</script>

{@render children()}
