<script lang="ts">
  import { tick, untrack } from 'svelte';
  import { api, type Diagram } from '../api/backend';
  import type { DiagramTab, Workspace } from '../app/workspace.svelte';
  import { bounds, layout, makeCards, makeLinks, manyMark, oneMark, relationText, type Card, type Measure, type Point } from './layout';
  import { startDrag } from '../ui/drag';
  import { formatCount } from '../ui/format';
  import Icon from '../ui/Icon.svelte';
  import LoadBar from '../ui/LoadBar.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import DiagramCard from './DiagramCard.svelte';
  import RelationDetail from './RelationDetail.svelte';

  let { ws, tab }: { ws: Workspace; tab: DiagramTab } = $props();

  type Selected = { kind: 'relation'; index: number } | { kind: 'table'; name: string } | null;

  const MIN_K = 0.1;
  const MAX_K = 2;

  let canvas = $state<HTMLDivElement>();
  let width = $state(0);
  let height = $state(0);
  let diagram = $state.raw<Diagram | null>(null);
  let cards = $state.raw(new Map<string, Card>());
  let pos = $state.raw(new Map<string, Point>());
  let view = $state({ x: 0, y: 0, k: 1 });
  let selected = $state<Selected>(null);
  let loading = $state(false);
  let error = $state('');
  let moved = false;

  const links = $derived(diagram ? makeLinks(diagram, cards, pos) : []);
  const gridStep = $derived.by(() => {
    let step = 20 * view.k;
    while (step < 14) step *= 2;
    return step;
  });
  const relation = $derived(selected?.kind === 'relation' ? diagram?.relations[selected.index] ?? null : null);
  const activeLinks = $derived.by(() => {
    if (!selected) return null;
    if (selected.kind === 'relation') return new Set([selected.index]);
    const name = selected.name;
    return new Set(links.filter(l => l.relation.table === name || l.relation.refTable === name).map(l => l.index));
  });
  const activeTables = $derived.by(() => {
    if (!selected) return null;
    if (selected.kind === 'relation') return relation ? new Set([relation.table, relation.refTable]) : null;
    const out = new Set([selected.name]);
    for (const l of links) if (activeLinks?.has(l.index)) out.add(l.relation.table).add(l.relation.refTable);
    return out;
  });

  function makeMeasure(): Measure {
    const ctx = document.createElement('canvas').getContext('2d');
    if (!ctx) return s => s.length * 7.2;
    ctx.font = `12px ${getComputedStyle(document.documentElement).getPropertyValue('--font-mono') || 'monospace'}`;
    return s => ctx.measureText(s).width;
  }

  async function load() {
    loading = true;
    error = '';
    try {
      const d = await api.diagram(ws.session.sessionId, tab.schema);
      const c = makeCards(d, makeMeasure());
      cards = c;
      pos = layout(d, c);
      diagram = d;
      selected = null;
      await tick();
      fit();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    untrack(load);
  });

  const clamp = (k: number) => Math.min(MAX_K, Math.max(MIN_K, k));

  function fit() {
    const b = bounds(cards, pos);
    if (b.w === 0 || width === 0 || height === 0) return;
    const k = clamp(Math.min(1, (width - 80) / b.w, (height - 80) / b.h));
    view = { k, x: (width - b.w * k) / 2 - b.x * k, y: (height - b.h * k) / 2 - b.y * k };
  }

  function zoomAt(cx: number, cy: number, factor: number) {
    const k = clamp(view.k * factor);
    view = { k, x: cx - ((cx - view.x) * k) / view.k, y: cy - ((cy - view.y) * k) / view.k };
  }

  const zoomBy = (factor: number) => zoomAt(width / 2, height / 2, factor);

  $effect(() => {
    const el = canvas;
    if (!el) return;
    const onwheel = (e: WheelEvent) => {
      e.preventDefault();
      if (e.ctrlKey || e.metaKey) {
        const r = el.getBoundingClientRect();
        zoomAt(e.clientX - r.left, e.clientY - r.top, Math.exp(-e.deltaY * 0.01));
      } else {
        view = { ...view, x: view.x - e.deltaX, y: view.y - e.deltaY };
      }
    };
    el.addEventListener('wheel', onwheel, { passive: false });
    return () => el.removeEventListener('wheel', onwheel);
  });

  function startPan(e: PointerEvent) {
    if (e.button !== 0) return;
    moved = false;
    canvas?.focus({ preventScroll: true });
    const v0 = view;
    startDrag(e, 'grabbing', (dx, dy) => {
      if (Math.abs(dx) + Math.abs(dy) > 3) moved = true;
      view = { ...v0, x: v0.x + dx, y: v0.y + dy };
    });
  }

  function dragCard(e: PointerEvent, name: string) {
    if (e.button !== 0) return;
    e.stopPropagation();
    moved = false;
    canvas?.focus({ preventScroll: true });
    const p0 = pos.get(name);
    if (!p0) return;
    const k = view.k;
    startDrag(e, 'grabbing', (dx, dy) => {
      if (Math.abs(dx) + Math.abs(dy) > 3) moved = true;
      pos = new Map(pos).set(name, { x: p0.x + dx / k, y: p0.y + dy / k });
    });
  }

  function open(name: string) {
    const t = diagram?.tables.find(x => x.name === name);
    if (t) ws.openTable({ schema: tab.schema, name, kind: t.kind });
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') selected = null;
    else if (e.key === '+' || e.key === '=') zoomBy(1.2);
    else if (e.key === '-') zoomBy(1 / 1.2);
    else if (e.key === '0') fit();
    else return;
    e.preventDefault();
  }

</script>

<div class="view">
  <div class="toolbar">
    <Icon name="diagram" size={13} />
    <span class="title mono faint">{tab.schema}</span>
    {#if diagram}
      <span class="small faint">{formatCount(diagram.tables.length, 'table')} · {formatCount(links.length, 'relation')}</span>
    {/if}
    <span style="flex:1"></span>
    <button class="btn icon sm ghost" onclick={() => zoomBy(1 / 1.2)} title="Zoom out (−)" aria-label="Zoom out"><Icon name="zoomOut" size={13} /></button>
    <span class="zoom small muted">{Math.round(view.k * 100)}%</span>
    <button class="btn icon sm ghost" onclick={() => zoomBy(1.2)} title="Zoom in (+)" aria-label="Zoom in"><Icon name="zoomIn" size={13} /></button>
    <button class="btn icon sm ghost" onclick={fit} title="Fit to window (0)" aria-label="Fit to window"><Icon name="fit" size={13} /></button>
    <button class="btn icon sm ghost" onclick={load} disabled={loading} title="Reload" aria-label="Reload">
      {#if loading}<Spinner size={12} />{:else}<Icon name="refresh" size={13} />{/if}
    </button>
  </div>

  <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div
    class="canvas"
    bind:this={canvas}
    bind:clientWidth={width}
    bind:clientHeight={height}
    tabindex="0"
    onpointerdown={startPan}
    onclick={() => !moved && (selected = null)}
    {onkeydown}
    style:background-size="{gridStep}px {gridStep}px"
    style:background-position="{view.x}px {view.y}px"
  >
    {#if loading}<LoadBar label="Loading the diagram" />{/if}
    {#if error}
      <div class="note error" role="alert"><Icon name="alert" />{error}</div>
    {:else if !diagram && loading}
      <div class="note faint"><Spinner />Reading tables and foreign keys…</div>
    {:else if diagram && diagram.tables.length === 0}
      <div class="note faint">No tables in {tab.schema}.</div>
    {/if}

    {#if diagram}
      <div class="world" style:transform="translate({view.x}px, {view.y}px) scale({view.k})">
        <svg class="links" width="1" height="1" overflow="visible" aria-label="Relations">
          {#each links as l (l.index)}
            {@const on = activeLinks?.has(l.index) ?? false}
            <g class="link" class:on class:dim={!!activeLinks && !on}>
              <path class="line" d={l.path} />
              <path class="mark" d={manyMark(l.from, l.fromDir)} />
              <path class="mark" d={oneMark(l.to, l.toDir)} />
              <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
              <path
                class="hit"
                d={l.path}
                data-relation="{l.relation.table}.{l.relation.name}"
                onpointerdown={e => e.stopPropagation()}
                onclick={e => {
                  e.stopPropagation();
                  selected = { kind: 'relation', index: l.index };
                }}
              >
                <title>{relationText(l.relation)}</title>
              </path>
            </g>
          {/each}
        </svg>

        {#each diagram.tables as t (t.name)}
          {@const c = cards.get(t.name)}
          {@const p = pos.get(t.name)}
          {#if c && p}
            {@const on = activeTables?.has(t.name) ?? false}
            <DiagramCard
              table={t}
              card={c}
              at={p}
              {on}
              dim={!!activeTables && !on}
              picked={selected?.kind === 'table' && selected.name === t.name}
              {relation}
              ondrag={e => dragCard(e, t.name)}
              onpick={() => !moved && (selected = { kind: 'table', name: t.name })}
              onopen={() => open(t.name)}
            />
          {/if}
        {/each}
      </div>
    {/if}

    {#if relation}
      <RelationDetail {relation} onopen={open} />
    {/if}
  </div>
</div>

<style>
  .view { height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 36px;
    padding: 0 10px 0 14px;
    border-bottom: 1px solid var(--border-subtle);
    color: var(--text-2);
  }
  .title { font-size: 12px; }
  .small { font-size: 12px; }
  .zoom { min-width: 38px; text-align: center; font-variant-numeric: tabular-nums; }

  .canvas {
    position: relative;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    outline: none;
    cursor: grab;
    background-color: var(--bg);
    background-image: radial-gradient(var(--border) 1px, transparent 1px);
    user-select: none;
    -webkit-user-select: none;
  }
  .world { position: absolute; left: 0; top: 0; transform-origin: 0 0; }
  .links { position: absolute; left: 0; top: 0; pointer-events: none; }
  .link .line, .link .mark { fill: none; stroke: var(--text-3); stroke-width: 1.4; }
  .link .hit { fill: none; stroke: transparent; stroke-width: 14; pointer-events: stroke; cursor: pointer; }
  .link:hover .line, .link:hover .mark { stroke: var(--text); }
  .link.on .line, .link.on .mark { stroke: var(--accent); stroke-width: 2.2; }
  .link.dim { opacity: 0.18; }

  .note {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    font-size: 12.5px;
  }
  .note.error { color: var(--danger); user-select: text; -webkit-user-select: text; }

</style>
