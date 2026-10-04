<script lang="ts">
  import type { Relation } from '../api/backend';
  import { relationText } from './layout';
  import Icon from '../ui/Icon.svelte';

  let { relation, onopen }: { relation: Relation; onopen: (table: string) => void } = $props();
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="detail" onpointerdown={e => e.stopPropagation()} onclick={e => e.stopPropagation()}>
  <div class="text">
    <code>{relationText(relation)}</code>
    <span class="faint">{relation.name} · on delete {relation.onDelete.toLowerCase()} · on update {relation.onUpdate.toLowerCase()}</span>
  </div>
  <button class="btn sm" onclick={() => onopen(relation.table)}><Icon name="table" size={12} />{relation.table}</button>
  {#if relation.refTable !== relation.table}
    <button class="btn sm" onclick={() => onopen(relation.refTable)}><Icon name="table" size={12} />{relation.refTable}</button>
  {/if}
</div>

<style>
  .detail {
    position: absolute;
    left: 50%;
    bottom: 16px;
    transform: translateX(-50%);
    max-width: calc(100% - 32px);
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 10px 8px 14px;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: var(--elevated);
    box-shadow: var(--shadow-modal);
    cursor: default;
  }
  .text { min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  code { font-family: var(--font-mono); font-size: 12.5px; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .faint { font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
