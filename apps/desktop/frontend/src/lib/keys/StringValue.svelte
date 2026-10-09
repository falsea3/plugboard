<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { Compartment, EditorState, Prec } from '@codemirror/state';
  import { drawSelection, EditorView, highlightActiveLine, keymap, lineNumbers } from '@codemirror/view';
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
  import { syntaxHighlighting } from '@codemirror/language';
  import { json } from '@codemirror/lang-json';
  import { highlight, theme } from '../ui/editorTheme';
  import { looksLikeJSON, reindent } from '../json/text';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let { text, readOnly, note, onsave }: { text: string; readOnly: boolean; note: string; onsave: (text: string) => Promise<boolean> } = $props();

  let host = $state<HTMLDivElement>();
  let view: EditorView | undefined;
  let current = $state(untrack(() => text));
  let saving = $state(false);
  const isJSON = $derived(looksLikeJSON(current));
  const dirty = $derived(current !== text);
  const lang = new Compartment();

  async function save() {
    if (readOnly || !dirty || saving) return true;
    saving = true;
    try {
      await onsave(current);
    } finally {
      saving = false;
    }
    return true;
  }

  function set(next: string) {
    view?.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: next } });
  }

  function format() {
    set(reindent(current, current.includes('\n') ? 0 : 2));
  }

  $effect(() => {
    const t = text;
    untrack(() => {
      if (view && view.state.doc.toString() !== t) set(t);
    });
  });

  $effect(() => {
    view?.dispatch({ effects: lang.reconfigure(isJSON ? json() : []) });
  });

  onMount(() => {
    view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: untrack(() => text),
        extensions: [
          lineNumbers(),
          highlightActiveLine(),
          drawSelection(),
          history(),
          theme,
          syntaxHighlighting(highlight),
          lang.of([]),
          EditorView.lineWrapping,
          EditorState.readOnly.of(untrack(() => readOnly)),
          Prec.highest(keymap.of([{ key: 'Mod-s', run: () => (save(), true) }])),
          keymap.of([...defaultKeymap, ...historyKeymap]),
          EditorView.updateListener.of(u => {
            if (u.docChanged) current = u.state.doc.toString();
          }),
        ],
      }),
    });
    return () => view?.destroy();
  });
</script>

<div class="string">
  <div class="bar">
    {#if note}<span class="note faint">{note}</span>{/if}
    <span style="flex:1"></span>
    {#if isJSON}<button class="btn sm ghost" onclick={format} disabled={readOnly}><Icon name="json" size={12} />{current.includes('\n') ? 'Compact' : 'Format'} JSON</button>{/if}
    {#if !readOnly}
      <button class="btn sm ghost" onclick={() => set(text)} disabled={!dirty || saving}>Revert</button>
      <button class="btn sm primary" onclick={save} disabled={!dirty || saving} title="Save (⌘S)">{#if saving}<Spinner size={11} />{/if}Save</button>
    {/if}
  </div>
  <div class="editor" bind:this={host}></div>
</div>

<style>
  .string { height: 100%; display: flex; flex-direction: column; min-height: 0; }
  .bar { flex: none; display: flex; align-items: center; gap: 6px; height: 34px; padding: 0 10px; border-bottom: 1px solid var(--border-subtle); }
  .note { font-size: 12px; }
  .editor { flex: 1; min-height: 0; overflow: auto; }
  .editor :global(.cm-editor) { height: 100%; }
</style>
