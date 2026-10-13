import { describe, expect, it } from 'vitest';
import type { Role } from '../api/backend';
import { filterRoles, newRole, roleBadges, sameSpec, specOf } from './roles';

const role = (name: string, extra: Partial<Role> = {}): Role => ({ ...newRole(), name, ...extra });

describe('roles', () => {
  it('a new role logs in, inherits and has no connection limit', () => {
    expect(newRole()).toMatchObject({ login: true, inherit: true, connLimit: -1, memberOf: [] });
  });
  it('compares forms ignoring membership order and surrounding spaces', () => {
    const a = specOf(role('app', { memberOf: ['a', 'b'] }));
    expect(sameSpec(a, { ...a, name: ' app ', memberOf: ['b', 'a'] })).toBe(true);
    expect(sameSpec(a, { ...a, password: 'x' })).toBe(false);
  });
  it('copies the membership list so the form can change it', () => {
    const r = role('app', { memberOf: ['a'] });
    specOf(r).memberOf.push('b');
    expect(r.memberOf).toEqual(['a']);
  });
  it('hides the built-in pg_ roles unless asked', () => {
    const list = [role('app'), role('pg_monitor', { system: true }), role('admin')];
    expect(filterRoles(list, '', false).map(r => r.name)).toEqual(['app', 'admin']);
    expect(filterRoles(list, 'MON', true).map(r => r.name)).toEqual(['pg_monitor']);
  });
  it('shows what matters most first', () => {
    expect(roleBadges(role('x', { superuser: true, login: true }))).toEqual(['superuser', 'login']);
    expect(roleBadges(role('x', { login: false }))).toEqual([]);
  });
});
