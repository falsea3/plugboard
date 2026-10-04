import { jsonProblem, lineCol } from './check';

export function reindent(text: string, indent: number): string {
  const pad = (depth: number) => (indent ? '\n' + ' '.repeat(depth * indent) : '');
  let out = '';
  let depth = 0;
  let inString = false;
  let escaped = false;
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (inString) {
      out += ch;
      if (escaped) escaped = false;
      else if (ch === '\\') escaped = true;
      else if (ch === '"') inString = false;
      continue;
    }
    switch (ch) {
      case '"':
        inString = true;
        out += ch;
        break;
      case '{':
      case '[': {
        const close = ch === '{' ? '}' : ']';
        let j = i + 1;
        while (j < text.length && /\s/.test(text[j])) j++;
        if (text[j] === close) {
          out += ch + close;
          i = j;
          break;
        }
        depth++;
        out += ch + pad(depth);
        break;
      }
      case '}':
      case ']':
        depth--;
        out += pad(depth) + ch;
        break;
      case ',':
        out += ',' + pad(depth);
        break;
      case ':':
        out += indent ? ': ' : ':';
        break;
      case ' ':
      case '\t':
      case '\n':
      case '\r':
        break;
      default:
        out += ch;
    }
  }
  return out;
}

export function jsonError(text: string): string {
  const p = jsonProblem(text);
  if (!p) return '';
  if (p.from === 0 && p.to === 0) return p.message;
  const { line, col } = lineCol(text, p.from);
  return `Line ${line}, column ${col}: ${p.message}`;
}

export function looksLikeJSON(value: unknown): boolean {
  if (typeof value !== 'string') return false;
  const s = value.trim();
  if (!((s.startsWith('{') && s.endsWith('}')) || (s.startsWith('[') && s.endsWith(']')))) return false;
  return jsonError(s) === '';
}

export function jsonToSave(original: string, edited: string): string | null {
  if (original !== '' && reindent(original, 0) === reindent(edited, 0)) return null;
  return original.includes('\n') ? reindent(edited, 2) : reindent(edited, 0);
}
