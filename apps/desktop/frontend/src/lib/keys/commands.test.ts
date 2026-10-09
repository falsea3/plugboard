import { expect, it } from 'vitest';
import { COMMANDS, lineAt } from './commands';

it('lineAt takes the trimmed line under the cursor, skipping blanks and comments', () => {
  const doc = 'GET a\n  HGETALL h  \n\n# note\nINCR x';
  expect(lineAt(doc, 0)).toEqual({ text: 'GET a', from: 0 });
  expect(lineAt(doc, 9)).toEqual({ text: 'HGETALL h', from: 8 });
  expect(lineAt(doc, doc.indexOf('\n\n') + 1)).toBeNull();
  expect(lineAt(doc, doc.indexOf('# note') + 2)).toBeNull();
  expect(lineAt(doc, doc.length)).toEqual({ text: 'INCR x', from: doc.length - 6 });
});

it('lists commands once, in capitals', () => {
  expect(new Set(COMMANDS).size).toBe(COMMANDS.length);
  expect(COMMANDS.every(c => c === c.toUpperCase() && c.length > 0)).toBe(true);
});
