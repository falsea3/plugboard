export interface JSONProblem {
  from: number;
  to: number;
  message: string;
}

class Stop extends Error {
  constructor(readonly problem: JSONProblem) {
    super(problem.message);
  }
}

const NUMBER = /-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/y;
const ESCAPES = new Set(['"', '\\', '/', 'b', 'f', 'n', 'r', 't']);
const WORD = /[^\s,:{}[\]"]*/y;

export function jsonProblem(text: string): JSONProblem | null {
  let i = 0;
  const fail = (from: number, message: string, to = from + 1): never => {
    throw new Stop({ from, to: Math.min(Math.max(to, from + 1), Math.max(text.length, from + 1)), message });
  };
  const skip = () => {
    while (i < text.length && ' \t\n\r'.includes(text[i])) i++;
  };
  const what = (at: number) => {
    WORD.lastIndex = at;
    const m = WORD.exec(text);
    return m && m[0] ? m[0] : text[at];
  };

  const string = () => {
    const start = i++;
    while (i < text.length) {
      const ch = text[i];
      if (ch === '"') {
        i++;
        return;
      }
      if (ch === '\\') {
        const next = text[i + 1];
        if (next === 'u') {
          if (!/^[0-9a-fA-F]{4}$/.test(text.slice(i + 2, i + 6))) fail(i, 'A \\u escape needs four hex digits', i + 6);
          i += 6;
          continue;
        }
        if (!ESCAPES.has(next)) fail(i, `“\\${next ?? ''}” isn't an escape JSON knows`, i + 2);
        i += 2;
        continue;
      }
      if (ch < ' ') fail(i, ch === '\n' ? 'The string isn’t closed before the line ends' : 'A string can’t hold a raw control character');
      i++;
    }
    fail(start, 'The string isn’t closed', text.length);
  };

  const value = (): void => {
    skip();
    const ch = text[i];
    if (ch === undefined) fail(i, 'Expected a value, but the text ends here');
    if (ch === '{') return object();
    if (ch === '[') return array();
    if (ch === '"') return string();
    for (const lit of ['true', 'false', 'null']) {
      if (text.startsWith(lit, i)) {
        i += lit.length;
        return;
      }
    }
    NUMBER.lastIndex = i;
    const m = NUMBER.exec(text);
    if (m && m[0] !== '-') {
      i += m[0].length;
      return;
    }
    const found = what(i);
    fail(i, `Expected a value, found “${found}”`, i + found.length);
  };

  const object = () => {
    i++;
    skip();
    if (text[i] === '}') {
      i++;
      return;
    }
    for (;;) {
      skip();
      if (text[i] !== '"') {
        if (text[i] === '}') fail(text.lastIndexOf(',', i), 'Remove the comma before “}”');
        fail(i, text[i] === undefined ? 'The object isn’t closed with “}”' : 'Expected a property name in double quotes');
      }
      string();
      skip();
      if (text[i] !== ':') fail(i, 'Expected “:” after the property name');
      i++;
      value();
      skip();
      if (text[i] === ',') {
        i++;
        continue;
      }
      if (text[i] === '}') {
        i++;
        return;
      }
      fail(i, text[i] === undefined ? 'The object isn’t closed with “}”' : 'Expected “,” or “}”');
    }
  };

  const array = () => {
    i++;
    skip();
    if (text[i] === ']') {
      i++;
      return;
    }
    for (;;) {
      skip();
      if (text[i] === ']') fail(text.lastIndexOf(',', i), 'Remove the comma before “]”');
      value();
      skip();
      if (text[i] === ',') {
        i++;
        continue;
      }
      if (text[i] === ']') {
        i++;
        return;
      }
      fail(i, text[i] === undefined ? 'The array isn’t closed with “]”' : 'Expected “,” or “]”');
    }
  };

  try {
    skip();
    if (i === text.length) return { from: 0, to: 0, message: 'Empty — type a JSON value' };
    value();
    skip();
    if (i < text.length) fail(i, 'Unexpected text after the JSON value', text.length);
    return null;
  } catch (err) {
    if (err instanceof Stop) return err.problem;
    throw err;
  }
}

export function lineCol(text: string, offset: number): { line: number; col: number } {
  let line = 1;
  let start = 0;
  for (let i = 0; i < offset && i < text.length; i++) {
    if (text[i] === '\n') {
      line++;
      start = i + 1;
    }
  }
  return { line, col: offset - start + 1 };
}
