<script lang="ts">
  import { keys } from '../ui/keys';
  import { tick } from 'svelte';
  import { dismiss } from '../ui/dismiss';
  import { app } from './app.svelte';
  import { engine } from '../engines';
  import type { Connection } from '../api/wire';
  import { filterSections, groupNameProblem } from '../connections/groups';
  import Icon from '../ui/Icon.svelte';
  import PromptModal from '../ui/PromptModal.svelte';
  import SwitcherList from './SwitcherList.svelte';

  let query = $state('');
  let selected = $state(app.active?.connection.id ?? '');
  let list = $state<ReturnType<typeof SwitcherList>>();
  let prompt = $state<{ group: string } | null>(null);
  let promptError = $state('');

  const at = anchor();

  const secs = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return app.groups.sections;
    return filterSections(app.groups.sections, (c: Connection) =>
      [c.name, c.host, c.database, c.file, c.env, c.group, engine(c.driver).name].some(v => v?.toLowerCase().includes(q)));
  });
  const visible = $derived(secs.flatMap(s => (s.collapsed ? [] : s.items)));

  $effect(() => {
    if (!visible.some(c => c.id === selected)) selected = visible[0]?.id ?? '';
  });

  function anchor() {
    const el = [...document.querySelectorAll<HTMLElement>('[data-switch-anchor]')].find(e => e.offsetParent !== null && e.getBoundingClientRect().right > 0);
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { left: r.left, top: r.bottom + 6 };
  }

  function close() {
    app.switcherOpen = false;
  }

  function choose(c: Connection) {
    close();
    app.open(c);
  }

  async function step(by: number) {
    if (visible.length === 0) return;
    const i = visible.findIndex(c => c.id === selected);
    selected = visible[(i + by + visible.length) % visible.length].id;
    await tick();
    list?.scrollTo(selected);
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
    } else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      step(e.key === 'ArrowDown' ? 1 : -1);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const c = visible.find(v => v.id === selected);
      if (c) choose(c);
    } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'n') {
      e.preventDefault();
      close();
      app.editing = null;
    }
  }

  function saveGroup(name: string) {
    const from = prompt?.group ?? '';
    promptError = groupNameProblem(name, app.groups.sections, from);
    if (promptError) return;
    if (from) app.groups.rename(from, name);
    else app.groups.add(name);
    prompt = null;
  }
</script>

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
{:else}
  <div class="backdrop" class:centered={!at} use:dismiss={close}>
    <div class="pop" role="dialog" aria-modal="true" aria-label="Switch connection" style:left={at ? `${at.left}px` : null} style:top={at ? `${at.top}px` : null}>
      <label class="search">
        <Icon name="search" size={14} />
        <!-- svelte-ignore a11y_autofocus -->
        <input bind:value={query} {onkeydown} placeholder="Find a connection" aria-label="Find a connection" spellcheck="false" autofocus />
        <span class="kbd">{keys('⌘K')}</span>
      </label>

      {#if app.connections.length === 0}
        <p class="empty">No saved connections yet.</p>
      {:else if visible.length === 0 && query.trim()}
        <p class="empty">Nothing matches “{query}”.</p>
      {:else}
        <SwitcherList bind:this={list} {secs} {selected} arranging={!query.trim()} onchoose={choose} onselect={id => (selected = id)} onrename={group => (prompt = { group })} />
      {/if}

      <footer>
        <button class="btn ghost sm" onclick={() => { close(); app.editing = null; }}><Icon name="plus" size={14} />Connection</button>
        <button class="btn ghost sm" onclick={() => (prompt = { group: '' })}><Icon name="group" size={14} />Group</button>
        <span class="spacer"></span>
        {#if app.workspaces.length > 1}<span class="hint"><span class="kbd">{keys('⌃Tab')}</span> next open</span>{/if}
        <button class="btn ghost sm" onclick={() => { close(); app.goHome(); }}>All</button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .backdrop { position: fixed; inset: 0; z-index: 60; }
  .backdrop.centered { display: flex; justify-content: center; align-items: flex-start; padding-top: 14vh; background: rgba(0, 0, 0, 0.32); }
  .pop {
    position: absolute;
    width: 380px;
    max-width: calc(100vw - 32px);
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border);
    border-radius: var(--radius-xl);
    background: var(--elevated);
    box-shadow: var(--shadow-pop);
    animation: pop 0.12s ease-out;
  }
  .centered .pop { position: relative; width: 440px; }
  .search { display: flex; align-items: center; gap: 8px; height: 44px; padding: 0 14px; border-bottom: 1px solid var(--border); color: var(--text-3); }
  .search input { flex: 1; min-width: 0; border: 0; outline: none; background: transparent; color: var(--text); font-size: 13.5px; }
  .search input::placeholder { color: var(--text-3); }
  .empty { margin: 0; padding: 24px; text-align: center; color: var(--text-3); }
  footer { display: flex; align-items: center; gap: 2px; padding: 8px; border-top: 1px solid var(--border); }
  footer .btn { gap: 6px; }
  .spacer { flex: 1; }
  .hint { padding-right: 6px; font-size: 11.5px; color: var(--text-4); }
  @keyframes pop { from { opacity: 0; transform: translateY(-4px) scale(0.985); } }
</style>
