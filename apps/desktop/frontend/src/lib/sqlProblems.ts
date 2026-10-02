import { statementRanges } from './sqlStatements';
import type { SqlSyntax, SyntaxProblem } from './wire';

export interface Problem {
  from: number;
  to: number;
  message: string;
}

export function utf16Offset(text: string, bytes: number): number {
  let used = 0;
  let i = 0;
  for (const ch of text) {
    if (used >= bytes) return i;
    const cp = ch.codePointAt(0)!;
    used += cp < 0x80 ? 1 : cp < 0x800 ? 2 : cp < 0x10000 ? 3 : 4;
    i += ch.length;
  }
  return text.length;
}

const wordChar = /[\p{L}\p{N}_$]/u;

export function errorRange(text: string, at: number): [number, number] {
  if (at >= text.length) {
    let end = text.length;
    while (end > 0 && /\s/.test(text[end - 1])) end--;
    let start = end;
    while (start > 0 && wordChar.test(text[start - 1])) start--;
    return start < end ? [start, end] : [Math.max(0, end - 1), end];
  }
  let end = at;
  while (end < text.length && wordChar.test(text[end])) end++;
  return [at, Math.max(end, at + 1)];
}

export function serverProblems(script: string, base: number, syntax: SqlSyntax, problems: SyntaxProblem[]): Problem[] {
  const ranges = statementRanges(script, syntax);
  return problems.flatMap(p => {
    const r = ranges[p.index];
    if (!r || p.position < 0) return [];
    const [from, to] = errorRange(r.text, utf16Offset(r.text, p.position));
    return [{ from: base + r.from + from, to: base + r.from + to, message: p.message }];
  });
}
