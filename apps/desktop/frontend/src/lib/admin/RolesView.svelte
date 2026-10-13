<script lang="ts">
  import { tick, untrack } from 'svelte';
  import { api, type Role, type RoleForm as Form, type RoleSpec } from '../api/backend';
  import { app } from '../app/app.svelte';
  import type { Workspace } from '../app/workspace.svelte';
  import SqlPreview from '../table/SqlPreview.svelte';
  import Icon from '../ui/Icon.svelte';
  import LoadBar from '../ui/LoadBar.svelte';
  import Modal from '../ui/Modal.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import RoleForm from './RoleForm.svelte';
  import { displayName, filterRoles, grantsScript, newRole, roleBadges, sameSpec, specOf } from './roles';

  let { ws }: { ws: Workspace } = $props();

  const emptyForm: Form = { noun: 'role', flags: [], host: false, validUntil: false, connLimit: false, membership: false, rules: '', new: { id: '', name: '', host: '', flags: {}, connLimit: -1, validUntil: '', memberOf: [], grants: [], rules: '', grantable: true, system: false, current: false } };
  let form = $state<Form>(emptyForm);
  let roles = $state<Role[]>([]);
  let loading = $state(false);
  let error = $state('');
  let query = $state('');
  let showSystem = $state(false);
  let selected = $state<string | null>(null);
  let creating = $state(false);
  let original = $state<RoleSpec>(newRole(emptyForm));
  let spec = $state<RoleSpec>(newRole(emptyForm));
  let preview = $state<string[] | null>(null);
  let busy = $state(false);
  let askDrop = $state(false);

  const prod = $derived(ws.connection.env === 'prod');
  const shown = $derived(filterRoles(roles, query, showSystem));
  const changed = $derived(!sameSpec(spec, original));
  const editing = $derived(selected !== null || creating);
  const noun = $derived(form.noun);
  const newTitle = $derived(ws.readOnly ? `Read-only is on — turn it off to create ${noun}s` : `New ${noun}`);
  const current = $derived(roles.find(r => r.id === selected));

  async function load(keep = selected) {
    loading = true;
    error = '';
    try {
      const list = await api.roles(ws.session.sessionId);
      form = list.form;
      roles = list.roles;
      const role = roles.find(r => r.id === keep);
      if (role) select(role);
      else selected = null;
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  function select(role: Role) {
    selected = role.id;
    creating = false;
    original = specOf(role);
    spec = specOf(role);
  }

  function startNew() {
    selected = null;
    creating = true;
    original = newRole(form);
    spec = newRole(form);
    void tick().then(() => document.getElementById('role-name')?.focus());
  }

  function sent(): RoleSpec {
    return { ...spec, connLimit: Number.isFinite(spec.connLimit) ? spec.connLimit : -1 };
  }

  async function review() {
    busy = true;
    try {
      const sql = await api.roleSQL(ws.session.sessionId, selected ?? '', sent());
      if (sql.length === 0) app.notify('Nothing to change.', 'info');
      else preview = sql;
    } catch (err) {
      app.notify(err);
    } finally {
      busy = false;
    }
  }

  async function apply() {
    preview = null;
    busy = true;
    try {
      await load(await api.saveRole(ws.session.sessionId, selected ?? '', sent()));
    } catch (err) {
      app.notify(err);
    } finally {
      busy = false;
    }
  }

  async function drop() {
    askDrop = false;
    if (selected === null) return;
    busy = true;
    try {
      await api.dropRole(ws.session.sessionId, selected);
      selected = null;
      await load(null);
    } catch (err) {
      app.notify(err);
    } finally {
      busy = false;
    }
  }

  untrack(() => load());
</script>

<div class="roles">
  <aside class="list">
    <div class="top">
      <input class="input" bind:value={query} placeholder="Filter {noun}s" spellcheck="false" aria-label="Filter {noun}s" />
      <button class="btn icon sm ghost" onclick={() => load()} disabled={loading} title="Reload" aria-label="Reload roles"><Icon name="refresh" size={13} /></button>
      <button class="btn icon sm ghost" onclick={startNew} disabled={ws.readOnly} title={newTitle} aria-label="New {noun}"><Icon name="plus" size={13} /></button>
    </div>
    <div class="items" role="listbox" aria-label="Roles">
      {#if loading}<LoadBar label="Loading roles" />{/if}
      {#each shown as r (r.id)}
        <button class="item" class:on={r.id === selected} role="option" aria-selected={r.id === selected} onclick={() => select(r)}>
          <span class="name mono">{displayName(r)}</span>
          {#if r.current}<span class="tag you">you</span>{/if}
          {#each roleBadges(r, form) as b (b)}<span class="tag">{b}</span>{/each}
        </button>
      {/each}
    </div>
    <div class="bottom">
      <label class="check"><input type="checkbox" bind:checked={showSystem} />Built-in {noun}s</label>
    </div>
  </aside>

  <section class="detail">
    {#if error}
      <div class="error" role="alert"><Icon name="alert" />{error}</div>
    {:else if editing}
      <h2>{current ? displayName(current) : `New ${noun}`}</h2>
      {#if ws.readOnly}<p class="hint">Read-only is on — turn it off to change {noun}s.</p>{/if}
      <RoleForm
        bind:spec
        {form}
        existing={selected !== null}
        showHost={form.host && (!current || !!current.host)}
        {roles}
        readOnly={ws.readOnly || original.system}
        onquery={() => ws.newQuery(grantsScript(original.grants))}
      />
      <div class="actions">
        {#if selected !== null}
          <button class="btn danger" onclick={() => (askDrop = true)} disabled={ws.readOnly || busy || original.system || original.current}>Drop {noun}</button>
        {/if}
        <span style="flex:1"></span>
        <button class="btn" onclick={() => (selected ? select(roles.find(r => r.name === selected)!) : startNew())} disabled={!changed || busy}>Revert</button>
        <button class="btn primary" onclick={review} disabled={!changed || busy || ws.readOnly || !spec.name.trim()}>
          {#if busy}<Spinner size={12} />{/if}{selected ? 'Save' : `Create ${noun}`}
        </button>
      </div>
    {:else}
      <div class="empty">
        <p>Pick a {noun} on the left, or</p>
        <button class="btn primary" onclick={startNew} disabled={ws.readOnly} title={newTitle}><Icon name="plus" size={13} />New {noun}</button>
        {#if ws.readOnly}<p class="hint">Read-only is on — turn it off to create {noun}s.</p>{/if}
      </div>
    {/if}
  </section>
</div>

{#if preview}
  <SqlPreview
    title={prod ? `Change ${noun}s on Production?` : `${noun[0].toUpperCase()}${noun.slice(1)} changes`}
    sql={preview}
    confirm={prod}
    confirmLabel="Run on Production"
    onapply={apply}
    oncommit={apply}
    onclose={() => (preview = null)}
  >
    {#if prod}<strong>{ws.connection.name}</strong> is tagged Production. {/if}These statements run in one transaction; the password never shows in the preview.
  </SqlPreview>
{/if}

{#if askDrop}
  <Modal title="Drop {noun} {current ? displayName(current) : ''}?" width={420} onclose={() => (askDrop = false)}>
    <p class="text">The server refuses while the {noun} still owns objects — reassign or drop them first.</p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (askDrop = false)}>Cancel</button>
      <button class="btn primary danger-fill" onclick={drop}>Drop {noun}</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .roles { height: 100%; display: flex; min-width: 0; }
  .list { flex: none; width: 260px; display: flex; flex-direction: column; border-right: 1px solid var(--border); background: var(--surface); }
  .top { display: flex; gap: 4px; padding: 8px; }
  .top .input { height: 26px; }
  .items { position: relative; flex: 1; min-height: 0; overflow: auto; padding: 0 6px; }
  .item { display: flex; align-items: center; gap: 5px; width: 100%; height: 28px; padding: 0 8px; border: 0; border-radius: 6px; background: none; color: var(--text-2); text-align: left; }
  .item:hover { background: var(--hover); }
  .item.on { background: var(--accent-dim); color: var(--text); }
  .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; }
  .tag { flex: none; padding: 0 5px; border: 1px solid var(--border); border-radius: 4px; color: var(--text-3); font-size: 10.5px; }
  .tag.you { color: var(--accent); border-color: color-mix(in srgb, var(--accent) 40%, transparent); }
  .bottom { display: flex; align-items: center; gap: 6px; padding: 6px 8px; border-top: 1px solid var(--border); }
  .check { display: flex; align-items: center; gap: 5px; color: var(--text-2); font-size: 12px; }
  .detail { flex: 1; min-width: 0; overflow: auto; padding: 18px 22px; }
  h2 { margin: 0 0 14px; font-size: 15px; font-weight: 600; font-family: var(--font-mono); }
  .hint { margin: -6px 0 12px; color: var(--text-3); font-size: 12px; }
  .actions { display: flex; gap: 8px; max-width: 720px; margin-top: 20px; }
  .empty { display: flex; flex-direction: column; align-items: flex-start; gap: 10px; color: var(--text-3); font-size: 12.5px; }
  .empty p { margin: 0; }
  .empty .hint { margin: 0; }
  .error { display: flex; gap: 8px; padding: 10px 12px; border-radius: var(--radius); color: var(--danger); background: color-mix(in srgb, var(--danger) 10%, transparent); font-size: 12px; user-select: text; -webkit-user-select: text; }
  .text { margin: 0; color: var(--text-2); line-height: 1.5; }
</style>
