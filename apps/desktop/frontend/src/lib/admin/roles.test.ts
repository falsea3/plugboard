import { describe, expect, it } from 'vitest';
import type { Role, RoleForm } from '../api/backend';
import { displayName, filterRoles, grantsScript, newRole, roleBadges, sameSpec, specOf } from './roles';

const blank: Role = { id: '', name: '', host: '', flags: {}, connLimit: -1, validUntil: '', memberOf: [], grants: [], rules: '', kind: '', grantable: true, system: false, current: false };
const form: RoleForm = {
  noun: 'user',
  flags: [{ key: 'full', label: 'All privileges', hint: '', badge: 'all privileges' }, { key: 'process', label: 'Process', hint: '', badge: '' }],
  host: true,
  validUntil: false,
  connLimit: true,
  membership: true,
  rules: '',
  new: { ...blank, host: '%', flags: { login: true } },
};
const role = (name: string, extra: Partial<Role> = {}): Role => ({ ...blank, id: name, name, ...extra });

describe('roles', () => {
  it('starts a new role from what the backend says', () => {
    const r = newRole(form);
    r.flags.login = false;
    expect(form.new.flags.login).toBe(true);
    expect(r).toMatchObject({ host: '%', password: '', clearPassword: false });
  });
  it('compares forms ignoring order, spaces and flags that are off', () => {
    const a = specOf(role('app', { memberOf: ['a', 'b'], flags: { full: true } }));
    expect(sameSpec(a, { ...a, name: ' app ', memberOf: ['b', 'a'], flags: { full: true, process: false } })).toBe(true);
    expect(sameSpec(a, { ...a, password: 'x' })).toBe(false);
    expect(sameSpec(a, { ...a, flags: {} })).toBe(false);
  });
  it('names MySQL accounts with their host', () => {
    expect(displayName(role('app', { host: '%' }))).toBe('app@%');
    expect(displayName(role('readers'))).toBe('readers');
  });
  it('filters by the shown name and hides built-in accounts unless asked', () => {
    const list = [role('app', { host: 'localhost' }), role('mysql.sys', { system: true }), role('admin')];
    expect(filterRoles(list, '', false).map(r => r.name)).toEqual(['app', 'admin']);
    expect(filterRoles(list, '@LOCAL', true).map(r => r.name)).toEqual(['app']);
  });
  it('shows the badges the engine marks', () => {
    expect(roleBadges(role('x', { flags: { full: true, process: true } }), form)).toEqual(['all privileges']);
  });
  it('turns grants into a script', () => {
    expect(grantsScript(['GRANT USAGE ON *.* TO `a`@`%`', 'GRANT ALL ON `shop`.* TO `a`@`%`;'])).toBe('GRANT USAGE ON *.* TO `a`@`%`;\nGRANT ALL ON `shop`.* TO `a`@`%`;');
  });
});
