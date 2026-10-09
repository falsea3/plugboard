<script lang="ts">
  import Modal from '../ui/Modal.svelte';
  import Select from '../ui/Select.svelte';
  import { addFields, createChange, type Change } from './items';

  let { db, onsubmit, onclose }: { db: string; onsubmit: (name: string, ch: Change, ttl: number) => Promise<boolean>; onclose: () => void } = $props();

  const types = [
    { value: 'string', label: 'String' },
    { value: 'hash', label: 'Hash' },
    { value: 'list', label: 'List' },
    { value: 'set', label: 'Set' },
    { value: 'zset', label: 'Sorted set' },
    { value: 'stream', label: 'Stream' },
  ];
  let name = $state('');
  let type = $state('string');
  let ttl = $state('');
  let values = $state<Record<string, string>>({});
  let busy = $state(false);
  const fields = $derived(type === 'string' ? [{ name: 'value', label: 'Value', placeholder: 'value' }] : addFields(type));
  const ttlProblem = $derived(ttl.trim() && !/^\d+$/.test(ttl.trim()) ? 'TTL is a whole number of seconds.' : '');

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!name.trim() || ttlProblem || busy) return;
    busy = true;
    try {
      await onsubmit(name, createChange(type, values), Number(ttl.trim() || '0'));
    } finally {
      busy = false;
    }
  }
</script>

<Modal title="New key in database {db}" width={460} {onclose}>
  <form id="new-key-form" class="form" onsubmit={submit}>
    <label for="nk-name">Name</label>
    <!-- svelte-ignore a11y_autofocus -->
    <input id="nk-name" class="input mono" bind:value={name} placeholder="user:42" autofocus spellcheck="false" autocomplete="off" />
    <label for="nk-type">Type</label>
    <Select id="nk-type" bind:value={type} options={types} aria-label="Type" />
    {#each fields as f (f.name)}
      <label for="nk-{f.name}">{f.label}</label>
      <input id="nk-{f.name}" class="input" bind:value={values[f.name]} placeholder={f.placeholder} spellcheck="false" autocomplete="off" />
    {/each}
    <label for="nk-ttl">TTL</label>
    <input id="nk-ttl" class="input" bind:value={ttl} placeholder="Optional — seconds until it expires" inputmode="numeric" />
    {#if ttlProblem}<span></span><p class="problem">{ttlProblem}</p>{/if}
  </form>
  {#snippet footer()}
    <span style="flex:1"></span>
    <button class="btn" type="button" onclick={onclose}>Cancel</button>
    <button class="btn primary" type="submit" form="new-key-form" disabled={!name.trim() || !!ttlProblem || busy}>Create</button>
  {/snippet}
</Modal>

<style>
  .form { display: grid; grid-template-columns: 70px 1fr; gap: 8px 10px; align-items: center; }
  label { color: var(--text-2); font-size: 12px; }
  .problem { margin: 0; color: var(--danger); font-size: 11.5px; }
</style>
