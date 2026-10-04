import { describe, expect, it } from 'vitest';
import { deleteStatement, insertStatement, literal, quoteIdent, selectStatement, updateStatement } from './generate';
import engines from '../../../e2e/engines.json';

const { postgres, mysql, sqlite } = engines;

describe('sqlText', () => {
  it('quotes identifiers per engine', () => {
    expect(quoteIdent(postgres, 'we"ird')).toBe('"we""ird"');
    expect(quoteIdent(mysql, 'we`ird')).toBe('`we``ird`');
  });

  it('writes literals per engine', () => {
    expect(literal(postgres, "it's")).toBe("'it''s'");
    expect(literal(mysql, 'a\\b')).toBe("'a\\\\b'");
    expect(literal(postgres, 'a\\b')).toBe("'a\\b'");
    expect(literal(mysql, true)).toBe('1');
    expect(literal(postgres, false)).toBe('false');
    expect(literal(sqlite, null)).toBe('NULL');
    expect(literal(sqlite, 1.5)).toBe('1.5');
  });

  const users = { schema: 'public', table: 'users', columns: ['id', 'name'], key: ['id'] };
  const rows = [[1, 'ann'], [2, null]];

  it('builds a multi-row INSERT', () => {
    expect(insertStatement(postgres, users, rows)).toBe('INSERT INTO "public"."users" ("id", "name") VALUES\n  (1, \'ann\'),\n  (2, NULL);');
  });

  it('finds rows by primary key', () => {
    expect(selectStatement(postgres, users, rows)).toBe('SELECT * FROM "public"."users"\nWHERE "id" IN (1, 2);');
    expect(deleteStatement(mysql, { ...users, schema: '' }, rows.slice(0, 1))).toBe('DELETE FROM `users`\nWHERE `id` = 1;');
  });

  it('matches every column of a composite key', () => {
    const lines = { schema: '', table: 'lines', columns: ['order_id', 'n', 'qty'], key: ['order_id', 'n'] };
    expect(deleteStatement(sqlite, lines, [[7, 1, 3], [7, 2, 5]])).toBe(
      'DELETE FROM "lines"\nWHERE ("order_id" = 7 AND "n" = 1)\n   OR ("order_id" = 7 AND "n" = 2);',
    );
  });

  it('writes one UPDATE per row, setting the columns outside the key', () => {
    expect(updateStatement(mysql, users, rows)).toBe(
      "UPDATE `public`.`users` SET\n  `name` = 'ann'\nWHERE `id` = 1;\n\nUPDATE `public`.`users` SET\n  `name` = NULL\nWHERE `id` = 2;",
    );
  });
});
