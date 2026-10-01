import { describe, expect, it } from 'vitest';
import { parseNotes } from './releaseNotes';

describe('parseNotes', () => {
  it('reads headings, items, bold and code', () => {
    expect(parseNotes('### Fixed\n\n- **Edits** keep `char(2)` values whole.')).toEqual([
      { kind: 'heading', spans: [{ text: 'Fixed' }] },
      {
        kind: 'item',
        spans: [{ text: 'Edits', strong: true }, { text: ' keep ' }, { text: 'char(2)', code: true }, { text: ' values whole.' }],
      },
    ]);
  });

  it('keeps markup as plain text', () => {
    expect(parseNotes('<img src=x onerror=alert(1)> see [the docs](https://x)')).toEqual([
      { kind: 'text', spans: [{ text: '<img src=x onerror=alert(1)> see ' }, { text: 'the docs' }] },
    ]);
  });
});
