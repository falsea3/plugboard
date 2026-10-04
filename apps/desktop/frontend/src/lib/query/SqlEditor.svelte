<script lang="ts" module>
  import { EditorState, RangeSetBuilder, StateEffect, StateField } from '@codemirror/state';
  import { Decoration, EditorView, type DecorationSet } from '@codemirror/view';
  import { forEachDiagnostic, setDiagnostics, type Diagnostic } from '@codemirror/lint';
  import type { Problem } from '../sql/problems';

  const setRan = StateEffect.define<{ from: number; to: number }>();
  const ranLine = Decoration.line({ class: 'cm-ran' });

  function ranLines(state: EditorState, from: number, to: number): DecorationSet {
    if (from >= to) return Decoration.none;
    const lines = new RangeSetBuilder<Decoration>();
    for (let pos = from; pos <= to; ) {
      const line = state.doc.lineAt(pos);
      lines.add(line.from, line.from, ranLine);
      pos = line.to + 1;
    }
    return lines.finish();
  }

  const ranStatement = StateField.define<DecorationSet>({
    create: () => Decoration.none,
    update(marks, tr) {
      for (const e of tr.effects) {
        if (e.is(setRan)) return ranLines(tr.state, e.value.from, e.value.to);
      }
      return tr.docChanged || tr.selection ? Decoration.none : marks;
    },
    provide: f => EditorView.decorations.from(f),
  });

  export function showRan(view: EditorView, from: number, to: number) {
    view.dispatch({ effects: setRan.of({ from, to }) });
  }

  export function showProblem(view: EditorView, p: Problem) {
    const list: Diagnostic[] = [];
    forEachDiagnostic(view.state, (d, from, to) => list.push({ ...d, from, to }));
    list.push({ ...p, severity: 'error', source: 'server' });
    view.dispatch(setDiagnostics(view.state, list));
  }
</script>

<script lang="ts">
  import { onMount } from 'svelte';
  import { Compartment, Prec } from '@codemirror/state';
  import { drawSelection, tooltips, highlightActiveLine, highlightActiveLineGutter, keymap, lineNumbers, placeholder } from '@codemirror/view';
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
  import { bracketMatching, indentOnInput, syntaxHighlighting } from '@codemirror/language';
  import { autocompletion, closeBrackets, closeBracketsKeymap, completionKeymap } from '@codemirror/autocomplete';
  import { sql, type SQLDialect, type SQLNamespace } from '@codemirror/lang-sql';
  import { linter, lintGutter } from '@codemirror/lint';
  import { highlight, theme } from '../ui/editorTheme';
  import { lintSql } from '../sql/lint';
  import type { SqlSyntax } from '../api/wire';

  let {
    value = '',
    onchange,
    dialect,
    syntax,
    tables = [],
    defaultSchema = '',
    check,
    onrun,
    editor = $bindable<EditorView | undefined>(),
    hasSelection = $bindable(false),
  }: {
    value?: string;
    onchange?: (text: string) => void;
    dialect: SQLDialect;
    syntax: SqlSyntax;
    tables?: string[];
    defaultSchema?: string;
    check?: (doc: string) => Promise<Problem[]>;
    onrun: (all: boolean) => void;
    editor?: EditorView;
    hasSelection?: boolean;
  } = $props();

  let host: HTMLDivElement;
  const language = new Compartment();

  function languageFor(dialect: SQLDialect, tables: string[], schema: string) {
    const ns: SQLNamespace = {};
    for (const name of tables) ns[name] = [];
    return sql({ dialect, schema: ns, defaultSchema: schema || undefined, upperCaseKeywords: true });
  }

  onMount(() => {
    let saving: ReturnType<typeof setTimeout> | undefined;
    const save = () => {
      clearTimeout(saving);
      saving = undefined;
      onchange?.(view.state.doc.toString());
    };
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
          language.of(languageFor(dialect, tables, defaultSchema)),
          linter(
            async v => {
              const doc = v.state.doc.toString();
              const own: Diagnostic[] = lintSql(doc, syntax).map(p => ({ ...p, severity: 'error' }));
              if (own.length > 0 || !check || !doc.trim()) return own;
              try {
                return (await check(doc)).map(p => ({ ...p, severity: 'error', source: 'server' }));
              } catch {
                return own;
              }
            },
            { delay: 600 },
          ),
          lintGutter(),
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

  $effect(() => {
    const ext = languageFor(dialect, tables, defaultSchema);
    editor?.dispatch({ effects: language.reconfigure(ext) });
  });
</script>

<div class="editor" bind:this={host}></div>

<style>
  .editor { height: 100%; overflow: hidden; }
  .editor :global(.cm-editor.cm-focused) { outline: none; }
</style>
