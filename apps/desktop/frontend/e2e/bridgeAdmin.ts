import type { Page } from '@playwright/test';

export async function installAdmin(page: Page) {
  await page.addInitScript(() => {
    const App = (window as any).go.api.App;
    const ok = <T>(v: T) => Promise.resolve(v);
    const fail = (code: string, message: string) => Promise.reject(new Error(JSON.stringify({ code, message, detail: '' })));
    const dir = '/var/lib/postgresql/data';
    const disk: Record<string, string> = {
      [`${dir}/postgresql.conf`]: "# PostgreSQL configuration\nlisten_addresses = '*'\nmax_connections = 100\nshared_buffers = 128MB\n",
      [`${dir}/pg_hba.conf`]: 'local all all trust\nhost all all all scram-sha-256\n',
    };
    const role = (name: string, extra = {}) => ({
      name, login: false, superuser: false, createDB: false, createRole: false, inherit: true, replication: false, bypassRLS: false,
      connLimit: -1, validUntil: '', memberOf: [] as string[], system: false, current: false, ...extra,
    });
    let roles = [
      role('app', { login: true, memberOf: ['readers'] }),
      role('readers'),
      role('relay', { login: true, superuser: true, current: true }),
      role('pg_monitor', { system: true }),
    ];
    const w = window as any;
    Object.assign(App, {
      ConfigFiles: () => ok(['postgresql.conf', 'pg_hba.conf', 'pg_ident.conf'].map(name => ({ name, path: `${dir}/${name}` }))),
      ReadConfig: (_: string, path: string) => (path in disk ? ok({ path, text: disk[path], via: 'sql' }) : fail('error', `could not open file "${path}" for reading: No such file or directory`)),
      WriteConfig: (_: string, path: string, text: string) => {
        disk[path] = text;
        w.lastConfig = { path, text };
        const bad = text.split('\n').findIndex(l => /=\s*$/.test(l));
        if (bad >= 0) return ok({ via: 'sql', reloaded: false, problems: [{ line: bad + 1, message: 'syntax error in file' }], restart: [] });
        return ok({ via: 'sql', reloaded: true, problems: [], restart: /shared_buffers = 256MB/.test(text) ? ['shared_buffers'] : [] });
      },
      Roles: () => ok(roles),
      RoleSQL: (_: string, from: string, to: { name: string; password: string }) =>
        ok(from ? [`ALTER ROLE "${from}" WITH LOGIN`] : [`CREATE ROLE "${to.name}" WITH LOGIN${to.password ? " PASSWORD '********'" : ''}`]),
      SaveRole: (_: string, from: string, to: any) => {
        w.lastRole = { from, to };
        const { password: _p, clearPassword: _c, ...saved } = to;
        roles = from ? roles.map(r => (r.name === from ? saved : r)) : [...roles, saved];
        return ok(undefined);
      },
      DropRole: (_: string, name: string) => {
        roles = roles.filter(r => r.name !== name);
        return ok(undefined);
      },
    });
  });
}
