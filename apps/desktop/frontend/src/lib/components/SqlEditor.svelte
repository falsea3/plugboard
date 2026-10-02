<script lang="ts" module>
  import { StateEffect, StateField } from '@codemirror/state';
  import { Decoration, EditorView, type DecorationSet } from '@codemirror/view';

  const setRan = StateEffect.define<{ from: number; to: number }>();
  const ranMark = Decoration.mark({ class: 'cm-ran' });

  const ranStatement = StateField.define<DecorationSet>({
    create: () => Decoration.none,
    update(marks, tr) {
      for (const e of tr.effects) {
        if (e.is(setRan)) return e.value.from < e.value.to ? Decoration.set([ranMark.range(e.value.from, e.value.to)]) : Decoration.none;
      }
      return tr.docChanged ? Decoration.none : marks;
    },
    provide: f => EditorView.decorations.from(f),
  });

  export function showRan(view: EditorView, from: number, to: number) {
    view.dispatch({ effects: setRan.of({ from, to }) });
  }
</script>

<script lang="ts">
  import { onMount } from 'svelte';
  import { Compartment, EditorState, Prec } from '@codemirror/state';
  import { drawSelection, tooltips, highlightActiveLine, highlightActiveLineGutter, keymap, lineNumbers, placeholder } from '@codemirror/view';
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
  import { bracketMatching, HighlightStyle, indentOnInput, syntaxHighlighting } from '@codemirror/language';
  import { autocompletion, closeBrackets, closeBracketsKeymap, completionKeymap } from '@codemirror/autocomplete';
  import { MySQL, PostgreSQL, SQLite, sql, type SQLNamespace } from '@codemirror/lang-sql';
  import { linter } from '@codemirror/lint';
  import { tags as t } from '@lezer/highlight';
  import { lintSql } from '../sqlLint';
  import type { Driver } from '../wire';

  let {
    value = $bindable(''),
    driver,
    tables = [],
    defaultSchema = '',
    onrun,
    editor = $bindable<EditorView | undefined>(),
    hasSelection = $bindable(false),
  }: {
    value?: string;
    driver: Driver;
    tables?: string[];
    defaultSchema?: string;
    onrun: (all: boolean) => void;
    editor?: EditorView;
    hasSelection?: boolean;
  } = $props();

  let host: HTMLDivElement;
  const language = new Compartment();

  const dialects = { postgres: PostgreSQL, mysql: MySQL, sqlite: SQLite };

  function languageFor(driver: Driver, tables: string[], schema: string) {
    const ns: SQLNamespace = {};
    for (const name of tables) ns[name] = [];
    return sql({ dialect: dialects[driver], schema: ns, defaultSchema: schema || undefined, upperCaseKeywords: true });
  }

  const highlight = HighlightStyle.define([
    { tag: [t.keyword, t.operatorKeyword, t.modifier], color: 'var(--syn-kw)', fontWeight: '500' },
    { tag: [t.string, t.special(t.string)], color: 'var(--syn-str)' },
    { tag: [t.number, t.bool, t.null], color: 'var(--syn-num)' },
    { tag: [t.lineComment, t.blockComment], color: 'var(--syn-comment)', fontStyle: 'italic' },
    { tag: [t.typeName, t.standard(t.name)], color: 'var(--syn-type)' },
    { tag: [t.operator, t.punctuation], color: 'var(--syn-op)' },
    { tag: [t.function(t.variableName), t.special(t.name)], color: 'var(--syn-fn)' },
  ]);

  const theme = EditorView.theme({
    '&': { height: '100%', fontSize: 'var(--editor-font-size, 13px)', backgroundColor: 'var(--bg)', color: 'var(--text)' },
    '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.6' },
    '.cm-content': { padding: '10px 0', caretColor: 'var(--accent)' },
    '.cm-gutters': { backgroundColor: 'var(--bg)', color: 'var(--text-3)', border: 'none', paddingLeft: '6px' },
    '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--text-2)' },
    '.cm-activeLine': { backgroundColor: 'var(--grid-row-alt)' },
    '.cm-cursor': { borderLeftColor: 'var(--accent)', borderLeftWidth: '2px' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': { backgroundColor: 'var(--grid-selected) !important' },
    '.cm-ran': { backgroundColor: 'var(--grid-selected)' },
    '.cm-matchingBracket': { backgroundColor: 'var(--accent-dim)', outline: '1px solid var(--accent)' },
    '.cm-placeholder': { color: 'var(--text-3)' },
    '.cm-tooltip': { border: '1px solid var(--border)', backgroundColor: 'var(--elevated)', borderRadius: '6px', overflow: 'hidden' },
    '.cm-diagnostic': { padding: '6px 10px', fontFamily: 'var(--font-ui)', fontSize: '12px' },
    '.cm-diagnostic-error': { borderLeft: '3px solid var(--danger)' },
    '.cm-tooltip-autocomplete > ul': { fontFamily: 'var(--font-mono)', fontSize: '12px' },
    '.cm-tooltip-autocomplete > ul > li[aria-selected]': { backgroundColor: 'var(--accent)', color: 'var(--on-accent)' },
  });

  onMount(() => {
    const view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: value,
        extensions: [
          tooltips({ parent: document.body }),
          lineNumbers(),
          highlightActiveLineGutter(),
          highlightActiveLine(),
          drawSelection(),
          ranStatement,
          history(),
          indentOnInput(),
          bracketMatching(),
          closeBrackets(),
          autocompletion({ activateOnTyping: true }),
          syntaxHighlighting(highlight),
          language.of(languageFor(driver, tables, defaultSchema)),
          linter(v => lintSql(v.state.doc.toString(), driver === 'mysql').map(p => ({ ...p, severity: 'error' as const })), { delay: 500 }),
          placeholder('Write SQL…  ⌘↵ runs the statement under the cursor, ⇧⌘↵ runs everything'),
          Prec.highest(
            keymap.of([
              { key: 'Mod-Enter', run: () => (onrun(false), true) },
              { key: 'Shift-Mod-Enter', run: () => (onrun(true), true) },
            ]),
          ),
          keymap.of([...closeBracketsKeymap, ...defaultKeymap, ...historyKeymap, ...completionKeymap, indentWithTab]),
          theme,
          EditorView.updateListener.of(u => {
            if (u.docChanged) value = u.state.doc.toString();
            if (u.selectionSet) hasSelection = !u.state.selection.main.empty;
          }),
        ],
      }),
    });
    editor = view;
    view.focus();
    return () => view.destroy();
  });

  $effect(() => {
    const ext = languageFor(driver, tables, defaultSchema);
    editor?.dispatch({ effects: language.reconfigure(ext) });
  });
</script>

<div class="editor" bind:this={host}></div>

<style>
  .editor { height: 100%; overflow: hidden; }
  .editor :global(.cm-editor.cm-focused) { outline: none; }
</style>
