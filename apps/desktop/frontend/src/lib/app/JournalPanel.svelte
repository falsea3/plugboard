<script lang="ts">
  import { journal, type Entry } from './journal.svelte';
  import { copyToClipboard } from '../ui/clipboard';
  import Icon from '../ui/Icon.svelte';

  let list = $state<HTMLDivElement>();

  const time = (d: Date) => d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  const full = (e: Entry) => (e.detail && e.detail !== e.text ? `${e.text}\n${e.detail}` : e.text);

  $effect(() => {
    void journal.entries.length;
    if (journal.open && list) list.scrollTop = list.scrollHeight;
  });
</script>

{#if journal.open}
  <section class="journal" aria-label="Journal">
    <header>
      <span class="title">Journal</span>
      <span class="count">{journal.entries.length}</span>
      <span class="space"></span>
      <button class="btn ghost sm" onclick={() => copyToClipboard(journal.entries.map(e => `${time(e.at)} ${e.level.toUpperCase()} ${full(e)}`).join('\n'))} disabled={!journal.entries.length}><Icon name="copy" size={13} />Copy all</button>
      <button class="btn ghost sm" onclick={() => journal.clear()} disabled={!journal.entries.length}>Clear</button>
      <button class="btn ghost icon sm" onclick={() => (journal.open = false)} title="Close (⇧⌘J)" aria-label="Close the journal"><Icon name="x" size={13} /></button>
    </header>
    <div class="list" bind:this={list}>
      {#each journal.entries as e (e.id)}
        <div class="entry {e.level}">
          <span class="at">{time(e.at)}</span>
          <span class="dot"></span>
          <div class="body">
            <div class="text">{e.text}</div>
            {#if e.detail && e.detail !== e.text}<pre class="detail">{e.detail}</pre>{/if}
          </div>
          <button class="copy" onclick={() => copyToClipboard(full(e))} title="Copy" aria-label="Copy this entry"><Icon name="copy" size={12} /></button>
        </div>
      {:else}
        <p class="empty">Errors and notices from this session show up here.</p>
      {/each}
    </div>
  </section>
{/if}

<style>
  .journal { position: fixed; left: 12px; right: 12px; bottom: 10px; z-index: 80; display: flex; flex-direction: column; height: min(320px, 45vh); border: 1px solid var(--hover); border-radius: var(--radius-lg); background: var(--elevated); box-shadow: var(--shadow-pop); animation: rise 0.14s ease-out; }
  header { flex: none; display: flex; align-items: center; gap: 6px; height: 42px; padding: 0 8px 0 14px; border-bottom: 1px solid var(--grid-line); }
  .title { font-weight: 500; }
  .count { padding: 0 7px; border-radius: 6px; background: var(--hover); color: var(--text-3); font-size: 11.5px; }
  .space { flex: 1; }
  .list { flex: 1; min-height: 0; overflow: auto; padding: 6px 0; }
  .entry { display: flex; align-items: flex-start; gap: 10px; padding: 6px 14px; user-select: text; -webkit-user-select: text; }
  .entry:hover { background: var(--hover); }
  .at { flex: none; color: var(--text-4); font-family: var(--font-mono); font-size: 12px; line-height: 20px; }
  .dot { flex: none; width: 6px; height: 6px; margin-top: 7px; border-radius: 50%; background: var(--ok); }
  .error .dot { background: var(--danger); }
  .body { flex: 1; min-width: 0; }
  .text { line-height: 20px; }
  .error .text { color: var(--danger); }
  .detail { margin: 2px 0 0; color: var(--text-3); font: 12px/1.5 var(--font-mono); white-space: pre-wrap; overflow-wrap: anywhere; }
  .copy { flex: none; display: grid; place-items: center; width: 24px; height: 24px; border: 0; border-radius: 7px; background: transparent; color: var(--text-4); opacity: 0; }
  .entry:hover .copy { opacity: 1; }
  .copy:hover { background: var(--active); color: var(--text); }
  .empty { margin: 0; padding: 20px; color: var(--text-4); text-align: center; }
  @keyframes rise { from { opacity: 0; transform: translateY(8px); } }
</style>
