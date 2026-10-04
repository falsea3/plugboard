<script lang="ts">
  import type { Driver } from '../api/backend';
  import { ENGINES } from '../engines';
  import DriverMark from './DriverMark.svelte';

  let { driver, onpick }: { driver: Driver; onpick: (d: Driver) => void } = $props();
</script>

<div class="drivers" role="radiogroup" aria-label="Database">
  {#each ENGINES as e (e.driver)}
    <button type="button" role="radio" aria-checked={driver === e.driver} aria-label={e.name} class="driver" class:selected={driver === e.driver} onclick={() => onpick(e.driver)}>
      <DriverMark driver={e.driver} size={26} />
      <span>{e.name}</span>
    </button>
  {/each}
</div>

<style>
  .drivers {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    margin: 14px 0;
  }
  .driver {
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 8px 10px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    font-weight: 500;
    text-align: left;
  }
  .driver:hover { background: var(--hover); }
  .driver.selected {
    border-color: var(--accent);
    background: var(--accent-dim);
    box-shadow: 0 0 0 1px var(--accent);
  }
</style>
