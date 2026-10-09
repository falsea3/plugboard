import type { Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { installKeys } from './bridgeKeys';

const engines = JSON.parse(readFileSync(new URL('./engines.json', import.meta.url), 'utf8'));

export async function installBridge(page: Page) {
  await page.addInitScript((engines: Record<string, unknown>) => {
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
      { name: 'id', type: 'bigint', nullable: false, default: "nextval('customers_id_seq'::regclass)", primaryKey: true, enum: null, kind: 'number' },
      { name: 'email', type: 'text', nullable: false, default: null, primaryKey: false, enum: null, kind: 'text' },
      { name: 'name', type: 'text', nullable: false, default: null, primaryKey: false, enum: null, kind: 'text' },
      { name: 'country', type: 'character(2)', nullable: true, default: null, primaryKey: false, enum: null, kind: 'text' },
      { name: 'plan', type: 'plan', nullable: false, default: "'free'::plan", primaryKey: false, enum: ['free', 'team', 'pro'], kind: '' },
      { name: 'is_active', type: 'boolean', nullable: false, default: 'true', primaryKey: false, enum: null, kind: 'bool' },
      { name: 'created_at', type: 'timestamp with time zone', nullable: false, default: 'now()', primaryKey: false, enum: null, kind: 'datetime' },
      { name: 'meta', type: 'jsonb', nullable: true, default: null, primaryKey: false, enum: null, kind: 'json' },
    ];
    const first = ['Ann', 'Bob', 'Chen', 'Dana', 'Eli', 'Fatima', 'Goran', 'Hana', 'Ivan', 'Jun', 'Kofi', 'Lena'];
    const last = ['Novak', 'Ito', 'Garcia', 'Smirnova', 'Okafor', 'Berg', 'Rossi', 'Kim', 'Haddad', 'Silva'];
    const countries = ['US', 'DE', 'JP', 'BR', 'NL', null, 'FR', 'IN'];
    const plans = ['free', 'team', 'pro', 'free', 'team'];
    const rows = Array.from({ length: 1200 }, (_, i) => {
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
        i % 5 === 4 ? null : `{"source": "${['ads', 'blog', 'direct'][i % 3]}", "visits": ${i * 3}}`,
      ];
    });

    const result = (cols: string[], data: unknown[][], extra = {}) => ({
      statement: '', columns: cols.map(name => ({ name, type: 'text', kind: 'text' })), rows: data, rowsAffected: 0,
      hasRows: true, truncated: false, pageable: true, hasMore: false, offset: 0, durationMs: 14.2, ...extra,
    });

    const settings = { theme: 'dark', pageSize: 300, editorFontSize: 13, confirmProdWrites: true, autoUpdate: false };
    const ok = <T>(v: T) => Promise.resolve(v);
    const fail = (e: { code: string; message: string; detail?: string }) => Promise.reject(new Error(JSON.stringify({ detail: '', ...e })));
    const saved = JSON.parse(localStorage.getItem('bridge-scripts') ?? 'null');
    const files: Record<string, Record<string, string>> = saved?.files ?? { shop: { 'monthly revenue': 'SELECT date_trunc(\'month\', created_at), sum(total)\nFROM orders\nGROUP BY 1;' } };
    const queryTabs: Record<string, { script?: string; saved: string }[]> = saved?.tabs ?? {};
    const keep = () => localStorage.setItem('bridge-scripts', JSON.stringify({ files, tabs: queryTabs }));
    const scripts = (id: string) => (files[id] ??= {});
    const exists = () => fail({ code: 'script_exists', message: 'a script with this name already exists' });

    (window as any).go = {
      api: {
        App: {
          AppInfo: () => ok({ name: 'Plugboard', version: '0.3.0', goVersion: 'go1.26.0', platform: 'darwin/arm64', dataDir: '/Users/demo/Library/Application Support/Plugboard', copyright: '© 2026 Relay Client' }),
          GetSettings: () => ok(settings),
          SaveSettings: (s: typeof settings) => ok(Object.assign(settings, s)),
          ListConnections: () => ok(connections),
          SaveConnection: (c: any) => ok(c),
          DeleteConnection: () => ok(undefined),
          TestConnection: () => ok({ ok: true, serverVersion: 'PostgreSQL 17.2', latencyMs: 18.4 }),
          Connect: (id: string) => {
            const c = connections.find(x => x.id === id)!;
            return ok({ session: { sessionId: `s-${id}`, connection: c, serverVersion: 'PostgreSQL 17.2', schemas: ['analytics', 'public'], defaultSchema: 'public', engine: engines[c.driver] } });
          },
          SetReadOnly: () => Promise.reject(new Error('not in the demo')),
          Disconnect: () => ok(undefined),
          ListTables: () => ok(tables),
          DescribeTable: () => ok(columns),
          LastCrash: () => ok((window as any).crashed ? '/Users/demo/Library/Application Support/Plugboard/logs/crash.log' : ''),
          OpenLogs: () => ok(undefined),
          Diagram: (_: string, schema: string) => {
            if (schema === 'analytics') {
              return fail({ code: 'timeout', message: "db.internal:5432 didn't answer in time.", detail: 'read tcp 10.0.0.2:51234->10.0.0.9:5432: i/o timeout' });
            }
            const col = (name: string, type: string, primaryKey = false) =>
              ({ name, type, nullable: !primaryKey, default: null, primaryKey, enum: null, kind: type === 'text' ? 'text' : 'number' });
            const rel = (table: string, column: string, refTable: string, onDelete = 'NO ACTION') =>
              ({ name: `${table}_${column}_fkey`, table, columns: [column], refSchema: schema, refTable, refColumns: ['id'], onDelete, onUpdate: 'NO ACTION' });
            return ok({
              schema,
              tables: [
                { name: 'customers', kind: 'table', columns },
                { name: 'invoices', kind: 'table', columns: [col('id', 'bigint', true), col('order_id', 'bigint'), col('amount', 'numeric(12,2)'), col('issued_at', 'timestamp with time zone')] },
                { name: 'order_items', kind: 'table', columns: [col('id', 'bigint', true), col('order_id', 'bigint'), col('product_id', 'bigint'), col('qty', 'integer')] },
                { name: 'orders', kind: 'table', columns: [col('id', 'bigint', true), col('customer_id', 'bigint'), col('total', 'numeric(12,2)'), col('status', 'text'), col('created_at', 'timestamp with time zone')] },
                { name: 'products', kind: 'table', columns: [col('id', 'bigint', true), col('sku', 'text'), col('name', 'text'), col('price', 'numeric(10,2)')] },
                { name: 'refunds', kind: 'table', columns: [col('id', 'bigint', true), col('invoice_id', 'bigint'), col('reason', 'text')] },
                { name: 'shipments', kind: 'table', columns: [col('id', 'bigint', true), col('order_id', 'bigint'), col('carrier', 'text'), col('tracking', 'text')] },
                { name: 'order_summary', kind: 'view', columns: [col('order_id', 'bigint'), col('customer', 'text'), col('total', 'numeric')] },
              ],
              relations: [
                rel('invoices', 'order_id', 'orders'),
                rel('order_items', 'order_id', 'orders', 'CASCADE'),
                rel('order_items', 'product_id', 'products'),
                rel('orders', 'customer_id', 'customers', 'RESTRICT'),
                rel('refunds', 'invoice_id', 'invoices'),
                rel('shipments', 'order_id', 'orders', 'CASCADE'),
              ],
            });
          },
          ListObjects: (_: string, schema: string) =>
            ok([
              { schema, name: 'add_tax', kind: 'function', detail: 'amount numeric' },
              { schema, name: 'plan', kind: 'enum', values: ['free', 'team', 'pro'] },
              { schema, name: 'order_seq', kind: 'sequence' },
              { schema, name: 'orders_touch', kind: 'trigger', detail: 'orders' },
              { schema: 'public', name: 'pgcrypto', kind: 'extension', detail: '1.3' },
            ]),
          ObjectDDL: (_: string, o: { schema: string; name: string; kind: string }) =>
            ok(o.kind === 'table' ? `CREATE TABLE "${o.schema}"."${o.name}" (\n  "id" serial,\n  CONSTRAINT ${o.name}_pkey PRIMARY KEY (id)\n);` : `CREATE OR REPLACE FUNCTION ${o.schema}.${o.name}(amount numeric)\n RETURNS numeric\n LANGUAGE sql\nAS $function$ select amount * 1.2 $function$;`),
          Indexes: () =>
            ok([
              { name: 'customers_pkey', columns: ['id'], unique: true, primary: true, method: 'btree', where: '', definition: 'CREATE UNIQUE INDEX customers_pkey ON public.customers USING btree (id)' },
              { name: 'customers_email_key', columns: ['email'], unique: true, primary: false, method: 'btree', where: '', definition: 'CREATE UNIQUE INDEX customers_email_key ON public.customers USING btree (email)' },
              { name: 'customers_active_country', columns: ['country', 'plan'], unique: false, primary: false, method: 'btree', where: 'is_active', definition: 'CREATE INDEX customers_active_country ON public.customers USING btree (country, plan) WHERE is_active' },
            ]),
          Relations: (_: string, schema: string) =>
            ok([{ name: 'customers_country_fkey', table: 'customers', columns: ['country'], refSchema: schema, refTable: 'countries', refColumns: ['code'], onDelete: 'NO ACTION', onUpdate: 'NO ACTION' }]),
          FetchTablePage: (_: string, q: { limit: number; after?: unknown[] }) => {
            (window as any).lastPage = q;
            const from = q.after ? rows.findIndex(r => r[0] === q.after![0]) + 1 : 0;
            const chunk = rows.slice(from, from + q.limit);
            return ok({ result: { ...result(columns.map(c => c.name), chunk, { durationMs: from === 0 ? 14.2 : 9.6 }), columns: columns.map(c => ({ name: c.name, type: c.type, kind: c.kind })), pageable: false },
                        hasMore: from + chunk.length < rows.length, defaultOrder: ['id'], hasPrev: from > 0, keyset: true, offset: -1 });
          },
          CountRows: () => ok({ count: 24813, exact: false, known: true }),
          WriteStatements: (_: string, script: string) => ok(/\b(insert|update|delete|drop|alter)\b/i.test(script) ? [script] : []),
          CheckSyntax: (_: string, script: string) =>
            ok(script.split(';').map(t => t.trim()).filter(Boolean)
              .map((t, index) => ({ index, position: t.search(/\bfron\b/), message: 'syntax error at or near "fron"' }))
              .filter(p => p.position >= 0)),
          RunQuery: () => ok({
            results: [result(['country', 'customers', 'revenue'], [
              ['US', 6412, '1284390.50'], ['DE', 3180, '702118.00'], ['JP', 2957, '655004.75'], ['BR', 2210, '318760.20'],
              ['NL', 1874, '402551.10'], ['FR', 1602, '351920.00'], ['IN', 1544, '189402.35'],
            ])],
            errorIndex: -1, errorPosition: -1, cancelled: false, rolledBack: false,
          }),
          RunMore: () => ok(result([], [])),
          CancelQuery: () => ok(undefined),
          ApplyChanges: (_: string, cs: unknown) => {
            (window as any).lastChanges = cs;
            return ok({ applied: 0, failedIndex: -1, partial: false, cancelled: false });
          },
          PreviewChanges: () => ok([]),
          ApplyStructure: () => ok({ applied: 1, failedIndex: -1, partial: false, cancelled: false }),
          PreviewStructure: (_: string, sc: { table: string; changes: { kind: string; name?: string; type?: string }[] }) =>
            ok(sc.changes.map(c => `ALTER TABLE "public"."${sc.table}" ${c.kind === 'insert' ? `ADD COLUMN "${c.name}" ${c.type}` : c.type ? `ALTER COLUMN "${(c as any).column}" TYPE ${c.type}` : '…'}`)),
          CreateIndexSQL: (_: string, schema: string, table: string, idx: { name: string; columns: string[]; unique: boolean }) =>
            ok(`CREATE ${idx.unique ? 'UNIQUE ' : ''}INDEX "${idx.name}" ON "${schema}"."${table}" (${idx.columns.map(c => `"${c}"`).join(', ')});`),
          DropIndexSQL: (_: string, schema: string, _t: string, name: string) => ok(`DROP INDEX "${schema}"."${name}";`),
          RenameTableSQL: (_: string, schema: string, from: string, to: string) => ok(`ALTER TABLE "${schema}"."${from}" RENAME TO "${to}";`),
          ListScripts: (id: string) => ok(Object.keys(scripts(id)).sort((a, b) => a.localeCompare(b))),
          ReadScript: (id: string, name: string) => ok(scripts(id)[name]),
          CreateScript: (id: string, name: string, sql: string) => (name in scripts(id) ? exists() : ok(void ((scripts(id)[name] = sql), keep()))),
          WriteScript: (id: string, name: string, sql: string) => ok(void ((scripts(id)[name] = sql), keep())),
          RenameScript: (id: string, from: string, to: string) => {
            if (to in scripts(id)) return exists();
            scripts(id)[to] = scripts(id)[from];
            delete scripts(id)[from];
            return ok(void keep());
          },
          DeleteScript: (id: string, name: string) => ok(void (delete scripts(id)[name], keep())),
          QueryTabs: (id: string) =>
            ok((queryTabs[id] ?? []).map(t => (t.script ? (t.script in scripts(id) ? { ...t, saved: scripts(id)[t.script] } : { ...t, script: '', saved: '' }) : t))),
          SaveQueryTabs: (id: string, tabs: { script?: string; saved: string }[]) => ok(void ((queryTabs[id] = tabs), keep())),
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
  }, engines);
  await installKeys(page, engines.redis);
}
