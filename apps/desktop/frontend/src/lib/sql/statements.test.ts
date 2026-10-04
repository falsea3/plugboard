import { describe, expect, it } from 'vitest';
import { statementAt, statementRanges } from './statements';
import engines from '../../../e2e/engines.json';


const pg = engines.postgres.syntax;
const my = engines.mysql.syntax;
const texts = (s: string, syntax = pg) => statementRanges(s, syntax).map(r => r.text);

describe('statementRanges', () => {
  it('splits on top-level semicolons only', () => {
    expect(texts("select 'a;b'; select \"x;y\" -- ;\n; select 3")).toEqual([
      "select 'a;b'",
      'select "x;y" -- ;',
      'select 3',
    ]);
  });

  it('keeps PostgreSQL dollar-quoted bodies whole', () => {
    expect(texts('do $$ begin perform 1; end $$; select 2')).toEqual(['do $$ begin perform 1; end $$', 'select 2']);
    expect(texts('select a$b$ from t; select 2')).toEqual(['select a$b$ from t', 'select 2']);
  });

  it('drops comment-only pieces', () => {
    expect(texts('-- note\n; /* x */ ; select 1')).toEqual(['select 1']);
  });

  it('honours MySQL hash comments and backslash escapes', () => {
    expect(texts("select 'a\\';b'; select 1 # ;\n", my)).toEqual(["select 'a\\';b'", 'select 1 # ;']);
    expect(texts("select d #> '{a}' from t; select 2")).toEqual(["select d #> '{a}' from t", 'select 2']);
    expect(texts('select `a\\`; select 2', my)).toEqual(['select `a\\`', 'select 2']);
    expect(texts('select 1--1; select 2', my)).toEqual(['select 1--1', 'select 2']);
    expect(texts('select 1--; x\n; select 2')).toEqual(['select 1--; x', 'select 2']);
  });

  it("reads PostgreSQL backslashes as escapes only in E'…' strings", () => {
    expect(texts("select 'C:\\'; select 2")).toEqual(["select 'C:\\'", 'select 2']);
    expect(texts("select E'it\\'s; x'; select 2")).toEqual(["select E'it\\'s; x'", 'select 2']);
    expect(texts("select type'a\\'; select 2")).toEqual(["select type'a\\'", 'select 2']);
  });

  it('reports offsets that slice back to the text', () => {
    const s = '  select 1;\n\n  select 2  ';
    for (const r of statementRanges(s, pg)) expect(s.slice(r.from, r.to)).toBe(r.text);
  });
});

describe('statementAt', () => {
  const s = 'select 1;\nselect 2;\n\nselect 3';

  it('finds the statement under the cursor', () => {
    expect(statementAt(s, 3, pg)?.text).toBe('select 1');
    expect(statementAt(s, s.indexOf('2'), pg)?.text).toBe('select 2');
    expect(statementAt(s, s.length, pg)?.text).toBe('select 3');
  });

  it('uses the statement just before the cursor when it sits after a semicolon', () => {
    expect(statementAt(s, s.indexOf(';') + 1, pg)?.text).toBe('select 1');
  });

  it('returns null for an empty script', () => {
    expect(statementAt('  ', 0, pg)).toBeNull();
  });
});
