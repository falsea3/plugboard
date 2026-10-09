import type { Page } from '@playwright/test';

export async function installKeys(page: Page, engine: unknown) {
  await page.addInitScript((engine: unknown) => {
    const App = (window as any).go.api.App;
    const ok = <T>(v: T) => Promise.resolve(v);
    const fail = (code: string, message: string) => Promise.reject(new Error(JSON.stringify({ code, message, detail: '' })));
    const cache = {
      id: 'cache', name: 'Cache', driver: 'redis', host: 'cache.internal', port: 6379, user: '', savePassword: true, database: '0',
      file: '', sslMode: '', env: 'staging', color: '', readOnly: false,
      ssh: { enabled: false, host: '', port: 22, user: '', auth: 'password', keyFile: '' },
    };
    type Item = { field: string; value: string; score?: string };
    const store: Record<string, { type: string; ttl: number; text?: string; items?: Item[] }> = {
      'shop:customer:1': { type: 'hash', ttl: -1, items: [{ field: 'name', value: 'Ann Novak' }, { field: 'plan', value: 'pro' }] },
      'shop:customer:2': { type: 'hash', ttl: -1, items: [{ field: 'name', value: 'Bob Ito' }] },
      'shop:config:currency': { type: 'string', ttl: -1, text: 'USD' },
      'shop:session:8f2a': { type: 'string', ttl: 86400, text: '{"customer": 1, "cart": [1, 3]}' },
      'shop:queue:emails': { type: 'list', ttl: -1, items: ['welcome:1', 'receipt:1042'].map((value, i) => ({ field: String(i), value })) },
      'shop:leaderboard': { type: 'zset', ttl: -1, items: [{ field: 'BR', value: 'BR', score: '318760.2' }, { field: 'US', value: 'US', score: '1284390.5' }] },
    };
    (window as any).redisStore = store;
    const glob = (p: string) => new RegExp('^' + p.replace(/[.+^${}()|\\]/g, '\\$&').replace(/\*/g, '.*').replace(/\?/g, '.') + '$');
    const listConnections = App.ListConnections;
    const connect = App.Connect;
    const runQuery = App.RunQuery;
    const writeStatements = App.WriteStatements;
    Object.assign(App, {
      ListConnections: async () => [...(await listConnections()), cache],
      Connect: (id: string, secrets: unknown) =>
        id === 'cache'
          ? ok({ session: { sessionId: 's-cache', connection: cache, serverVersion: 'Redis 7.4.0', schemas: Array.from({ length: 16 }, (_, i) => String(i)), defaultSchema: '0', engine } })
          : connect(id, secrets),
      KeyCounts: () => ok({ '0': Object.keys(store).length, '9': 3 }),
      ScanKeys: (_: string, _db: string, pattern: string) =>
        ok({ keys: Object.keys(store).filter(k => glob(pattern).test(k)).map(k => ({ name: k, key: k, type: store[k].type })), cursor: '0', done: true }),
      ReadKey: (_: string, _db: string, key: string) => {
        const v = store[key];
        if (!v) return fail('key_gone', 'this key no longer exists — it may have expired or been deleted');
        const size = v.type === 'string' ? (v.text ?? '').length : (v.items ?? []).length;
        return ok({ key, type: v.type, ttl: v.ttl, size, text: v.text ?? '', items: v.items ?? [], cursor: '0', done: true });
      },
      EditKey: (_: string, _db: string, e: { key: string; op: string; field: string; value: string; score: string; index: number }) => {
        (window as any).lastKeyEdit = e;
        const v = store[e.key];
        if (e.op.startsWith('create:')) {
          if (v) return fail('exists', 'that name is already taken');
          const type = e.op.slice(7);
          store[e.key] = { type, ttl: -1, text: type === 'string' ? e.value : undefined, items: type === 'string' ? undefined : [{ field: e.field || e.value, value: e.value, score: e.score }] };
          return ok(undefined);
        }
        if (!v) return fail('key_gone', 'this key no longer exists — it may have expired or been deleted');
        const at = (v.items ?? []).findIndex(it => it.field === e.field);
        if (e.op === 'set') v.text = e.value;
        else if (e.op === 'hset') v.items![at].value = e.value;
        else if (e.op === 'hadd') v.items!.push({ field: e.field, value: e.value });
        else if (e.op === 'hdel') v.items!.splice(at, 1);
        else if (e.op === 'expire') v.ttl = e.index > 0 ? e.index : -1;
        else if (e.op === 'rename') {
          if (store[e.value]) return fail('exists', 'that name is already taken');
          store[e.value] = v;
          delete store[e.key];
        } else if (e.op === 'delete') delete store[e.key];
        return ok(undefined);
      },
      RunQuery: (sid: string, qid: string, script: string) => {
        if (sid !== 's-cache') return runQuery(sid, qid, script);
        const lines = script.split('\n').map(l => l.trim()).filter(Boolean);
        const results = lines.map(line => {
          const [cmd, key] = line.split(/\s+/);
          const text = cmd.toUpperCase() === 'GET' ? (store[key]?.text ?? null) : 'OK';
          return { statement: line, columns: [{ name: 'value', type: '', kind: 'text' }], rows: [[text]], rowsAffected: 0, hasRows: true, truncated: false, pageable: false, hasMore: false, offset: 0, durationMs: 0.4 };
        });
        return ok({ results, errorIndex: -1, errorPosition: -1, cancelled: false, rolledBack: false });
      },
      WriteStatements: (sid: string, script: string) =>
        sid === 's-cache' ? ok(script.split('\n').filter(l => /^\s*(set|del|hset|flushdb)\b/i.test(l))) : writeStatements(sid, script),
    });
  }, engine);
}
