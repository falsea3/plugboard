import { RangeSetBuilder, StateField, type EditorState, type Extension, type RangeSet } from '@codemirror/state';
import { Decoration, EditorView, GutterMarker, gutterLineClass, type DecorationSet } from '@codemirror/view';
import type { SqlSyntax } from '../api/wire';
import { statementRanges } from '../sql/statements';

export type CursorInfo = { line: number; col: number; statement: number; statements: number };

const MAX_DOC = 200_000;

export function statementOf(script: string, pos: number, syntax: SqlSyntax): { from: number; to: number; index: number; count: number } | null {
  const ranges = statementRanges(script, syntax);
  const i = ranges.findIndex(r => pos >= r.from && pos <= r.to + 1);
  if (i < 0) return ranges.length ? { from: -1, to: -1, index: -1, count: ranges.length } : null;
  return { from: ranges[i].from, to: ranges[i].to, index: i, count: ranges.length };
}

class NumberMark extends GutterMarker {
  elementClass = 'cm-stmt-num';
}
const numberMark = new NumberMark();
const lineMark = Decoration.line({ class: 'cm-stmt' });
const headMark = Decoration.line({ class: 'cm-stmt cm-stmt-head' });

type Marks = { lines: DecorationSet; numbers: RangeSet<GutterMarker>; info: CursorInfo };

function calc(state: EditorState, syntax: SqlSyntax): Marks {
  const doc = state.doc;
  const head = state.selection.main.head;
  const at = doc.lineAt(head);
  const info: CursorInfo = { line: at.number, col: head - at.from + 1, statement: 0, statements: 0 };
  const lines = new RangeSetBuilder<Decoration>();
  const numbers = new RangeSetBuilder<GutterMarker>();
  const stmt = doc.length <= MAX_DOC ? statementOf(doc.toString(), head, syntax) : null;
  if (stmt) {
    info.statements = stmt.count;
    info.statement = stmt.index + 1;
  }
  if (stmt && stmt.index >= 0 && stmt.count > 1 && state.selection.main.empty) {
    const last = doc.lineAt(stmt.to).number;
    let first = doc.lineAt(stmt.from).number;
    while (first < last && /^\s*(--.*)?$/.test(doc.line(first).text)) first++;
    for (let n = first; n <= last; n++) {
      const line = doc.line(n);
      lines.add(line.from, line.from, n === at.number ? headMark : lineMark);
      numbers.add(line.from, line.from, numberMark);
    }
  }
  return { lines: lines.finish(), numbers: numbers.finish(), info };
}

export function currentStatement(syntax: SqlSyntax, oncursor: (info: CursorInfo) => void): Extension {
  const field = StateField.define<Marks>({
    create: s => calc(s, syntax),
    update: (v, tr) => (tr.docChanged || tr.selection ? calc(tr.state, syntax) : v),
    provide: f => [EditorView.decorations.from(f, v => v.lines), gutterLineClass.from(f, v => v.numbers)],
  });
  return [
    field,
    EditorView.updateListener.of(u => {
      const before = u.startState.field(field, false);
      const now = u.state.field(field);
      if (before !== now || u.view.hasFocus) oncursor(now.info);
    }),
  ];
}
