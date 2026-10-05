import type { SqlSyntax } from '../api/wire';
import { Blocks, delimiterAt, isWordChar } from './blocks';

export interface StatementRange {
  from: number;
  to: number;
  text: string;
}

export function statementRanges(script: string, syntax: SqlSyntax): StatementRange[] {
  const out: StatementRange[] = [];
  const n = script.length;
  let start = 0;
  let delim = ';';
  let blocks = new Blocks();

  const flush = (end: number) => {
    const raw = script.slice(start, end);
    const lead = raw.length - raw.trimStart().length;
    const text = raw.trim();
    if (text && hasCode(text, syntax)) out.push({ from: start + lead, to: start + lead + text.length, text });
    blocks = new Blocks();
  };
  const word = (i: number) => {
    let j = i;
    while (isWordChar(script[j])) j++;
    blocks.word(script.slice(i, j));
    return j - 1;
  };

  for (let i = 0; i < n; i++) {
    const c = script[i];
    const d = blocks.words === 0 && (c === 'D' || c === 'd') && script.slice(start, i).trim() === '' ? delimiterAt(script, i) : null;
    if (d) {
      delim = d.delim;
      start = d.next;
      i = d.next;
    } else if (delim !== ';' && script.startsWith(delim, i)) {
      flush(i);
      i += delim.length - 1;
      start = i + 1;
    } else if (isWordChar(c) && !isWordChar(script[i - 1])) {
      i = word(i);
    } else if (c === "'" || c === '"' || c === '`') {
      i = skipQuoted(script, i, c, escapesBackslash(script, i, syntax));
    } else if (lineCommentAt(script, i, syntax)) {
      i = skipLine(script, i);
    } else if (c === '/' && script[i + 1] === '*') {
      const end = script.indexOf('*/', i + 2);
      i = end < 0 ? n - 1 : end + 1;
    } else if (c === '$' && syntax.dollarQuotes) {
      const tag = dollarTagAt(script, i);
      if (tag) {
        const end = script.indexOf(tag, i + tag.length);
        i = end < 0 ? n - 1 : end + tag.length - 1;
      }
    } else if (c === ';' && delim === ';' && !blocks.open) {
      flush(i);
      start = i + 1;
    }
  }
  flush(n);
  return out;
}

export function statementAt(script: string, pos: number, syntax: SqlSyntax): StatementRange | null {
  const ranges = statementRanges(script, syntax);
  let best: StatementRange | null = null;
  for (const r of ranges) {
    if (r.from <= pos) best = r;
    if (pos <= r.to + 1) return pos >= r.from ? r : (best ?? r);
  }
  return best;
}

export function lineCommentAt(s: string, i: number, syntax: SqlSyntax): boolean {
  if (s[i] === '#') return syntax.hashComments;
  if (s[i] !== '-' || s[i + 1] !== '-') return false;
  return !syntax.dashCommentNeedsSpace || i + 2 >= s.length || s.charCodeAt(i + 2) <= 32;
}

export function dollarTagAt(s: string, i: number): string | null {
  if (i > 0 && /[\p{L}\p{N}_]/u.test(s[i - 1])) return null;
  return /^\$(?:[A-Za-z_][A-Za-z0-9_]*)?\$/.exec(s.slice(i, i + 64))?.[0] ?? null;
}

function skipQuoted(s: string, i: number, q: string, backslash: boolean): number {
  for (let j = i + 1; j < s.length; j++) {
    if (s[j] === '\\' && backslash) {
      j++;
    } else if (s[j] === q) {
      if (s[j + 1] === q) {
        j++;
        continue;
      }
      return j;
    }
  }
  return s.length - 1;
}

export function escapesBackslash(s: string, i: number, syntax: SqlSyntax): boolean {
  if (s[i] === '`') return false;
  if (syntax.backslashEscapes) return true;
  return syntax.escapeStrings && s[i] === "'" && /[Ee]/.test(s[i - 1] ?? '') && !/\w/.test(s[i - 2] ?? '');
}

function skipLine(s: string, i: number): number {
  const nl = s.indexOf('\n', i);
  return nl < 0 ? s.length - 1 : nl;
}

function hasCode(text: string, syntax: SqlSyntax): boolean {
  let stripped = text.replace(/\/\*[\s\S]*?(\*\/|$)/g, '');
  stripped = stripped.replace(syntax.dashCommentNeedsSpace ? /--(?=[\x00-\x20]|$)[^\n]*/g : /--[^\n]*/g, '');
  if (syntax.hashComments) stripped = stripped.replace(/#[^\n]*/g, '');
  return stripped.trim() !== '';
}
