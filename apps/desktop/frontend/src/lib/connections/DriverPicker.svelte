<script lang="ts">
  import type { Driver } from '../api/backend';
  import { ENGINES } from '../engines';
  import DriverMark from './DriverMark.svelte';

  let { driver, onpick }: { driver: Driver; onpick: (d: Driver) => void } = $props();
</script>

<div class="drivers" role="radiogroup" aria-label="Database">
  {#each ENGINES as e (e.driver)}
    <button type="button" role="radio" aria-checked={driver === e.driver} aria-label={e.name} class="driver" class:selected={driver === e.driver} onclick={() => onpick(e.driver)}>
      <DriverMark driver={e.driver} size={24} />
      <span>{e.name}</span>
    </button>
  {/each}
</div>

<style>
  .drivers { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 8px; }
  .driver {
    height: 64px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 0 4px;
    border: 1px solid var(--border-subtle);
    border-radius: 12px;
    background: var(--surface);
    color: var(--text-2);
    font-size: 12px;
  }
  .driver span { max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .driver:hover { background: var(--hover); color: var(--text); }
  .driver.selected { border-color: var(--accent); background: var(--accent-bg); color: var(--text); }
  .driver :global(.mark) { border-radius: 7px; }
</style>
