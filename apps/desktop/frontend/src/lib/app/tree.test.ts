import { describe, expect, it } from 'vitest';
import features from '../../../e2e/engines.json';
import type { Column, EngineFeatures, TableInfo } from '../api/wire';
import { columnChange, treeMenu, treePrompt, treeSQL } from './tree';

const pg = (features as Record<string, EngineFeatures>).postgres;
const my = (features as Record<string, EngineFeatures>).mysql;
const albums: TableInfo = { schema: 'public', name: 'albums', kind: 'table' } as TableInfo;
const column = { name: 'title', type: 'text' } as Column;

describe('sidebar tree', () => {
  it('turns a table action into SQL the engine reads', () => {
    expect(treeSQL('tree-query', pg, { table: albums })).toBe('SELECT *\nFROM "public"."albums"\nLIMIT 100;');
    expect(treeSQL('tree-drop', my, { table: { ...albums, schema: 'shop' } })).toBe('DROP TABLE `shop`.`albums`;');
    expect(treeSQL('tree-drop', pg, { table: { ...albums, kind: 'view' } })).toBe('DROP VIEW "public"."albums";');
    expect(treeSQL('tree-open', pg, { table: albums })).toBeNull();
  });

  it('keeps changes off in a read-only session and for views', () => {
    const ids = (items: ReturnType<typeof treeMenu>) => items.flatMap(i => (i !== 'sep' && !i.disabled ? [i.id] : []));
    expect(ids(treeMenu({ table: albums }, true, true))).not.toContain('tree-drop');
    expect(ids(treeMenu({ table: { ...albums, kind: 'view' } }, false, true))).toEqual(
      expect.arrayContaining(['tree-open', 'tree-drop']),
    );
    expect(ids(treeMenu({ table: { ...albums, kind: 'view' } }, false, true))).not.toContain('tree-rename');
    expect(ids(treeMenu({ table: albums, column }, false, true))).toEqual(['tree-copy-name', 'tree-query', 'tree-structure', 'tree-rename', 'tree-type', 'tree-drop']);
    expect(ids(treeMenu({ table: albums, column }, false, false))).not.toContain('tree-type');
  });

  it('asks for a new name or type and turns it into a structure change', () => {
    expect(treePrompt('tree-type', { table: albums, column }, [{ name: 'year', type: 'integer' } as Column])).toMatchObject({
      value: 'text',
      suggestions: expect.arrayContaining(['integer', 'varchar(255)']),
    });
    expect(treePrompt('tree-rename', { table: albums })).toMatchObject({ value: 'albums' });
    expect(columnChange('tree-type', { table: albums, column }, ' varchar(80) ')).toEqual({
      schema: 'public',
      table: 'albums',
      changes: [{ kind: 'update', column: 'title', type: 'varchar(80)', defaultSet: false, default: null }],
    });
    expect(columnChange('tree-rename', { table: albums, column }, 'name')?.changes[0]).toMatchObject({ name: 'name' });
    expect(columnChange('tree-rename', { table: albums }, 'x')).toBeNull();
  });
});
