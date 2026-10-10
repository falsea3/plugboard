import { expect, it } from 'vitest';
import { failMessage } from './commit';

it('failMessage names the row that failed', () => {
  expect(failMessage('update', 4, 'boom')).toBe('Updating row 5 failed — nothing was saved: boom');
  expect(failMessage('insert', 9, 'dup')).toBe('Inserting a new row failed — nothing was saved: dup');
  expect(failMessage(undefined, null, 'gone')).toBe('gone');
});
