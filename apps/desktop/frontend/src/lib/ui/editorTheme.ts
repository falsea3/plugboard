import { EditorView } from '@codemirror/view';
import { HighlightStyle } from '@codemirror/language';
import { tags as t } from '@lezer/highlight';

export const highlight = HighlightStyle.define([
  { tag: t.propertyName, color: 'var(--syn-fn)' },
  { tag: [t.keyword, t.operatorKeyword, t.modifier], color: 'var(--syn-kw)', fontWeight: '500' },
  { tag: [t.string, t.special(t.string)], color: 'var(--syn-str)' },
  { tag: [t.number, t.bool, t.null], color: 'var(--syn-num)' },
  { tag: [t.lineComment, t.blockComment], color: 'var(--syn-comment)', fontStyle: 'italic' },
  { tag: [t.typeName, t.standard(t.name)], color: 'var(--syn-type)' },
  { tag: [t.operator, t.punctuation], color: 'var(--syn-op)' },
  { tag: [t.function(t.variableName), t.special(t.name)], color: 'var(--syn-fn)' },
]);

export const theme = EditorView.theme({
  '&': { height: '100%', fontSize: 'var(--editor-font-size, 13px)', backgroundColor: 'var(--bg)', color: 'var(--text)' },
  '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.6' },
  '.cm-content': { padding: '10px 0', caretColor: 'var(--accent)' },
  '.cm-gutters': { backgroundColor: 'var(--bg)', color: 'var(--text-3)', border: 'none', paddingLeft: '6px' },
  '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--text-2)' },
  '.cm-activeLine': { backgroundColor: 'var(--grid-row-alt)' },
  '.cm-cursor': { borderLeftColor: 'var(--accent)', borderLeftWidth: '2px' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': { backgroundColor: 'var(--grid-selected) !important' },
  '.cm-ran': { boxShadow: 'inset 2px 0 0 var(--accent)' },
  '.cm-lintRange-error': { backgroundImage: 'none', textDecoration: 'underline wavy var(--danger)', textDecorationThickness: '1.5px', textUnderlineOffset: '3px', textDecorationSkipInk: 'none' },
  '.cm-matchingBracket': { backgroundColor: 'var(--accent-dim)', outline: '1px solid var(--accent)' },
  '.cm-placeholder': { color: 'var(--text-3)' },
  '.cm-tooltip': { border: '1px solid var(--border)', backgroundColor: 'var(--elevated)', borderRadius: '6px', overflow: 'hidden' },
  '.cm-diagnostic': { padding: '6px 10px', fontFamily: 'var(--font-ui)', fontSize: '12px' },
  '.cm-diagnostic-error': { borderLeft: '3px solid var(--danger)' },
  '.cm-tooltip-autocomplete > ul': { fontFamily: 'var(--font-mono)', fontSize: '12px' },
  '.cm-tooltip-autocomplete > ul > li[aria-selected]': { backgroundColor: 'var(--accent)', color: 'var(--on-accent)' },
});
