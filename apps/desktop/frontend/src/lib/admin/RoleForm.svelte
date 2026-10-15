<script lang="ts">
  import type { Role, RoleForm, RoleSpec } from '../api/backend';
  import Icon from '../ui/Icon.svelte';
  import Select from '../ui/Select.svelte';
  import ToggleRow from '../ui/ToggleRow.svelte';
  import { displayName } from './roles';

  let {
    spec = $bindable(),
    form,
    existing,
    showHost,
    roles,
    readOnly,
    onquery,
  }: { spec: RoleSpec; form: RoleForm; existing: boolean; showHost: boolean; roles: Role[]; readOnly: boolean; onquery: () => void } = $props();

  let adding = $state(0);
  const label = (id: string) => {
    const r = roles.find(x => x.id === id);
    return r ? displayName(r) : id;
  };
  const addable = $derived(roles.filter(r => r.grantable && r.id !== spec.id && !spec.memberOf.includes(r.id)).map(r => ({ value: r.id, label: displayName(r) })));
</script>

<div class="form">
<fieldset class="form" disabled={readOnly}>
  <div class="form-grid">
    <label for="role-name">Name</label>
    <input id="role-name" class="input mono" bind:value={spec.name} spellcheck="false" autocomplete="off" placeholder="app_user" />

    {#if showHost}
      <label for="role-host">Host</label>
      <div class="form-row">
        <input id="role-host" class="input mono host" bind:value={spec.host} spellcheck="false" autocomplete="off" placeholder="%" />
        <span class="note">where it may connect from — % is anywhere</span>
      </div>
    {/if}

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

    {#if form.connLimit}
      <label for="role-limit">Connections</label>
      <div class="form-row">
        <input id="role-limit" class="input limit" type="number" min="-1" bind:value={spec.connLimit} />
        <span class="note">{spec.connLimit === -1 ? 'no limit' : 'at most at once'}</span>
      </div>
    {/if}

    {#if form.validUntil}
      <label for="role-until">Valid until</label>
      <input id="role-until" class="input mono" bind:value={spec.validUntil} spellcheck="false" placeholder="Forever — or 2027-01-01" />
    {/if}
  </div>

  <div class="attrs">
    {#each form.flags as f (f.key)}
      <ToggleRow checked={!!spec.flags[f.key]} onchange={on => (spec.flags = { ...spec.flags, [f.key]: on })}>
        {#snippet title()}{f.label}{/snippet}
        {f.hint}
      </ToggleRow>
    {/each}
  </div>

  {#if form.rules}
    <label class="label" for="role-rules">Rules</label>
    <textarea id="role-rules" class="input rules mono" bind:value={spec.rules} rows="3" spellcheck="false" placeholder={form.rules}></textarea>
    <p class="note rules-hint">{form.rules}</p>
  {/if}

  {#if form.membership}
  <div class="label">Member of</div>
  <div class="members">
    {#each spec.memberOf as g (g)}
      <span class="chip mono">{label(g)}<button class="btn icon sm ghost" onclick={() => (spec.memberOf = spec.memberOf.filter(x => x !== g))} aria-label="Remove {label(g)}" title="Remove from {label(g)}"><Icon name="x" size={11} /></button></span>
    {/each}
    {#if addable.length}
      {#key adding}
        <Select value="" options={addable} placeholder="Add to a role…" class="add" aria-label="Add to a role" onchange={g => ((spec.memberOf = [...spec.memberOf, g]), adding++)} />
      {/key}
    {:else if spec.memberOf.length === 0}
      <span class="note">No roles to join</span>
    {/if}
  </div>
  {/if}

</fieldset>

  {#if spec.grants.length}
    <div class="label grants-label">
      Privileges
      <button class="btn sm ghost" type="button" onclick={onquery} title="Open these grants as SQL to change them"><Icon name="code" size={12} />Open as SQL</button>
    </div>
    <pre class="grants mono">{spec.grants.join('\n')}</pre>
  {/if}
</div>

<style>
  .form { display: flex; flex-direction: column; gap: 14px; margin: 0; padding: 0; border: 0; min-width: 0; }
  .form-grid { grid-template-columns: 104px 1fr; max-width: 560px; }
  .limit { width: 90px; flex: none; }
  .host { width: 160px; flex: none; }
  .note { color: var(--text-3); font-size: 12px; }
  .check { display: flex; align-items: center; gap: 5px; flex: none; color: var(--text-2); font-size: 12px; }
  .attrs { display: grid; grid-template-columns: repeat(auto-fill, minmax(230px, 1fr)); gap: 4px 18px; }
  .label { color: var(--text-3); font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; }
  .grants-label { display: flex; align-items: center; justify-content: space-between; max-width: 720px; }
  .rules { height: auto; min-height: 64px; max-width: 720px; padding: 8px 12px; margin-top: -8px; resize: vertical; line-height: 1.5; font-size: 12px; }
  .rules-hint { margin: -10px 0 0; }
  .members { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin-top: -6px; }
  .chip { display: inline-flex; align-items: center; gap: 2px; padding-left: 8px; border: 0; border-radius: 8px; background: var(--hover); font-size: 12px; }
  .members :global(.add) { width: 180px; }
  .grants { max-width: 720px; margin: -6px 0 0; padding: 10px 12px; border: 1px solid var(--hover); border-radius: var(--radius); background: var(--well); color: var(--text-2); font-size: 11.5px; line-height: 1.6; white-space: pre-wrap; overflow-wrap: anywhere; user-select: text; -webkit-user-select: text; }
</style>
