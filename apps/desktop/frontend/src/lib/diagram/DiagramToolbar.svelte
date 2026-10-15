<script lang="ts">
  import { formatCount } from '../ui/format';
  import Icon from '../ui/Icon.svelte';
  import Select from '../ui/Select.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let {
    schema,
    tables,
    relations,
    zoom,
    loading,
    exporting,
    onzoom,
    onfit,
    onreload,
    onexport,
  }: {
    schema: string;
    tables: number | null;
    relations: number;
    zoom: number;
    loading: boolean;
    exporting: boolean;
    onzoom: (by: number) => void;
    onfit: () => void;
    onreload: () => void;
    onexport: (format: 'png' | 'svg') => void;
  } = $props();

  let picker = $state(0);
  const formats = [
    { value: 'png' as const, label: 'PNG image' },
    { value: 'svg' as const, label: 'SVG image' },
  ];
</script>

<div class="pane-bar">
  <span class="pane-title"><strong>{schema}</strong>{#if tables !== null} · {formatCount(tables, 'table')} · {formatCount(relations, 'relation')}{/if}</span>
  {#if tables}
    <div class="export" class:busy={exporting}>
      {#key picker}
        <Select value={'' as 'png' | 'svg'} options={formats} placeholder="Export" aria-label="Export the diagram" onchange={f => (picker++, onexport(f))} />
      {/key}
    </div>
  {/if}
  <div class="zoom-group">
    <button class="btn ghost icon" onclick={() => onzoom(1 / 1.2)} title="Zoom out (−)" aria-label="Zoom out"><Icon name="zoomOut" size={14} /></button>
    <span class="zoom">{Math.round(zoom * 100)}%</span>
    <button class="btn ghost icon" onclick={() => onzoom(1.2)} title="Zoom in (+)" aria-label="Zoom in"><Icon name="zoomIn" size={14} /></button>
  </div>
  <button class="btn outline icon" onclick={onfit} title="Fit to window (0)" aria-label="Fit to window"><Icon name="fit" size={14} /></button>
  <button class="btn outline icon" onclick={onreload} disabled={loading} title="Reload" aria-label="Reload">
    {#if loading}<Spinner size={12} />{:else}<Icon name="refresh" size={14} />{/if}
  </button>
</div>

<style>
  .zoom-group { display: flex; align-items: center; height: 32px; padding: 0 2px; border: 1px solid var(--border); border-radius: var(--radius); }
  .zoom-group .btn { width: 28px; height: 26px; }
  .zoom { min-width: 42px; color: var(--text-2); font-size: 12px; text-align: center; font-variant-numeric: tabular-nums; }
  .export { width: 120px; }
  .export :global(.select-button) { height: 32px; border-radius: var(--radius); }
  .export.busy { opacity: 0.5; pointer-events: none; }
</style>
