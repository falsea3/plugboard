<script lang="ts">
  import { untrack } from 'svelte';
  import Modal from '../ui/Modal.svelte';
  import { nameProblem } from './names';

  let {
    title,
    value,
    taken,
    current = '',
    action,
    onsubmit,
    onclose,
  }: {
    title: string;
    value: string;
    taken: string[];
    current?: string;
    action: string;
    onsubmit: (name: string) => Promise<void>;
    onclose: () => void;
  } = $props();

  let name = $state(untrack(() => value));
  let failed = $state('');
  let busy = $state(false);
  const problem = $derived(nameProblem(name, taken, current));

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (problem || busy) return;
    if (current && name === current) return onclose();
    busy = true;
    try {
      await onsubmit(name);
    } catch (err) {
      failed = err instanceof Error ? err.message : String(err);
    } finally {
      busy = false;
    }
  }
</script>

<Modal {title} width={400} {onclose}>
  <form id="script-name-form" onsubmit={submit}>
    <label class="label" for="script-name">Script name</label>
    <!-- svelte-ignore a11y_autofocus -->
    <input id="script-name" class="input" bind:value={name} oninput={() => (failed = '')} autofocus spellcheck="false" autocomplete="off" />
    {#if (problem && name) || failed}<p class="problem" role="alert">{failed || problem}</p>{/if}
    <p class="hint">Scripts are kept with this connection and listed under Scripts in the sidebar.</p>
  </form>
  {#snippet footer()}
    <span style="flex:1"></span>
    <button class="btn" type="button" onclick={onclose}>Cancel</button>
    <button class="btn primary" type="submit" form="script-name-form" disabled={!!problem || busy}>{action}</button>
  {/snippet}
</Modal>

<style>
  .label { display: block; margin-bottom: 6px; color: var(--text-2); font-size: 12px; }
  .input { width: 100%; }
  .problem { margin: 6px 0 0; color: var(--danger); font-size: 11.5px; }
  .hint { margin: 8px 0 0; color: var(--text-3); font-size: 11.5px; }
</style>
