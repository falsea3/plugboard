import type { Role, RoleSpec } from '../api/backend';

type Flag = 'login' | 'superuser' | 'createDB' | 'createRole' | 'inherit' | 'replication' | 'bypassRLS';

export const ATTRIBUTES: { key: Flag; label: string; hint: string }[] = [
  { key: 'login', label: 'Can log in', hint: 'A user rather than a group' },
  { key: 'superuser', label: 'Superuser', hint: 'Bypasses every permission check' },
  { key: 'createDB', label: 'Create databases', hint: '' },
  { key: 'createRole', label: 'Create roles', hint: 'Create, change and drop other roles' },
  { key: 'inherit', label: 'Inherit privileges', hint: 'Use the privileges of the roles it is a member of' },
  { key: 'replication', label: 'Replication', hint: 'Connect for streaming replication and backups' },
  { key: 'bypassRLS', label: 'Bypass row-level security', hint: '' },
];

export function newRole(): RoleSpec {
  return {
    name: '', login: true, superuser: false, createDB: false, createRole: false, inherit: true, replication: false, bypassRLS: false,
    connLimit: -1, validUntil: '', memberOf: [], system: false, current: false, password: '', clearPassword: false,
  };
}

export function specOf(role: Role): RoleSpec {
  return { ...role, memberOf: [...role.memberOf], password: '', clearPassword: false };
}

export function sameSpec(a: RoleSpec, b: RoleSpec): boolean {
  const key = (s: RoleSpec) => JSON.stringify({ ...s, name: s.name.trim(), validUntil: s.validUntil.trim(), memberOf: [...s.memberOf].sort() });
  return key(a) === key(b);
}

export function roleBadges(role: Role): string[] {
  const out: string[] = [];
  if (role.superuser) out.push('superuser');
  if (role.login) out.push('login');
  if (role.replication) out.push('replication');
  return out;
}

export function filterRoles(roles: Role[], query: string, system: boolean): Role[] {
  const q = query.trim().toLowerCase();
  return roles.filter(r => (system || !r.system) && (!q || r.name.toLowerCase().includes(q)));
}
