import { describe, expect, it } from 'vitest';
import engines from '../../../e2e/engines.json';
import { errorRange, serverProblems, utf16Offset } from './problems';

const pg = engines.postgres.syntax;

describe('sqlProblems', () => {
  it('turns byte offsets into editor positions', () => {
    expect(utf16Offset("select 'шл' fron", 14)).toBe(12);
    expect(utf16Offset('a😀b', 5)).toBe(3);
    expect(utf16Offset('abc', 10)).toBe(3);
  });

  it('marks the word the server points at, or the last one at the end', () => {
    expect(errorRange('select * fron t', 9)).toEqual([9, 13]);
    expect(errorRange('select (1', 7)).toEqual([7, 8]);
    expect(errorRange('select * from t where  ', 23)).toEqual([16, 21]);
  });

  it('places a problem in its statement of the script', () => {
    const script = 'select 1;\n  selec 2;\nselect * from t where';
    const problems = serverProblems(script, 5, pg, [
      { index: 1, position: 0, message: 'near selec' },
      { index: 2, position: 21, message: 'end of input' },
      { index: 9, position: 0, message: 'gone' },
    ]);
    expect(problems.map(p => script.slice(p.from - 5, p.to - 5))).toEqual(['selec', 'where']);
  });
});
