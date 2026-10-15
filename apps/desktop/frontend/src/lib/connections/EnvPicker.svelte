<script lang="ts">
  import type { Env } from '../api/backend';

  let { value = $bindable(), onchange }: { value: Env; onchange?: () => void } = $props();

  const envs: { value: Env; label: string }[] = [
    { value: '', label: 'None' },
    { value: 'local', label: 'Local' },
    { value: 'dev', label: 'Dev' },
    { value: 'staging', label: 'Staging' },
    { value: 'prod', label: 'Production' },
  ];
</script>

<div class="envs" role="radiogroup" aria-label="Environment">
  {#each envs as e (e.value)}
    <button
      type="button"
      role="radio"
      aria-checked={value === e.value}
      class:on={value === e.value}
      style:--tone={e.value ? `var(--env-${e.value})` : 'var(--text-2)'}
      onclick={() => { value = e.value; onchange?.(); }}
    >{e.label}</button>
  {/each}
</div>

<style>
  .envs { display: flex; padding: 3px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--card); }
  button { flex: 1; height: 28px; border: 0; border-radius: var(--radius-sm); background: transparent; color: var(--text-2); font-size: 12.5px; }
  button:hover { color: var(--text); }
  button.on { background: color-mix(in srgb, var(--tone) 16%, var(--elevated)); color: var(--tone); font-weight: 500; }
</style>
