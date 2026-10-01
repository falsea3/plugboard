import { describe, expect, it } from 'vitest';
import { lintSql } from './sqlLint';

const messages = (sql: string, mysql = false) => lintSql(sql, mysql).map(p => p.message);
const marked = (sql: string, mysql = false) => lintSql(sql, mysql).map(p => sql.slice(p.from, p.to));

describe('lintSql', () => {
  it('leaves valid SQL alone, including what parsers trip on', () => {
    for (const sql of [
      'select 1',
      "delete from t using u where u.id = t.id returning *",
      'select count(*) filter (where a > 1) over (partition by b) from t',
      'explain analyze select 1; values (1, 2); table users',
      "do $$ begin perform 1; end $$",
      "select 'a, from b', \"weird, ) name\" from t -- , from",
      'select a, /* note */ b from t',
      "select E'it\\'s', 'C:\\' from t",
      'select * from t for update skip locked',
      'sel', // still typing the first word
    ]) {
      expect(messages(sql), sql).toEqual([]);
    }
    expect(messages("select 'it\\'s' from t # comment (", true)).toEqual([]);
  });

  it('flags unclosed quotes, brackets and comments', () => {
    expect(marked("select 'abc from t")).toEqual(["'abc from t"]);
    expect(marked('select (1 + 2 from t')).toEqual(['(']);
    expect(marked('select 1) from t')).toEqual([')']);
    expect(messages('select 1 /* note')).toEqual(['This comment is never closed: add */']);
    expect(messages('do $body$ begin')).toEqual(['This $body$ body is never closed: add $body$']);
  });

  it('flags a comma with nothing after it', () => {
    expect(marked('select a, b, from t')).toEqual([',']);
    expect(marked('create table t (a int,\n)')).toEqual([',']);
    expect(messages('select a, b from t')).toEqual([]);
  });

  it('suggests the statement a typo was meant to be', () => {
    expect(messages('selec * from t')).toEqual(['Unknown statement “selec” — did you mean SELECT?']);
    expect(marked('-- note\n  upadte t set a = 1')).toEqual(['upadte']);
    expect(messages('vaccum analyze')).toEqual(['Unknown statement “vaccum” — did you mean VACUUM?']);
    expect(messages('frobnicate the database')).toEqual([]); // not a near miss: maybe something we don't know
  });

  it('keeps checking the rest when a quote is left open', () => {
    expect(marked("selec id, from t where name = 'Ann")).toEqual(["'Ann", ',', 'selec']);
  });

  it('points at the right statement in a script', () => {
    const sql = "select 1;\nselect 'x from t;\n";
    const [p] = lintSql(sql, false);
    expect(sql.slice(p.from, p.to)).toBe("'x from t;\n".trimEnd());
  });
});
