import type { Page } from '@playwright/test';

/**
 * Stands in for the Go backend (window.go.api.App) with a made-up "shop"
 * database, so the UI runs in a plain browser: no Wails, no servers, and
 * screenshots that never show anyone's real data.
 */
export async function installBridge(page: Page) {
  await page.addInitScript(() => {
    const connections = [
      {
        id: 'shop', name: 'Shop', driver: 'postgres', host: 'db.internal', port: 5432, user: 'app', savePassword: true,
        database: 'shop', file: '', sslMode: 'require', env: 'staging', color: '', readOnly: false,
        ssh: { enabled: true, host: 'bastion.example.com', port: 22, user: 'deploy', auth: 'agent', keyFile: '' },
      },
      {
        id: 'analytics', name: 'Analytics', driver: 'mysql', host: 'analytics.example.com', port: 3306, user: 'reporting',
        savePassword: true, database: 'events', file: '', sslMode: 'verify-full', env: 'prod', color: '', readOnly: true,
        ssh: { enabled: false, host: '', port: 22, user: '', auth: 'password', keyFile: '' },
      },
      {
        id: 'music', name: 'Music library', driver: 'sqlite', host: '', port: 0, user: '', savePassword: true, database: '',
        file: '/Users/demo/Music/library.sqlite3', sslMode: '', env: 'local', color: '', readOnly: false,
        ssh: { enabled: false, host: '', port: 22, user: '', auth: 'password', keyFile: '' },
      },
    ];

    const tables = ['customers', 'invoices', 'order_items', 'orders', 'products', 'refunds', 'shipments']
      .map(name => ({ schema: 'public', name, kind: 'table' }))
      .concat([{ schema: 'public', name: 'order_summary', kind: 'view' }]);

    const columns = [
      { name: 'id', type: 'bigint', nullable: false, default: "nextval('customers_id_seq'::regclass)", primaryKey: true, enum: null, binary: false },
      { name: 'email', type: 'text', nullable: false, default: null, primaryKey: false, enum: null, binary: false },
      { name: 'name', type: 'text', nullable: false, default: null, primaryKey: false, enum: null, binary: false },
      { name: 'country', type: 'character(2)', nullable: true, default: null, primaryKey: false, enum: null, binary: false },
      { name: 'plan', type: 'plan', nullable: false, default: "'free'::plan", primaryKey: false, enum: ['free', 'team', 'pro'], binary: false },
      { name: 'is_active', type: 'boolean', nullable: false, default: 'true', primaryKey: false, enum: null, binary: false },
      { name: 'created_at', type: 'timestamp with time zone', nullable: false, default: 'now()', primaryKey: false, enum: null, binary: false },
    ];
    const first = ['Ann', 'Bob', 'Chen', 'Dana', 'Eli', 'Fatima', 'Goran', 'Hana', 'Ivan', 'Jun', 'Kofi', 'Lena'];
    const last = ['Novak', 'Ito', 'Garcia', 'Smirnova', 'Okafor', 'Berg', 'Rossi', 'Kim', 'Haddad', 'Silva'];
    const countries = ['US', 'DE', 'JP', 'BR', 'NL', null, 'FR', 'IN'];
    const plans = ['free', 'team', 'pro', 'free', 'team'];
    const rows = Array.from({ length: 300 }, (_, i) => {
      const n = i + 1;
      const name = `${first[i % first.length]} ${last[(i * 7) % last.length]}`;
      const day = String(1 + (i % 28)).padStart(2, '0');
      return [
        n,
        `${name.toLowerCase().replace(' ', '.')}${n}@example.com`,
        name,
        countries[i % countries.length],
        plans[(i * 3) % plans.length],
        i % 9 !== 0,
        `2026-0${1 + (i % 9)}-${day} ${String(8 + (i % 12)).padStart(2, '0')}:${String((i * 17) % 60).padStart(2, '0')}:00+00:00`,
      ];
    });

    const result = (cols: string[], data: unknown[][], extra = {}) => ({
      statement: '', columns: cols.map(name => ({ name, type: 'text' })), rows: data, rowsAffected: 0,
      hasRows: true, truncated: false, pageable: true, hasMore: false, offset: 0, durationMs: 14.2, ...extra,
    });

    const settings = { theme: 'dark', pageSize: 300, editorFontSize: 13, confirmProdWrites: true, autoUpdate: false };
    const ok = <T>(v: T) => Promise.resolve(v);

    (window as any).go = {
      api: {
        App: {
          AppInfo: () => ok({ name: 'Relay DB', version: '0.1.0', goVersion: 'go1.26.0', platform: 'darwin/arm64', dataDir: '/Users/demo/Library/Application Support/Relay DB', copyright: '© 2026 Relay Client' }),
          GetSettings: () => ok(settings),
          SaveSettings: (s: typeof settings) => ok(Object.assign(settings, s)),
          ListConnections: () => ok(connections),
          SaveConnection: (c: any) => ok(c),
          DeleteConnection: () => ok(undefined),
          TestConnection: () => ok({ ok: true, serverVersion: 'PostgreSQL 17.2', latencyMs: 18.4 }),
          Connect: (id: string) => {
            const c = connections.find(x => x.id === id)!;
            return ok({ session: { sessionId: `s-${id}`, connection: c, serverVersion: 'PostgreSQL 17.2', schemas: ['analytics', 'public'], defaultSchema: 'public' } });
          },
          SetReadOnly: () => Promise.reject('not in the demo'),
          Disconnect: () => ok(undefined),
          ListTables: () => ok(tables),
          DescribeTable: () => ok(columns),
          FetchTablePage: (_: string, q: { limit: number }) =>
            ok({ result: { ...result(columns.map(c => c.name), rows.slice(0, q.limit)), columns: columns.map(c => ({ name: c.name, type: c.type })), pageable: false },
                 hasMore: true, defaultOrder: ['id'], hasPrev: false, keyset: true, offset: -1 }),
          CountRows: () => ok({ count: 24813, exact: false, known: true }),
          WriteStatements: (_: string, script: string) => ok(/\b(insert|update|delete|drop|alter)\b/i.test(script) ? [script] : []),
          RunQuery: () => ok({
            results: [result(['country', 'customers', 'revenue'], [
              ['US', 6412, '1284390.50'], ['DE', 3180, '702118.00'], ['JP', 2957, '655004.75'], ['BR', 2210, '318760.20'],
              ['NL', 1874, '402551.10'], ['FR', 1602, '351920.00'], ['IN', 1544, '189402.35'],
            ])],
            errorIndex: -1, cancelled: false, rolledBack: false,
          }),
          RunMore: () => ok(result([], [])),
          CancelQuery: () => ok(undefined),
          ApplyChanges: () => ok({ applied: 0, failedIndex: -1 }),
          PreviewChanges: () => ok([]),
          ChooseSQLiteFile: () => ok(''),
          ChooseSSHKeyFile: () => ok(''),
          TrustHostKey: () => ok(undefined),
          OpenDataFolder: () => ok(undefined),
          CheckForUpdate: () => ok({}),
          InstallUpdate: () => ok('0.1.0'),
          RestartApp: () => ok(undefined),
        },
      },
    };
    (window as any).runtime = {
      EventsOn: () => () => {},
      ClipboardSetText: () => ok(true),
      BrowserOpenURL: () => {},
    };
  });
}
