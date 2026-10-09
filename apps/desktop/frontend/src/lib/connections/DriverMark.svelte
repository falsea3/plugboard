<script lang="ts">
  import { engine } from '../engines';
  import type { Driver } from '../api/wire';

  let { driver, size = 32, round = false }: { driver: Driver; size?: number; round?: boolean } = $props();
  const e = $derived(engine(driver));
</script>

<span
  class="mark"
  class:round
  style:width="{size}px"
  style:height="{size}px"
  style:background={e.background}
  style:--ink={e.ink ?? '#fff'}
  title={e.name}
  aria-label={e.name}
  role="img"
>
  {@html e.logo}
</span>

<style>
  .mark {
    flex: none;
    display: grid;
    place-items: center;
    border-radius: 24%;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.28), inset 0 -1px 0 rgba(0, 0, 0, 0.12);
  }
  .mark.round { border-radius: 50%; }
  .mark :global(svg) {
    width: 62%;
    height: 62%;
    fill: var(--ink);
    filter: drop-shadow(0 1px 1px rgba(0, 0, 0, 0.22));
  }
  .mark :global(title) { display: none; }
</style>
