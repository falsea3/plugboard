<script lang="ts">
  import type { RoleSpec } from '../api/backend';
  import Icon from '../ui/Icon.svelte';
  import Select from '../ui/Select.svelte';
  import ToggleRow from '../ui/ToggleRow.svelte';
  import { ATTRIBUTES } from './roles';

  let {
    spec = $bindable(),
    existing,
    others,
    readOnly,
  }: { spec: RoleSpec; existing: boolean; others: string[]; readOnly: boolean } = $props();

  let adding = $state(0);
  const addable = $derived(others.filter(r => r !== spec.name && !spec.memberOf.includes(r)).map(r => ({ value: r, label: r })));
</script>

<fieldset class="form" disabled={readOnly}>
  <div class="form-grid">
    <label for="role-name">Name</label>
    <input id="role-name" class="input mono" bind:value={spec.name} spellcheck="false" autocomplete="off" placeholder="app_user" />

    <label for="role-password">Password</label>
    <div class="form-row">
      <input
        id="role-password"
        class="input"
        type="password"
        bind:value={spec.password}
        disabled={spec.clearPassword}
        autocomplete="new-password"
        placeholder={existing ? 'Unchanged' : 'None'}
      />
      {#if existing}
        <label class="check"><input type="checkbox" bind:checked={spec.clearPassword} onchange={() => (spec.password = '')} />Remove</label>
      {/if}
    </div>

    <label for="role-limit">Connections</label>
    <div class="form-row">
      <input id="role-limit" class="input limit" type="number" min="-1" bind:value={spec.connLimit} />
      <span class="note">{spec.connLimit === -1 ? 'no limit' : 'at most at once'}</span>
    </div>

    <label for="role-until">Valid until</label>
    <input id="role-until" class="input mono" bind:value={spec.validUntil} spellcheck="false" placeholder="Forever — or 2027-01-01" />
  </div>

  <div class="attrs">
    {#each ATTRIBUTES as a (a.key)}
      <ToggleRow bind:checked={spec[a.key]}>
        {#snippet title()}{a.label}{/snippet}
        {a.hint}
      </ToggleRow>
    {/each}
  </div>

  <div class="label">Member of</div>
  <div class="members">
    {#each spec.memberOf as g (g)}
      <span class="chip mono">{g}<button class="btn icon sm ghost" onclick={() => (spec.memberOf = spec.memberOf.filter(x => x !== g))} aria-label="Remove {g}" title="Remove from {g}"><Icon name="x" size={11} /></button></span>
    {/each}
    {#if addable.length}
      {#key adding}
        <Select value="" options={addable} placeholder="Add to a role…" class="add" aria-label="Add to a role" onchange={g => ((spec.memberOf = [...spec.memberOf, g]), adding++)} />
      {/key}
    {/if}
  </div>
</fieldset>

<style>
  .form { display: flex; flex-direction: column; gap: 14px; margin: 0; padding: 0; border: 0; min-width: 0; }
  .form-grid { grid-template-columns: 96px 1fr; max-width: 520px; }
  .limit { width: 90px; flex: none; }
  .note { color: var(--text-3); font-size: 12px; }
  .check { display: flex; align-items: center; gap: 5px; flex: none; color: var(--text-2); font-size: 12px; }
  .attrs { display: grid; grid-template-columns: repeat(auto-fill, minmax(230px, 1fr)); gap: 4px 18px; }
  .label { color: var(--text-2); font-size: 12px; font-weight: 550; }
  .members { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin-top: -6px; }
  .chip { display: inline-flex; align-items: center; gap: 2px; padding-left: 8px; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); font-size: 12px; }
  .members :global(.add) { width: 180px; }
</style>
