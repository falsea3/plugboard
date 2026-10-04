<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { EditorState, Prec } from '@codemirror/state';
  import { drawSelection, EditorView, highlightActiveLine, highlightActiveLineGutter, keymap, lineNumbers, tooltips } from '@codemirror/view';
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
  import { bracketMatching, foldGutter, indentOnInput, syntaxHighlighting } from '@codemirror/language';
  import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete';
  import { json } from '@codemirror/lang-json';
  import { linter, lintGutter, type Diagnostic } from '@codemirror/lint';
  import { jsonProblem } from './check';
  import { highlight, theme } from '../ui/editorTheme';
  import { jsonError, reindent } from './text';
  import Modal from '../ui/Modal.svelte';

  let {
    title,
    value,
    readOnly,
    onsave,
    onclose,
  }: { title: string; value: string; readOnly: boolean; onsave: (text: string) => void; onclose: () => void } = $props();

  let host = $state<HTMLDivElement>();
  let view: EditorView | undefined;
  let error = $state('');

  const start = untrack(() => (jsonError(value) === '' ? reindent(value, 2) : value));

  function problems(text: string): Diagnostic[] {
    const p = jsonProblem(text);
    if (!p || (p.from === 0 && p.to === 0)) return [];
    return [{ from: p.from, to: Math.min(p.to, text.length), severity: 'error', message: p.message }];
  }

  function save() {
    if (!view || readOnly) return true;
    const text = view.state.doc.toString();
    if (jsonError(text) !== '') return true;
    onsave(text);
    return true;
  }

  function format() {
    if (!view) return;
    const text = view.state.doc.toString();
    if (jsonError(text) !== '') return;
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: reindent(text, 2) } });
    view.focus();
  }

  onMount(() => {
    view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: start,
        extensions: [
          tooltips({ parent: document.body }),
          lineNumbers(),
          foldGutter(),
          highlightActiveLineGutter(),
          highlightActiveLine(),
          drawSelection(),
          history(),
          indentOnInput(),
          bracketMatching(),
          closeBrackets(),
          json(),
          linter(v => problems(v.state.doc.toString()), { delay: 250 }),
          lintGutter(),
          syntaxHighlighting(highlight),
          theme,
          EditorState.readOnly.of(readOnly),
          EditorView.editable.of(!readOnly),
          Prec.highest(keymap.of([{ key: 'Mod-s', run: save }, { key: 'Mod-Enter', run: save }])),
          keymap.of([...closeBracketsKeymap, ...defaultKeymap, ...historyKeymap, indentWithTab]),
          EditorView.updateListener.of(u => {
            if (u.docChanged) error = jsonError(u.state.doc.toString());
          }),
        ],
      }),
    });
    error = jsonError(start);
    view.focus();
    return () => view?.destroy();
  });
</script>

<Modal {title} width={760} {onclose}>
  <div class="host" bind:this={host}></div>
  {#snippet footer()}
    <span class="status" class:bad={!!error && !readOnly}>{readOnly ? 'Read-only' : error || 'Valid JSON'}</span>
    <span style="flex:1"></span>
    {#if !readOnly}
      <button class="btn" onclick={format} disabled={!!error}>Format</button>
      <button class="btn" onclick={onclose}>Cancel</button>
      <button class="btn primary" onclick={save} disabled={!!error}>Save<span class="kbd on-accent">⌘S</span></button>
    {:else}
      <button class="btn" onclick={onclose}>Close</button>
    {/if}
  {/snippet}
</Modal>

<style>
  .host { height: min(60vh, 520px); border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
  .host :global(.cm-editor) { height: 100%; }
  .status { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; color: var(--text-3); }
  .status.bad { color: var(--danger); }
</style>
