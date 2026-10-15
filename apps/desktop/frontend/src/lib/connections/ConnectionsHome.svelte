<script lang="ts">
  import { tick } from 'svelte';
  import { app } from '../app/app.svelte';
  import type { Connection } from '../api/wire';
  import { filterSections, groupNameProblem } from './groups';
  import { inScope, matches, nextId, type Scope } from './home';
  import HomeNav from './HomeNav.svelte';
  import HomeList from './HomeList.svelte';
  import Passport from './Passport.svelte';
  import Button from '../ui/Button.svelte';
  import TextField from '../ui/TextField.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import Modal from '../ui/Modal.svelte';
  import PromptModal from '../ui/PromptModal.svelte';

  let query = $state('');
  let scope = $state<Scope>({ kind: 'all' });
  let selected = $state(app.workspaces.find(w => w.id === app.activeId)?.connection.id ?? '');
  let deleting = $state<Connection | null>(null);
  let prompt = $state<{ group: string } | null>(null);
  let promptError = $state('');
  let list = $state<ReturnType<typeof HomeList>>();

  const arranging = $derived(!query.trim() && scope.kind === 'all');
  const secs = $derived(arranging ? app.groups.sections : filterSections(app.groups.sections, c => inScope(scope, c) && matches(query, c)));
  const visible = $derived(secs.flatMap(s => (s.collapsed ? [] : s.items)));
  const current = $derived(app.connections.find(c => c.id === selected));

  $effect(() => {
    if (!visible.some(c => c.id === selected)) selected = visible[0]?.id ?? '';
  });

  async function onkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      selected = nextId(visible.map(c => c.id), selected, e.key === 'ArrowDown' ? 1 : -1);
      await tick();
      list?.scrollTo(selected);
    } else if (e.key === 'Enter' && current) {
      e.preventDefault();
      app.open(current);
    }
  }

  function saveGroup(name: string) {
    const from = prompt?.group ?? '';
    promptError = groupNameProblem(name, app.groups.sections, from);
    if (promptError) return;
    if (from) app.groups.rename(from, name);
    else app.groups.add(name);
    if (from && scope.kind === 'group' && scope.name === from) scope = { kind: 'group', name: name.trim() };
    prompt = null;
  }
</script>

<div class="home">
  <HomeNav bind:scope onnewgroup={() => (prompt = { group: '' })} />

  <section class="main">
    <div class="toolbar drag">
      <div class="search no-drag">
        <TextField icon="search" placeholder="Name, host, database…" label="Search connections" bind:value={query} {onkeydown} spellcheck="false" />
      </div>
      <span class="no-drag"><Button variant="primary" kbd="⌘N" onclick={() => (app.editing = null)}>New connection</Button></span>
    </div>

    <div class="body">
      {#if !app.connectionsLoaded}
        <div class="empty"><Spinner size={12} label="Loading connections" /></div>
      {:else if app.connections.length === 0}
        <div class="empty">
          <Icon name="database" size={28} />
          <span>No connections yet</span>
          <Button onclick={() => (app.editing = null)}>Create the first one</Button>
        </div>
      {:else}
        {#if visible.length === 0 && !arranging}
          <div class="empty"><span>No connections match{query.trim() ? ` “${query.trim()}”` : ' this filter'}.</span></div>
        {:else}
          <HomeList bind:this={list} {secs} bind:selected {arranging} {onkeydown} onrename={group => (prompt = { group })} />
        {/if}
        {#if current}<Passport connection={current} ondelete={() => (deleting = current)} />{/if}
      {/if}
    </div>
  </section>
</div>

{#if deleting}
  {@const target = deleting}
  <Modal title="Delete connection?" width={400} onclose={() => (deleting = null)}>
    <p class="note">“{target.name}” and its saved password will be removed. The database itself is not touched.</p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <button class="btn" onclick={() => (deleting = null)}>Cancel</button>
      <button class="btn primary danger-fill" onclick={() => { app.deleteConnection(target.id); deleting = null; }}>Delete</button>
    {/snippet}
  </Modal>
{/if}

{#if prompt}
  <PromptModal
    title={prompt.group ? 'Rename group' : 'New group'}
    label={promptError || 'Group name'}
    value={prompt.group}
    action={prompt.group ? 'Rename' : 'Create'}
    hint="Drag connections into the group, or pick it in the connection form."
    onsubmit={saveGroup}
    onclose={() => { prompt = null; promptError = ''; }}
  />
{/if}

<style>
  .home { height: 100%; display: flex; background: var(--bg); }
  .main { flex: 1; min-width: 0; display: flex; flex-direction: column; padding: 8px 12px 10px 16px; }
  .toolbar { height: 52px; flex: none; display: flex; align-items: center; gap: 8px; }
  .search { flex: 1; min-width: 0; }
  .search :global(.field) { width: 100%; height: 36px; border-radius: 11px; padding: 0 12px; gap: 10px; }
  .toolbar :global(.btn.primary) { height: 36px; border-radius: 11px; padding: 0 14px; }
  .body { flex: 1; min-height: 0; display: flex; gap: 10px; }
  .empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--card); color: var(--text-3); }
  .note { margin: 0; color: var(--text-2); line-height: 1.5; }
</style>
