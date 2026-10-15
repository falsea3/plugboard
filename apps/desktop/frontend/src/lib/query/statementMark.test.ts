import { describe, expect, it } from 'vitest';
import { statementOf } from './statementMark';

const syntax = { hashComments: false, dashCommentNeedsSpace: false, backslashEscapes: false, escapeStrings: true, doubleQuotedStrings: false, dollarQuotes: true, executableComments: false };

describe('statementOf', () => {
  const sql = 'select 1;\n\nselect 2;\n';
  it('finds the statement under the cursor and counts them', () => {
    expect(statementOf(sql, 3, syntax)).toMatchObject({ index: 0, count: 2 });
    expect(statementOf(sql, 14, syntax)).toMatchObject({ index: 1, count: 2, from: 11 });
    expect(statementOf(sql, 10, syntax)).toMatchObject({ index: -1, count: 2 });
    expect(statementOf('', 0, syntax)).toBeNull();
  });
});
