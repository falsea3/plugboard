import { describe, expect, it } from 'vitest';
import { insertStatement, literal, quoteIdent } from './sqlText';

describe('sqlText', () => {
  it('quotes identifiers per dialect', () => {
    expect(quoteIdent('postgres', 'we"ird')).toBe('"we""ird"');
    expect(quoteIdent('mysql', 'we`ird')).toBe('`we``ird`');
  });

  it('writes literals per dialect', () => {
    expect(literal('postgres', "it's")).toBe("'it''s'");
    expect(literal('mysql', 'a\\b')).toBe("'a\\\\b'");
    expect(literal('postgres', 'a\\b')).toBe("'a\\b'");
    expect(literal('mysql', true)).toBe('1');
    expect(literal('postgres', false)).toBe('false');
    expect(literal('sqlite', null)).toBe('NULL');
    expect(literal('sqlite', 1.5)).toBe('1.5');
  });

  it('builds a multi-row INSERT', () => {
    expect(insertStatement('postgres', 'public', 'users', ['id', 'name'], [[1, 'ann'], [2, null]])).toBe(
      'INSERT INTO "public"."users" ("id", "name") VALUES\n  (1, \'ann\'),\n  (2, NULL);',
    );
  });
});
