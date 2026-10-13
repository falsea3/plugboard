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

<div class="toolbar">
  <Icon name="diagram" size={13} />
  <span class="title mono faint">{schema}</span>
  {#if tables !== null}
    <span class="small faint">{formatCount(tables, 'table')} · {formatCount(relations, 'relation')}</span>
  {/if}
  <span style="flex:1"></span>
  {#if tables}
    <div class="export" class:busy={exporting}>
      {#key picker}
        <Select value={'' as 'png' | 'svg'} options={formats} placeholder="Export" aria-label="Export the diagram" onchange={f => (picker++, onexport(f))} />
      {/key}
    </div>
  {/if}
  <button class="btn icon sm ghost" onclick={() => onzoom(1 / 1.2)} title="Zoom out (−)" aria-label="Zoom out"><Icon name="zoomOut" size={13} /></button>
  <span class="zoom small muted">{Math.round(zoom * 100)}%</span>
  <button class="btn icon sm ghost" onclick={() => onzoom(1.2)} title="Zoom in (+)" aria-label="Zoom in"><Icon name="zoomIn" size={13} /></button>
  <button class="btn icon sm ghost" onclick={onfit} title="Fit to window (0)" aria-label="Fit to window"><Icon name="fit" size={13} /></button>
  <button class="btn icon sm ghost" onclick={onreload} disabled={loading} title="Reload" aria-label="Reload">
    {#if loading}<Spinner size={12} />{:else}<Icon name="refresh" size={13} />{/if}
  </button>
</div>

<style>
  .toolbar { flex: none; display: flex; align-items: center; gap: 8px; height: 36px; padding: 0 10px 0 14px; border-bottom: 1px solid var(--border-subtle); color: var(--text-2); }
  .title { font-size: 12px; }
  .small { font-size: 12px; }
  .zoom { min-width: 38px; text-align: center; font-variant-numeric: tabular-nums; }
  .export { width: 110px; height: 24px; }
  .export :global(.select-button) { height: 24px; font-size: 12px; }
  .export.busy { opacity: 0.5; pointer-events: none; }
</style>
