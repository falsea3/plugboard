import { describe, expect, it } from 'vitest';
import { problemRanges } from './conf';

describe('problemRanges', () => {
  const text = 'a = 1\nwork_mem = \n# end';
  it('marks the whole line a problem points at', () => {
    const [p] = problemRanges(text, [{ line: 2, message: 'syntax error' }]);
    expect(text.slice(p.from, p.to)).toBe('work_mem = ');
    expect(p.message).toBe('syntax error');
  });
  it('keeps lines past the end or before the start inside the text', () => {
    const [last, first] = problemRanges(text, [{ line: 9, message: '' }, { line: 0, message: '' }]);
    expect(text.slice(last.from, last.to)).toBe('# end');
    expect(text.slice(first.from, first.to)).toBe('a = 1');
  });
  it('handles an empty last line', () => {
    const [p] = problemRanges('a\n', [{ line: 2, message: '' }]);
    expect(p).toEqual({ from: 2, to: 2, message: '' });
  });
});
