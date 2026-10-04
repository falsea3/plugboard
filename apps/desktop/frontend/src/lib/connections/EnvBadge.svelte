<script lang="ts">
  import type { Env } from '../api/wire';

  let { env }: { env: Env } = $props();
</script>

{#if env}
  <span class="env {env}">{env}</span>
{/if}

<style>
  .env {
    padding: 0 6px;
    border-radius: 4px;
    font-size: 10px;
    font-weight: 650;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    white-space: nowrap;
    color: var(--c);
    background: color-mix(in srgb, var(--c) 16%, transparent);
    border: 1px solid color-mix(in srgb, var(--c) 34%, transparent);
  }
  .local { --c: var(--env-local); }
  .dev { --c: var(--env-dev); }
  .staging { --c: var(--env-staging); }
  .prod { --c: var(--env-prod); }
</style>
