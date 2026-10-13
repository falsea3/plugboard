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
    const w = window as any;
    const flag = (key: string, label: string, badge = '') => ({ key, label, hint: '', badge });
    const forms: Record<string, any> = {
      postgres: { noun: 'role', host: false, validUntil: true, flags: [flag('login', 'Can log in', 'login'), flag('superuser', 'Superuser', 'superuser'), flag('createDB', 'Create databases'), flag('inherit', 'Inherit privileges')] },
      mysql: { noun: 'user', host: true, validUntil: false, flags: [flag('full', 'All privileges', 'all privileges'), flag('process', 'See all sessions'), flag('locked', 'Account locked', 'locked')] },
    };
    const role = (name: string, extra: Record<string, unknown> = {}) => ({
      id: name, name, host: '', flags: {}, connLimit: -1, validUntil: '', memberOf: [] as string[], grants: [] as string[], grantable: true, system: false, current: false, ...extra,
    });
    const store: Record<string, any[]> = {
      postgres: [
        role('app', { flags: { login: true, inherit: true }, memberOf: ['readers'] }),
        role('readers', { flags: { inherit: true } }),
        role('relay', { flags: { login: true, superuser: true }, current: true }),
        role('pg_monitor', { system: true }),
      ],
      mysql: [
        role('reporting', { id: '`reporting`@`%`', host: '%', grants: ['GRANT USAGE ON *.* TO `reporting`@`%`', 'GRANT SELECT ON `events`.* TO `reporting`@`%`'] }),
        role('root', { id: '`root`@`localhost`', host: 'localhost', flags: { full: true }, current: true }),
      ],
    };
    forms.postgres.new = role('', { flags: { login: true, inherit: true } });
    forms.mysql.new = role('', { host: '%' });
    const driver = (id: string) => (id === 's-analytics' ? 'mysql' : 'postgres');
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
      Roles: (s: string) => ok({ form: forms[driver(s)], roles: store[driver(s)] }),
      RoleSQL: (_: string, from: string, to: { name: string; password: string }) =>
        ok(from ? [`ALTER ROLE "${from}" WITH LOGIN`] : [`CREATE ROLE "${to.name}" WITH LOGIN${to.password ? " PASSWORD '********'" : ''}`]),
      SaveRole: (s: string, from: string, to: any) => {
        w.lastRole = { from, to };
        const { password: _p, clearPassword: _c, ...saved } = to;
        saved.id = driver(s) === 'mysql' ? `\`${saved.name}\`@\`${saved.host || '%'}\`` : saved.name;
        const list = store[driver(s)];
        store[driver(s)] = from ? list.map(r => (r.id === from ? saved : r)) : [...list, saved];
        return ok(saved.id);
      },
      SaveDiagram: (name: string, data: string) => ((w.lastExport = { name, data }), ok(`/Users/demo/Desktop/${name}`)),
      DropRole: (s: string, id: string) => {
        store[driver(s)] = store[driver(s)].filter(r => r.id !== id);
        return ok(undefined);
      },
    });
  });
}
