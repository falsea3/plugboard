import { describe, expect, it } from 'vitest';
import { toCSV } from './csv';
import { formatSql } from './format';

describe('csv', () => {
  it('quotes only what needs quoting and leaves NULL empty', () => {
    expect(toCSV(['a', 'b'], [[1, 'x,y'], [null, 'say "hi"'], [true, ' pad']])).toBe('a,b\r\n1,"x,y"\r\n,"say ""hi"""\r\ntrue," pad"\r\n');
  });
});

describe('format', () => {
  it('lays out a statement and keeps keyword case', () => {
    expect(formatSql('select a,b from t where a=1', 'sql')).toBe('select\n  a,\n  b\nfrom\n  t\nwhere\n  a = 1');
  });
});
