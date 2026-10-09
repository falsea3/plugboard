<script lang="ts">
  import { untrack } from 'svelte';
  import Modal from './Modal.svelte';
  import Autocomplete from './Autocomplete.svelte';

  let {
    title,
    label,
    value,
    suggestions = [],
    action,
    hint = 'Opens the statement in a new query tab — look it over and run it there.',
    onsubmit,
    onclose,
  }: {
    title: string;
    label: string;
    value: string;
    suggestions?: string[];
    action: string;
    hint?: string;
    onsubmit: (value: string) => void;
    onclose: () => void;
  } = $props();

  let text = $state(untrack(() => value));
  const id = `prompt-${Math.random().toString(36).slice(2)}`;

  function submit(e: SubmitEvent) {
    e.preventDefault();
    if (text.trim() && text.trim() !== value) onsubmit(text.trim());
  }
</script>

<Modal {title} width={420} {onclose}>
  <form id="{id}-form" onsubmit={submit}>
    <label class="label" for={id}>{label}</label>
    <Autocomplete {id} bind:value={text} {suggestions} autofocus />
    {#if hint}<p class="hint">{hint}</p>{/if}
  </form>
  {#snippet footer()}
    <span style="flex:1"></span>
    <button class="btn" type="button" onclick={onclose}>Cancel</button>
    <button class="btn primary" type="submit" form="{id}-form" disabled={!text.trim() || text.trim() === value}>{action}</button>
  {/snippet}
</Modal>

<style>
  .label { display: block; margin-bottom: 6px; color: var(--text-2); font-size: 12px; }
  .hint { margin: 8px 0 0; color: var(--text-3); font-size: 11.5px; }
</style>
