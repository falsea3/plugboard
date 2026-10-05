<script lang="ts">
  import { untrack } from 'svelte';
  import Modal from './Modal.svelte';

  let {
    title,
    label,
    value,
    suggestions = [],
    action,
    onsubmit,
    onclose,
  }: {
    title: string;
    label: string;
    value: string;
    suggestions?: string[];
    action: string;
    onsubmit: (value: string) => void;
    onclose: () => void;
  } = $props();

  let text = $state(untrack(() => value));
  const id = `prompt-${Math.random().toString(36).slice(2)}`;

  function select(node: HTMLInputElement) {
    node.focus();
    node.select();
  }

  function submit(e: SubmitEvent) {
    e.preventDefault();
    if (text.trim() && text.trim() !== value) onsubmit(text.trim());
  }
</script>

<Modal {title} width={420} {onclose}>
  <form id="{id}-form" onsubmit={submit}>
    <label class="label" for={id}>{label}</label>
    <input {id} class="input mono" bind:value={text} list={suggestions.length ? `${id}-list` : undefined} spellcheck="false" autocomplete="off" use:select />
    {#if suggestions.length}
      <datalist id="{id}-list">{#each suggestions as s (s)}<option value={s}></option>{/each}</datalist>
    {/if}
    <p class="hint">Opens the statement in a new query tab — look it over and run it there.</p>
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
