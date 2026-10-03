<script lang="ts">
  import { tick, untrack } from 'svelte';
  import { api, type Diagram } from '../backend';
  import type { DiagramTab, Workspace } from '../stores/app.svelte';
  import { bounds, layout, makeCards, makeLinks, manyMark, oneMark, type Card, type Measure, type Point } from '../diagramLayout';
  import { startDrag } from '../drag';
  import { formatCount } from '../format';
  import Icon from './Icon.svelte';
  import LoadBar from './LoadBar.svelte';
  import Spinner from './Spinner.svelte';

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
  const marked = (table: string, column: string) =>
    !!relation &&
    ((relation.table === table && relation.columns.includes(column)) || (relation.refTable === table && relation.refColumns.includes(column)));

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

  const columnList = (cols: string[]) => (cols.length === 1 ? cols[0] : `(${cols.join(', ')})`);
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
    style:background-size="{20 * view.k}px {20 * view.k}px"
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
                <title>{l.relation.table}.{columnList(l.relation.columns)} → {l.relation.refTable}.{columnList(l.relation.refColumns)}</title>
              </path>
            </g>
          {/each}
        </svg>

        {#each diagram.tables as t (t.name)}
          {@const c = cards.get(t.name)}
          {@const p = pos.get(t.name)}
          {#if c && p}
            {@const on = activeTables?.has(t.name) ?? false}
            <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
            <div
              class="card"
              class:on
              class:dim={!!activeTables && !on}
              class:picked={selected?.kind === 'table' && selected.name === t.name}
              data-table={t.name}
              style:left="{p.x}px"
              style:top="{p.y}px"
              style:width="{c.w}px"
              style:height="{c.h}px"
              title="Double-click to open {t.name}"
              onpointerdown={e => dragCard(e, t.name)}
              onclick={e => {
                e.stopPropagation();
                if (!moved) selected = { kind: 'table', name: t.name };
              }}
              ondblclick={() => open(t.name)}
            >
              <div class="head">
                <Icon name={t.kind === 'view' ? 'view' : 'table'} size={12} />
                <span class="name">{t.name}</span>
              </div>
              {#each c.columns as col (col.name)}
                <div class="row" class:marked={marked(t.name, col.name)}>
                  <span class="key" class:pk={col.primaryKey}>
                    {#if col.primaryKey}<Icon name="key" size={10} />{:else if c.foreign.has(col.name)}<Icon name="link" size={10} />{/if}
                  </span>
                  <span class="col">{col.name}</span>
                  <span class="type">{col.type}</span>
                </div>
              {/each}
              {#if c.hidden > 0}<div class="row more">+{c.hidden} more</div>{/if}
            </div>
          {/if}
        {/each}
      </div>
    {/if}

    {#if relation}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div class="detail" onpointerdown={e => e.stopPropagation()} onclick={e => e.stopPropagation()}>
        <div class="detail-text">
          <code>{relation.table}.{columnList(relation.columns)} → {relation.refTable}.{columnList(relation.refColumns)}</code>
          <span class="small faint">{relation.name} · on delete {relation.onDelete.toLowerCase()} · on update {relation.onUpdate.toLowerCase()}</span>
        </div>
        <button class="btn sm" onclick={() => open(relation!.table)}><Icon name="table" size={12} />{relation.table}</button>
        {#if relation.refTable !== relation.table}
          <button class="btn sm" onclick={() => open(relation!.refTable)}><Icon name="table" size={12} />{relation.refTable}</button>
        {/if}
      </div>
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

  .card {
    position: absolute;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--surface);
    overflow: hidden;
    cursor: default;
    transition: opacity 0.12s;
  }
  .card.on { border-color: color-mix(in srgb, var(--accent) 55%, var(--border)); }
  .card.picked { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent); }
  .card.dim { opacity: 0.35; }
  .head {
    flex: none;
    display: flex;
    align-items: center;
    gap: 7px;
    height: 32px;
    padding: 0 10px;
    border-bottom: 1px solid var(--border);
    background: var(--elevated);
    color: var(--text);
    font-family: var(--font-ui);
    font-size: 12.5px;
    font-weight: 600;
    cursor: grab;
  }
  .head :global(.icon) { color: var(--text-3); }
  .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .row {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 22px;
    padding: 0 10px 0 6px;
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--text);
  }
  .row.marked { background: var(--accent-dim); }
  .row.more { padding-left: 26px; color: var(--text-3); font-style: italic; }
  .key { flex: none; width: 14px; display: flex; justify-content: center; color: var(--text-3); }
  .key.pk { color: var(--text-2); }
  .col { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .type { flex: none; max-width: 45%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-3); font-size: 11px; }

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
  .detail-text { min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  .detail code { font-family: var(--font-mono); font-size: 12.5px; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .detail .small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
