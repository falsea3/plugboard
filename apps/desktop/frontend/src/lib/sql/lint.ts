import { dollarTagAt, escapesBackslash, lineCommentAt, statementRanges } from './statements';
import type { SqlSyntax } from '../api/wire';

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
  'GET', 'SIGNAL', 'RESIGNAL', 'CLONE', 'CHANGE', 'STOP', 'RESTART', 'SHUTDOWN', 'BINLOG', 'CACHE',
  'LIST', 'EXISTS', 'SYSTEM', 'EXCHANGE',
  'IF', 'ELSE', 'ELSEIF', 'CASE', 'LOOP', 'WHILE', 'REPEAT', 'LEAVE', 'ITERATE', 'OPEN', 'RETURN',
]);

const AFTER_COMMA = new Set(['FROM', 'WHERE', 'GROUP', 'ORDER', 'HAVING', 'LIMIT']);
const WORD = /[A-Za-z_][A-Za-z0-9_$]*/y;
const SPACE = /\s/;

export function lintSql(script: string, syntax: SqlSyntax): SqlProblem[] {
  const problems: SqlProblem[] = [];
  for (const stmt of statementRanges(script, syntax)) {
    problems.push(...lintStatement(stmt.text, syntax).map(p => ({ ...p, from: p.from + stmt.from, to: p.to + stmt.from })));
  }
  return problems;
}

function lintStatement(s: string, syntax: SqlSyntax): SqlProblem[] {
  const problems: SqlProblem[] = [];
  const open: number[] = [];
  const code: number[] = [];
  let cut = false;

  for (let i = 0; i < s.length; i++) {
    const c = s[i];
    const tag = c === '$' && syntax.dollarQuotes ? dollarTagAt(s, i) : null;
    if (c === "'" || c === '"' || c === '`') {
      const end = closingQuote(s, i, syntax);
      if (end < 0) {
        const what = c === "'" || (c === '"' && syntax.doubleQuotedStrings) ? 'string' : 'quoted name';
        problems.push({ from: i, to: s.length, message: `This ${what} is never closed: add the ending ${c}` });
        cut = true;
        break;
      }
      i = end;
    } else if (lineCommentAt(s, i, syntax)) {
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
    } else if (tag) {
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

  problems.push(...trailingCommas(s, code, syntax));
  const typo = firstWordTypo(s);
  if (typo) problems.push(typo);
  return problems;
}

function closingQuote(s: string, i: number, syntax: SqlSyntax): number {
  const q = s[i];
  const backslash = escapesBackslash(s, i, syntax);
  for (let j = i + 1; j < s.length; j++) {
    if (s[j] === '\\' && backslash) j++;
    else if (s[j] === q) {
      if (s[j + 1] === q) j++;
      else return j;
    }
  }
  return -1;
}

function trailingCommas(s: string, code: number[], syntax: SqlSyntax): SqlProblem[] {
  const problems: SqlProblem[] = [];
  for (const i of code) {
    if (s[i] !== ',') continue;
    const j = skipSpaceAndComments(s, i + 1, syntax);
    WORD.lastIndex = j;
    const word = WORD.exec(s)?.[0].toUpperCase() ?? '';
    if (s[j] === ')' || AFTER_COMMA.has(word)) {
      problems.push({ from: i, to: i + 1, message: 'Nothing follows this comma' });
    }
  }
  return problems;
}

function skipSpaceAndComments(s: string, j: number, syntax: SqlSyntax): number {
  for (;;) {
    while (j < s.length && SPACE.test(s[j])) j++;
    if (j < s.length && lineCommentAt(s, j, syntax)) {
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
  if (!m || !m[3]) return null;
  const word = m[2].toUpperCase();
  if (STATEMENT_WORDS.has(word)) return null;
  let best = '';
  let bestDistance = Infinity;
  for (const k of STATEMENT_WORDS) {
    const d = distance(word, k);
    if (d < bestDistance) [best, bestDistance] = [k, d];
  }
  const limit = word.length >= 6 ? 2 : 1;
  if (bestDistance > limit) return null;
  const from = m[0].length - m[2].length - (m[3]?.length ?? 0);
  return { from, to: from + m[2].length, message: `Unknown statement “${m[2]}” — did you mean ${best}?` };
}

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
