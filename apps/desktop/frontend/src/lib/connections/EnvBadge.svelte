<script lang="ts">
  import type { Env } from '../api/wire';

  let { env }: { env: Env } = $props();
</script>

{#if env}
  <span class="env {env}">{env}</span>
{/if}

<style>
  .env {
    padding: 0 5px;
    border-radius: 4px;
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    white-space: nowrap;
    color: var(--c);
    border: 1px solid color-mix(in srgb, var(--c) 42%, transparent);
  }
  .local { --c: var(--env-local); }
  .dev { --c: var(--env-dev); }
  .staging { --c: var(--env-staging); }
  .prod { --c: var(--env-prod); }
</style>
