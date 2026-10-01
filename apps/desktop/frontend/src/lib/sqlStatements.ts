export interface StatementRange {
  from: number;
  to: number;
  text: string;
}

export function statementRanges(script: string, mysql: boolean): StatementRange[] {
  const out: StatementRange[] = [];
  const n = script.length;
  let start = 0;

  const flush = (end: number) => {
    const raw = script.slice(start, end);
    const lead = raw.length - raw.trimStart().length;
    const text = raw.trim();
    if (text && hasCode(text, mysql)) out.push({ from: start + lead, to: start + lead + text.length, text });
  };

  for (let i = 0; i < n; i++) {
    const c = script[i];
    if (c === "'" || c === '"' || c === '`') {
      i = skipQuoted(script, i, c, escapesBackslash(script, i, mysql));
    } else if (c === '-' && script[i + 1] === '-') {
      i = skipLine(script, i);
    } else if (c === '#' && mysql) {
      i = skipLine(script, i);
    } else if (c === '/' && script[i + 1] === '*') {
      const end = script.indexOf('*/', i + 2);
      i = end < 0 ? n - 1 : end + 1;
    } else if (c === '$' && !mysql) {
      const tag = /^\$(?:[A-Za-z_][A-Za-z0-9_]*)?\$/.exec(script.slice(i, i + 64))?.[0];
      if (tag) {
        const end = script.indexOf(tag, i + tag.length);
        i = end < 0 ? n - 1 : end + tag.length - 1;
      }
    } else if (c === ';') {
      flush(i);
      start = i + 1;
    }
  }
  flush(n);
  return out;
}

/** The statement containing pos, or the closest one before it. */
export function statementAt(script: string, pos: number, mysql: boolean): StatementRange | null {
  const ranges = statementRanges(script, mysql);
  let best: StatementRange | null = null;
  for (const r of ranges) {
    if (r.from <= pos) best = r;
    if (pos <= r.to + 1) return pos >= r.from ? r : (best ?? r);
  }
  return best;
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

/** Every MySQL string treats backslash as an escape, PostgreSQL only E'…'; backtick identifiers never do. */
function escapesBackslash(s: string, i: number, mysql: boolean): boolean {
  if (s[i] === '`') return false;
  if (mysql) return true;
  return s[i] === "'" && /[Ee]/.test(s[i - 1] ?? '') && !/\w/.test(s[i - 2] ?? '');
}

function skipLine(s: string, i: number): number {
  const nl = s.indexOf('\n', i);
  return nl < 0 ? s.length - 1 : nl;
}

function hasCode(text: string, mysql: boolean): boolean {
  const stripped = text
    .replace(/\/\*[\s\S]*?(\*\/|$)/g, '')
    .replace(mysql ? /(--|#)[^\n]*/g : /--[^\n]*/g, '');
  return stripped.trim() !== '';
}
