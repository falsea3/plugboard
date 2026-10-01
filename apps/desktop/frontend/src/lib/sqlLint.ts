import { statementRanges } from './sqlStatements';

// Checks SQL as it's typed, for mistakes that are wrong in every dialect:
// unclosed quotes, brackets and comments, a typo in the statement's first
// word, a comma with nothing after it. A real parser would catch more, but the
// ones tried flag valid PostgreSQL and MySQL too often (RETURNING on DELETE,
// FILTER, VALUES…), and a red line under correct SQL is worse than none. The
// server still reports everything else when the statement runs.

export interface SqlProblem {
  from: number;
  to: number;
  message: string;
}

const STATEMENT_WORDS = new Set([
  'SELECT', 'WITH', 'INSERT', 'UPDATE', 'DELETE', 'MERGE', 'UPSERT', 'REPLACE', 'VALUES', 'TABLE',
  'CREATE', 'ALTER', 'DROP', 'TRUNCATE', 'RENAME', 'COMMENT', 'GRANT', 'REVOKE', 'REASSIGN', 'SECURITY',
  'BEGIN', 'START', 'COMMIT', 'END', 'ROLLBACK', 'ABORT', 'SAVEPOINT', 'RELEASE',
  'SET', 'RESET', 'SHOW', 'EXPLAIN', 'DESCRIBE', 'DESC', 'USE', 'CALL', 'DO', 'COPY', 'IMPORT',
  'DECLARE', 'FETCH', 'MOVE', 'CLOSE', 'PREPARE', 'EXECUTE', 'DEALLOCATE', 'LOCK', 'UNLOCK',
  'VACUUM', 'ANALYZE', 'ANALYSE', 'REINDEX', 'CLUSTER', 'REFRESH', 'CHECKPOINT', 'DISCARD',
  'LISTEN', 'NOTIFY', 'UNLISTEN', 'LOAD', 'PRAGMA', 'ATTACH', 'DETACH', 'HANDLER', 'OPTIMIZE',
  'REPAIR', 'CHECK', 'CHECKSUM', 'FLUSH', 'KILL', 'PURGE', 'INSTALL', 'UNINSTALL', 'XA', 'HELP',
]);

/** Words that can't follow a comma: "select a, from t" is missing a column. */
const AFTER_COMMA = new Set(['FROM', 'WHERE', 'GROUP', 'ORDER', 'HAVING', 'LIMIT']);

export function lintSql(script: string, mysql: boolean): SqlProblem[] {
  const problems: SqlProblem[] = [];
  for (const stmt of statementRanges(script, mysql)) {
    problems.push(...lintStatement(stmt.text, mysql).map(p => ({ ...p, from: p.from + stmt.from, to: p.to + stmt.from })));
  }
  return problems;
}

function lintStatement(s: string, mysql: boolean): SqlProblem[] {
  const problems: SqlProblem[] = [];
  const open: number[] = [];
  const code: number[] = []; // offsets of characters outside quotes and comments
  let cut = false; // an unclosed quote or comment runs to the end; brackets after it can't be told

  for (let i = 0; i < s.length; i++) {
    const c = s[i];
    if (c === "'" || c === '"' || c === '`') {
      const end = closingQuote(s, i, mysql);
      if (end < 0) {
        const what = c === "'" || (c === '"' && mysql) ? 'string' : 'quoted name';
        problems.push({ from: i, to: s.length, message: `This ${what} is never closed: add the ending ${c}` });
        cut = true;
        break;
      }
      i = end;
    } else if ((c === '-' && s[i + 1] === '-') || (c === '#' && mysql)) {
      const nl = s.indexOf('\n', i);
      i = nl < 0 ? s.length : nl;
    } else if (c === '/' && s[i + 1] === '*') {
      const end = s.indexOf('*/', i + 2);
      if (end < 0) {
        problems.push({ from: i, to: s.length, message: 'This comment is never closed: add */' });
        cut = true;
        break;
      }
      i = end + 1;
    } else if (c === '$' && !mysql && /^\$([A-Za-z_][A-Za-z0-9_]*)?\$/.test(s.slice(i, i + 64))) {
      const tag = /^\$([A-Za-z_][A-Za-z0-9_]*)?\$/.exec(s.slice(i, i + 64))![0];
      const end = s.indexOf(tag, i + tag.length);
      if (end < 0) {
        problems.push({ from: i, to: s.length, message: `This ${tag} body is never closed: add ${tag}` });
        cut = true;
        break;
      }
      i = end + tag.length - 1;
    } else {
      code.push(i);
      if (c === '(') open.push(i);
      else if (c === ')') {
        if (open.length === 0) problems.push({ from: i, to: i + 1, message: 'This ) has no ( to close' });
        else open.pop();
      }
    }
  }
  if (!cut) for (const i of open) problems.push({ from: i, to: i + 1, message: 'This ( is never closed' });

  problems.push(...trailingCommas(s, code, mysql));
  const typo = firstWordTypo(s);
  if (typo) problems.push(typo);
  return problems;
}

/** Index of the quote closing the one at s[i], or -1. */
function closingQuote(s: string, i: number, mysql: boolean): number {
  const q = s[i];
  const backslash = q !== '`' && (mysql || (q === "'" && /[Ee]/.test(s[i - 1] ?? '') && !/\w/.test(s[i - 2] ?? '')));
  for (let j = i + 1; j < s.length; j++) {
    if (s[j] === '\\' && backslash) j++;
    else if (s[j] === q) {
      if (s[j + 1] === q) j++;
      else return j;
    }
  }
  return -1;
}

function trailingCommas(s: string, code: number[], mysql: boolean): SqlProblem[] {
  const problems: SqlProblem[] = [];
  for (const i of code) {
    if (s[i] !== ',') continue;
    const j = skipSpaceAndComments(s, i + 1, mysql);
    const word = /^[A-Za-z]+/.exec(s.slice(j))?.[0].toUpperCase() ?? '';
    if (s[j] === ')' || AFTER_COMMA.has(word)) {
      problems.push({ from: i, to: i + 1, message: 'Nothing follows this comma' });
    }
  }
  return problems;
}

function skipSpaceAndComments(s: string, j: number, mysql: boolean): number {
  for (;;) {
    while (j < s.length && /\s/.test(s[j])) j++;
    if (s.startsWith('--', j) || (mysql && s[j] === '#')) {
      const nl = s.indexOf('\n', j);
      j = nl < 0 ? s.length : nl;
    } else if (s.startsWith('/*', j)) {
      const end = s.indexOf('*/', j + 2);
      j = end < 0 ? s.length : end + 2;
    } else {
      return j;
    }
  }
}

function firstWordTypo(s: string): SqlProblem | null {
  const m = /^(\s|\(|--[^\n]*\n?|#[^\n]*\n?|\/\*[\s\S]*?\*\/)*([A-Za-z_]+)(\s+\S)?/.exec(s);
  if (!m || !m[3]) return null; // nothing after the word yet: still typing it
  const word = m[2].toUpperCase();
  if (STATEMENT_WORDS.has(word)) return null;
  let best = '';
  let bestDistance = Infinity;
  for (const k of STATEMENT_WORDS) {
    const d = distance(word, k);
    if (d < bestDistance) [best, bestDistance] = [k, d];
  }
  const limit = word.length >= 6 ? 2 : 1;
  if (bestDistance > limit) return null; // not a near miss; maybe a statement we don't know
  const from = m[0].length - m[2].length - (m[3]?.length ?? 0);
  return { from, to: from + m[2].length, message: `Unknown statement “${m[2]}” — did you mean ${best}?` };
}

/** Edits (insert, delete, change, swap two neighbours) to turn a into b. */
function distance(a: string, b: string): number {
  const d = Array.from({ length: a.length + 1 }, (_, i) => [i, ...new Array(b.length).fill(0)]);
  for (let j = 1; j <= b.length; j++) d[0][j] = j;
  for (let i = 1; i <= a.length; i++) {
    for (let j = 1; j <= b.length; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      d[i][j] = Math.min(d[i - 1][j] + 1, d[i][j - 1] + 1, d[i - 1][j - 1] + cost);
      if (i > 1 && j > 1 && a[i - 1] === b[j - 2] && a[i - 2] === b[j - 1]) d[i][j] = Math.min(d[i][j], d[i - 2][j - 2] + 1);
    }
  }
  return d[a.length][b.length];
}
