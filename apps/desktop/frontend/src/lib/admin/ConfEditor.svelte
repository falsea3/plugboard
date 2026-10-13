<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { EditorSelection, EditorState, Prec } from '@codemirror/state';
  import { drawSelection, EditorView, highlightActiveLine, highlightActiveLineGutter, keymap, lineNumbers } from '@codemirror/view';
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
  import { syntaxHighlighting } from '@codemirror/language';
  import { lintGutter, setDiagnostics } from '@codemirror/lint';
  import { highlight, theme } from '../ui/editorTheme';
  import { confLanguage, problemRanges } from './conf';
  import type { ConfigProblem } from '../api/backend';

  let {
    value,
    readOnly,
    problems,
    onchange,
    onsave,
  }: { value: string; readOnly: boolean; problems: ConfigProblem[]; onchange: (text: string) => void; onsave: () => void } = $props();

  let host = $state<HTMLDivElement>();
  let view: EditorView | undefined;

  export function goto(line: number) {
    if (!view) return;
    const at = view.state.doc.line(Math.min(Math.max(line, 1), view.state.doc.lines)).from;
    view.dispatch({ selection: EditorSelection.cursor(at), scrollIntoView: true });
    view.focus();
  }

  $effect(() => {
    const list = problems;
    if (!view) return;
    const ranges = problemRanges(view.state.doc.toString(), list);
    view.dispatch(setDiagnostics(view.state, ranges.map(r => ({ ...r, severity: 'error' as const, source: 'server' }))));
  });

  onMount(() => {
    view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: untrack(() => value),
        extensions: [
          lineNumbers(),
          highlightActiveLineGutter(),
          highlightActiveLine(),
          drawSelection(),
          history(),
          theme,
          confLanguage,
          syntaxHighlighting(highlight),
          lintGutter(),
          EditorView.lineWrapping,
          EditorState.readOnly.of(untrack(() => readOnly)),
          Prec.highest(keymap.of([{ key: 'Mod-s', run: () => (onsave(), true) }])),
          keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
          EditorView.updateListener.of(u => {
            if (u.docChanged) onchange(u.state.doc.toString());
          }),
        ],
      }),
    });
    return () => view?.destroy();
  });
</script>

<div class="host" bind:this={host} aria-label="Configuration file"></div>

<style>
  .host { flex: 1; min-height: 0; overflow: hidden; }
  .host :global(.cm-editor) { height: 100%; }
  .host :global(.cm-editor.cm-focused) { outline: none; }
</style>
