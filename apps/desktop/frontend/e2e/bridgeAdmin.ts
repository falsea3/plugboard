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
      postgres: { noun: 'role', host: false, validUntil: true, connLimit: true, membership: true, rules: '', flags: [flag('login', 'Can log in', 'login'), flag('superuser', 'Superuser', 'superuser'), flag('createDB', 'Create databases'), flag('inherit', 'Inherit privileges')] },
      mysql: { noun: 'user', host: true, validUntil: false, connLimit: true, membership: true, rules: '', flags: [flag('full', 'All privileges', 'all privileges'), flag('process', 'See all sessions'), flag('locked', 'Account locked', 'locked')] },
    };
    forms.cassandra = { noun: 'role', host: false, validUntil: false, connLimit: false, membership: true, rules: '', flags: [flag('login', 'Can log in', 'login'), flag('superuser', 'Superuser', 'superuser')] };
    forms.clickhouse = { noun: 'user', host: false, validUntil: false, connLimit: false, membership: true, rules: '', flags: [flag('full', 'All privileges', 'all privileges')] };
    forms.redis = { noun: 'user', host: false, validUntil: false, connLimit: false, membership: false, rules: 'ACL rules: ~cache:* +@read', flags: [flag('disabled', 'Disabled', 'off'), flag('nopass', 'No password', 'nopass')] };
    const role = (name: string, extra: Record<string, unknown> = {}) => ({
      id: name, name, host: '', flags: {}, connLimit: -1, validUntil: '', memberOf: [] as string[], grants: [] as string[], rules: '', kind: '', grantable: true, system: false, current: false, ...extra,
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
      cassandra: [role('relay', { flags: { login: true, superuser: true }, current: true }), role('readers')],
      clickhouse: [role('default', { system: true, current: true, grantable: false }), role('analysts', { id: 'role:analysts', kind: 'role' })],
      redis: [role('default', { rules: '~* &* +@all', current: true, grantable: false })],
    };
    forms.postgres.new = role('', { flags: { login: true, inherit: true } });
    forms.mysql.new = role('', { host: '%' });
    forms.redis.new = role('', { grantable: false });
    forms.cassandra.new = role('', { flags: { login: true } });
    forms.clickhouse.new = role('', { grantable: false });
    const driver = (id: string) => (id === 's-analytics' ? 'mysql' : id === 's-cache' ? 'redis' : id === 's-ledger' ? 'cassandra' : id === 's-warehouse' ? 'clickhouse' : 'postgres');
    let live = '# Live settings.\nmaxmemory 0\nmaxmemory-policy noeviction\n';
    Object.assign(App, {
      ConfigFiles: (s: string) =>
        ok(driver(s) === 'cassandra' ? [{ name: 'Current settings', path: 'system_views.settings' }, { name: 'cassandra.yaml', path: '/etc/cassandra/cassandra.yaml' }] : driver(s) === 'redis' ? [{ name: 'Live settings', path: 'CONFIG GET *' }] : ['postgresql.conf', 'pg_hba.conf', 'pg_ident.conf'].map(name => ({ name, path: `${dir}/${name}` }))),
      ReadConfig: (_: string, path: string) => (path === 'system_views.settings' ? ok({ path, text: '# Read-only.\nauthenticator: PasswordAuthenticator\n', via: 'live', readOnly: true }) : path === 'CONFIG GET *' ? ok({ path, text: live, via: 'live' }) : path in disk ? ok({ path, text: disk[path], via: 'sql' }) : fail('error', `could not open file "${path}" for reading: No such file or directory`)),
      WriteConfig: (_: string, path: string, text: string) => {
        if (path === 'CONFIG GET *') {
          live = text;
          const bad = text.split('\n').findIndex(l => l.startsWith('no-such'));
          return ok({ via: 'live', reloaded: bad < 0, problems: bad < 0 ? [] : [{ line: bad + 1, message: 'ERR Unknown option' }], restart: [], note: bad < 0 ? 'Applied, but not written to a config file — the change lasts until Redis restarts.' : '' });
        }
        disk[path] = text;
        w.lastConfig = { path, text };
        const bad = text.split('\n').findIndex(l => /=\s*$/.test(l));
        if (bad >= 0) return ok({ via: 'sql', reloaded: false, problems: [{ line: bad + 1, message: 'syntax error in file' }], restart: [], note: '' });
        return ok({ via: 'sql', reloaded: true, problems: [], restart: /shared_buffers = 256MB/.test(text) ? ['shared_buffers'] : [], note: '' });
      },
      Roles: (s: string) => ok({ form: forms[driver(s)], roles: store[driver(s)] }),
      RoleSQL: (_: string, from: string, to: { name: string; password: string }) =>
        ok(driver(_) === 'redis' ? [`ACL SETUSER ${to.name} on resetpass >******** ${(to as any).rules}`] : from ? [`ALTER ROLE "${from}" WITH LOGIN`] : [`CREATE ROLE "${to.name}" WITH LOGIN${to.password ? " PASSWORD '********'" : ''}`]),
      SaveRole: (s: string, from: string, to: any) => {
        w.lastRole = { from, to };
        const { password: _p, clearPassword: _c, ...saved } = to;
        saved.id = driver(s) === 'mysql' ? `\`${saved.name}\`@\`${saved.host || '%'}\`` : saved.name;
        const list = store[driver(s)];
        store[driver(s)] = from ? list.map(r => (r.id === from ? saved : r)) : [...list, saved];
        return ok(saved.id);
      },
      ReadCell: (_: string, q: { column: string; key: Record<string, unknown> }) => ((w.lastReadCell = q), ok(`whole value of ${q.column} in row ${q.key.id}: ` + 'x'.repeat(70000))),
      SaveExport: (name: string, data: string) => ((w.lastExport = { name, data }), ok(`/Users/demo/Desktop/${name}`)),
      DropRole: (s: string, id: string) => {
        store[driver(s)] = store[driver(s)].filter(r => r.id !== id);
        return ok(undefined);
      },
    });
  });
}
