import { StreamLanguage, type StringStream } from '@codemirror/language';
import type { ConfigProblem } from '../api/backend';

type LineState = { start: boolean };

function token(stream: StringStream, state: LineState): string | null {
  if (stream.sol()) {
    state.start = true;
  }
  if (stream.eatSpace()) return null;
  if (stream.peek() === '#') {
    stream.skipToEnd();
    return 'comment';
  }
  const quote = stream.peek();
  if (quote === "'" || quote === '"') {
    stream.next();
    while (!stream.eol()) {
      const ch = stream.next();
      if (ch === '\\') stream.next();
      else if (ch === quote) {
        if (stream.peek() === quote) stream.next();
        else break;
      }
    }
    return 'string';
  }
  if (stream.match(/^-?\d+(\.\d+)?[a-zA-Z]*/)) return state.start ? 'propertyName' : 'number';
  if (stream.match(/^[=,]/)) return 'operator';
  if (stream.match(/^[^\s#'"=,]+/)) {
    const word = stream.current();
    const first = state.start;
    state.start = false;
    if (first) return 'propertyName';
    return /^(on|off|true|false|yes|no|all)$/i.test(word) ? 'bool' : null;
  }
  stream.next();
  return null;
}

export const confLanguage = StreamLanguage.define<LineState>({
  name: 'conf',
  startState: () => ({ start: true }),
  token,
  languageData: { commentTokens: { line: '#' } },
});

export type LineProblem = { from: number; to: number; message: string };

export function problemRanges(text: string, problems: ConfigProblem[]): LineProblem[] {
  const starts = [0];
  for (let i = 0; i < text.length; i++) if (text[i] === '\n') starts.push(i + 1);
  return problems.map(p => {
    const n = Math.min(Math.max(p.line, 1), starts.length) - 1;
    const from = starts[n];
    const end = n + 1 < starts.length ? starts[n + 1] - 1 : text.length;
    return { from, to: Math.max(end, from), message: p.message };
  });
}

export function viaLabel(via: 'ssh' | 'sql'): { label: string; hint: string } {
  return via === 'ssh'
    ? { label: 'via SSH', hint: 'Read and written as a file on the SSH server, which runs the database' }
    : { label: 'via SQL', hint: 'Read and written by the database server itself — needs a superuser, or the right to read and write server files' };
}
