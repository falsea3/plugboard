<script lang="ts">
  import { onMount } from 'svelte';
  import { EditorState } from '@codemirror/state';
  import { EditorView, lineNumbers } from '@codemirror/view';
  import { syntaxHighlighting } from '@codemirror/language';
  import { sql, type SQLDialect } from '@codemirror/lang-sql';
  import { highlight, theme } from '../ui/editorTheme';

  let { text, dialect }: { text: string; dialect: SQLDialect } = $props();

  let host: HTMLDivElement;
  let view: EditorView | undefined;

  onMount(() => {
    view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: text,
        extensions: [lineNumbers(), syntaxHighlighting(highlight), sql({ dialect }), theme, EditorState.readOnly.of(true), EditorView.lineWrapping],
      }),
    });
    return () => view?.destroy();
  });

  $effect(() => {
    const doc = text;
    if (view && view.state.doc.toString() !== doc) view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: doc } });
  });
</script>

<div class="sql-view" bind:this={host}></div>

<style>
  .sql-view { height: 100%; overflow: hidden; }
  .sql-view :global(.cm-editor.cm-focused) { outline: none; }
</style>
