import { describe, expect, it } from 'vitest';
import fixture from '../../../e2e/engines.json';
import { ENGINES, engine, engineForScheme, nameFromFile, sqlite } from '.';

const sources = import.meta.glob('/src/**/*.{ts,svelte}', { query: '?raw', import: 'default', eager: true }) as Record<string, string>;

describe('engines', () => {
  it('has the features of every engine the backend sends', () => {
    expect(Object.keys(fixture).sort()).toEqual(ENGINES.map(e => e.driver).sort());
  });

  it('finds engines by driver and URL scheme', () => {
    for (const e of ENGINES) {
      expect(engine(e.driver)).toBe(e);
      for (const s of e.schemes) expect(engineForScheme(s)).toBe(e);
    }
  });

  it('names a connection after its file', () => {
    expect(nameFromFile(sqlite, '/data/shop.sqlite3')).toBe('shop');
    expect(nameFromFile(sqlite, 'C:\\data\\app.db')).toBe('app');
    expect(nameFromFile(sqlite, '/data/notes.txt')).toBe('notes.txt');
  });

  it('keeps engine names out of shared code', () => {
    expect(Object.keys(sources).length).toBeGreaterThan(30);
    const names = ENGINES.map(e => e.driver).join('|');
    const literal = new RegExp(`['"\`](${names})['"\`]`);
    const offenders = Object.entries(sources)
      .filter(([path]) => !path.startsWith('/src/lib/engines/') && path !== '/src/lib/api/wire.ts' && !path.endsWith('.test.ts'))
      .filter(([, text]) => literal.test(text))
      .map(([path, text]) => `${path}: ${text.split('\n').find(l => literal.test(l))!.trim()}`);
    expect(offenders).toEqual([]);
  });
});
