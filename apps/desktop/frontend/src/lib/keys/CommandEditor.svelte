<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { EditorState, Prec } from '@codemirror/state';
  import { drawSelection, EditorView, highlightActiveLine, keymap, lineNumbers, placeholder } from '@codemirror/view';
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
  import { autocompletion, completionKeymap, type CompletionContext } from '@codemirror/autocomplete';
  import { theme } from '../ui/editorTheme';
  import { COMMANDS } from './commands';

  let {
    value,
    editor = $bindable(),
    hasSelection = $bindable(false),
    onchange,
    onrun,
  }: { value: string; editor?: EditorView; hasSelection?: boolean; onchange: (text: string) => void; onrun: (all: boolean) => void } = $props();

  let host = $state<HTMLDivElement>();
  const options = COMMANDS.map(label => ({ label, type: 'keyword' }));

  function complete(ctx: CompletionContext) {
    const line = ctx.state.doc.lineAt(ctx.pos);
    const before = line.text.slice(0, ctx.pos - line.from);
    const word = /^\s*([A-Za-z_]*)$/.exec(before);
    if (!word || (!word[1] && !ctx.explicit)) return null;
    return { from: ctx.pos - word[1].length, options, validFor: /^[A-Za-z_]*$/ };
  }

  onMount(() => {
    let saving: ReturnType<typeof setTimeout> | undefined;
    const save = () => {
      clearTimeout(saving);
      saving = undefined;
      onchange(view.state.doc.toString());
    };
    const view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: untrack(() => value),
        extensions: [
          lineNumbers(),
          highlightActiveLine(),
          drawSelection(),
          history(),
          placeholder('Type Redis commands, one per line — ⌘↵ runs the line under the cursor, ⇧⌘↵ runs everything'),
          autocompletion({ override: [complete], icons: false }),
          Prec.highest(
            keymap.of([
              { key: 'Mod-Enter', run: () => (onrun(false), true) },
              { key: 'Shift-Mod-Enter', run: () => (onrun(true), true) },
            ]),
          ),
          keymap.of([...defaultKeymap, ...historyKeymap, ...completionKeymap]),
          theme,
          EditorView.updateListener.of(u => {
            if (u.docChanged) {
              clearTimeout(saving);
              saving = setTimeout(save, 300);
            }
            if (u.selectionSet) hasSelection = !u.state.selection.main.empty;
          }),
        ],
      }),
    });
    editor = view;
    view.focus();
    return () => {
      if (saving) save();
      view.destroy();
    };
  });
</script>

<div class="editor" bind:this={host}></div>

<style>
  .editor { height: 100%; overflow: hidden; }
  .editor :global(.cm-editor) { height: 100%; }
  .editor :global(.cm-editor.cm-focused) { outline: none; }
</style>
