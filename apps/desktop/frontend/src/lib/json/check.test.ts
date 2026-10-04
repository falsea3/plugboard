import { describe, expect, it } from 'vitest';
import { jsonProblem, lineCol } from './check';

const at = (text: string) => {
  const p = jsonProblem(text);
  return p && { ...lineCol(text, p.from), text: text.slice(p.from, p.to), message: p.message };
};

describe('JSON problems', () => {
  it('accepts what JSON.parse accepts', () => {
    for (const ok of ['{}', '[]', '0', '-1.5e+3', '"a\\u00e9\\n"', '{"a": [1, true, null, {"b": "c"}]}', '  [ 1 , 2 ]  ']) {
      expect(jsonProblem(ok)).toBeNull();
    }
  });

  it('points at the exact place', () => {
    expect(at('{\n  "a": 1\n  "b": 2\n}')).toEqual({ line: 3, col: 3, text: '"', message: 'Expected “,” or “}”' });
    expect(at('{"a": 1,}')).toMatchObject({ col: 8, text: ',', message: 'Remove the comma before “}”' });
    expect(at('[1, 2,]')).toMatchObject({ col: 6, text: ',', message: 'Remove the comma before “]”' });
    expect(at('{"a": tru}')).toMatchObject({ col: 7, text: 'tru', message: 'Expected a value, found “tru”' });
    expect(at('{a: 1}')).toMatchObject({ col: 2, message: 'Expected a property name in double quotes' });
    expect(at('{"a" 1}')).toMatchObject({ col: 6, message: 'Expected “:” after the property name' });
    expect(at('{"a": "x}')).toMatchObject({ col: 7, message: 'The string isn’t closed' });
    expect(at('{"a": "x\n"}')).toMatchObject({ line: 1, col: 9, message: 'The string isn’t closed before the line ends' });
    expect(at('{"a": "\\q"}')).toMatchObject({ col: 8, text: '\\q' });
    expect(at('{"a": 1')).toMatchObject({ message: 'The object isn’t closed with “}”' });
    expect(at('[1] [2]')).toMatchObject({ col: 5, text: '[2]', message: 'Unexpected text after the JSON value' });
    expect(at('   ')).toMatchObject({ message: 'Empty — type a JSON value' });
  });
});
