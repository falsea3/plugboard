import { describe, expect, it } from 'vitest';
import { cellKind, connectionTarget, calcColumnWidth, formatCell, formatDuration, toTSV } from './format';
import { emptyConnection, mysql, postgres, sqlite } from './engines';

describe('formatCell', () => {
  it('renders NULL, booleans and multiline text on one line', () => {
    expect(formatCell(null)).toBe('NULL');
    expect(formatCell(true)).toBe('true');
    expect(formatCell('a\nb')).toBe('a ↵');
    expect(formatCell(42)).toBe('42');
  });

  it('truncates very long values', () => {
    expect(formatCell('x'.repeat(500))).toHaveLength(301);
  });
});

describe('cellKind', () => {
  it('treats numeric strings from numeric columns as numbers', () => {
    expect(cellKind('9007199254740993', 'number')).toBe('number');
    expect(cellKind('12.50', 'number')).toBe('number');
    expect(cellKind('12', 'text')).toBe('text');
    expect(cellKind(null, 'number')).toBe('null');
  });
});

describe('toTSV', () => {
  it('quotes values that contain tabs, quotes or newlines', () => {
    expect(toTSV(['a', 'b'], [[1, 'x\ty'], [null, 'say "hi"']])).toBe('a\tb\n1\t"x\ty"\n\t"say ""hi"""');
  });
});

describe('connectionTarget', () => {
  it('shows host, port and database', () => {
    expect(connectionTarget({ ...emptyConnection(postgres), database: 'app' })).toBe('127.0.0.1:5432/app');
  });

  it('shows only the file name for SQLite', () => {
    expect(connectionTarget({ ...emptyConnection(sqlite), file: '/Users/me/data/app.db' })).toBe('app.db');
  });
});

describe('formatDuration', () => {
  it('scales units', () => {
    expect(formatDuration(0.5)).toBe('0.50 ms');
    expect(formatDuration(12.4)).toBe('12 ms');
    expect(formatDuration(2500)).toBe('2.50 s');
  });
});

describe('calcColumnWidth', () => {
  it('stays within bounds', () => {
    expect(calcColumnWidth('id', 'int', [[1]], 0)).toBe(64);
    expect(calcColumnWidth('body', 'text', [['x'.repeat(1000)]], 0)).toBe(360);
  });
});
