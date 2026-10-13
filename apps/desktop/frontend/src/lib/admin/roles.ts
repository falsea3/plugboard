import type { Role, RoleForm, RoleSpec } from '../api/backend';

export function specOf(role: Role): RoleSpec {
  return { ...role, flags: { ...role.flags }, memberOf: [...role.memberOf], grants: [...role.grants], password: '', clearPassword: false };
}

export function newRole(form: RoleForm): RoleSpec {
  return specOf(form.new);
}

export function sameSpec(a: RoleSpec, b: RoleSpec): boolean {
  const flags = (s: RoleSpec) => Object.keys(s.flags).filter(k => s.flags[k]).sort();
  const key = (s: RoleSpec) =>
    JSON.stringify({ ...s, name: s.name.trim(), host: s.host.trim(), validUntil: s.validUntil.trim(), flags: flags(s), memberOf: [...s.memberOf].sort() });
  return key(a) === key(b);
}

export function displayName(role: Pick<Role, 'name' | 'host'>): string {
  return role.host ? `${role.name}@${role.host}` : role.name;
}

export function roleBadges(role: Role, form: RoleForm): string[] {
  return form.flags.filter(f => f.badge && role.flags[f.key]).map(f => f.badge);
}

export function filterRoles(roles: Role[], query: string, system: boolean): Role[] {
  const q = query.trim().toLowerCase();
  return roles.filter(r => (system || !r.system) && (!q || displayName(r).toLowerCase().includes(q)));
}

export function grantsScript(grants: string[]): string {
  return grants.map(g => (g.trimEnd().endsWith(';') ? g : `${g};`)).join('\n');
}
