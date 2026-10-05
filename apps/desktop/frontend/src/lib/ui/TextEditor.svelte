<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { Compartment, EditorState, Prec } from '@codemirror/state';
  import { drawSelection, EditorView, highlightActiveLine, keymap, lineNumbers } from '@codemirror/view';
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
  import { theme } from './editorTheme';
  import Modal from './Modal.svelte';

  let {
    title,
    value,
    isNull = false,
    readOnly,
    onsave,
    onclose,
  }: { title: string; value: string; isNull?: boolean; readOnly: boolean; onsave: (text: string) => void; onclose: () => void } = $props();

  let host = $state<HTMLDivElement>();
  let view: EditorView | undefined;
  let wrap = $state(true);
  let chars = $state(untrack(() => value.length));
  let lines = $state(untrack(() => value.split('\n').length));
  const wrapping = new Compartment();

  function save() {
    if (view && !readOnly) onsave(view.state.doc.toString());
    return true;
  }

  function toggleWrap() {
    wrap = !wrap;
    view?.dispatch({ effects: wrapping.reconfigure(wrap ? EditorView.lineWrapping : []) });
  }

  onMount(() => {
    view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: untrack(() => value),
        extensions: [
          lineNumbers(),
          highlightActiveLine(),
          drawSelection(),
          history(),
          theme,
          wrapping.of(EditorView.lineWrapping),
          EditorState.readOnly.of(untrack(() => readOnly)),
          Prec.highest(keymap.of([{ key: 'Mod-s', run: save }, { key: 'Mod-Enter', run: save }])),
          keymap.of([...defaultKeymap, ...historyKeymap]),
          EditorView.updateListener.of(u => {
            if (!u.docChanged) return;
            chars = u.state.doc.length;
            lines = u.state.doc.lines;
          }),
        ],
      }),
    });
    view.focus();
    return () => view?.destroy();
  });
</script>

<Modal {title} width={760} {onclose}>
  <div class="host" bind:this={host}></div>
  {#snippet footer()}
    <span class="status">{isNull ? 'NULL · ' : ''}{chars.toLocaleString('en-US')} characters · {lines.toLocaleString('en-US')} {lines === 1 ? 'line' : 'lines'}{readOnly ? ' · read-only' : ''}</span>
    <span style="flex:1"></span>
    <button class="btn ghost" onclick={toggleWrap}>{wrap ? 'No wrap' : 'Wrap lines'}</button>
    {#if !readOnly}
      <button class="btn" onclick={onclose}>Cancel</button>
      <button class="btn primary" onclick={save}>Save<span class="kbd on-accent">⌘S</span></button>
    {:else}
      <button class="btn" onclick={onclose}>Close</button>
    {/if}
  {/snippet}
</Modal>

<style>
  .host { height: min(60vh, 520px); border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
  .host :global(.cm-editor) { height: 100%; }
  .host :global(.cm-scroller) { font-family: var(--font-mono); }
  .status { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; color: var(--text-3); }
  .kbd.on-accent { border-color: color-mix(in srgb, var(--on-accent) 35%, transparent); color: color-mix(in srgb, var(--on-accent) 85%, transparent); }
</style>
