import { describe, expect, it } from 'vitest';
import { jsonError, jsonToSave, looksLikeJSON, reindent } from './text';

describe('JSON text', () => {
  it('pretty-prints and compacts without touching strings or numbers', () => {
    const src = '{"id": 9007199254740993, "tags": ["a, b", "{x}"], "note": "say \\"hi\\" : ok", "empty": {}, "list": [ ]}';
    const pretty = reindent(src, 2);
    expect(pretty).toBe(
      '{\n  "id": 9007199254740993,\n  "tags": [\n    "a, b",\n    "{x}"\n  ],\n  "note": "say \\"hi\\" : ok",\n  "empty": {},\n  "list": []\n}',
    );
    expect(reindent(pretty, 0)).toBe('{"id":9007199254740993,"tags":["a, b","{x}"],"note":"say \\"hi\\" : ok","empty":{},"list":[]}');
  });

  it('tells valid JSON from the rest', () => {
    expect(jsonError('{"a": 1}')).toBe('');
    expect(jsonError('{"a": }')).not.toBe('');
    expect(jsonError('  ')).toContain('Empty');
    expect(looksLikeJSON('{"a":[1,2]}')).toBe(true);
    expect(looksLikeJSON('[1, 2')).toBe(false);
    expect(looksLikeJSON('hello')).toBe(false);
    expect(looksLikeJSON(42)).toBe(false);
  });

  it('saves nothing when only the layout changed, and keeps the original style', () => {
    expect(jsonToSave('{"a": 1}', '{\n  "a": 1\n}')).toBeNull();
    expect(jsonToSave('{"a": 1}', '{\n  "a": 2\n}')).toBe('{"a":2}');
    expect(jsonToSave('{\n  "a": 1\n}', '{"a":2}')).toBe('{\n  "a": 2\n}');
    expect(jsonToSave('', '[]')).toBe('[]');
  });
});
