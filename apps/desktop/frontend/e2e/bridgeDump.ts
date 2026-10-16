import type { Page } from '@playwright/test';

export async function installDump(page: Page) {
  await page.addInitScript(() => {
    const w = window as any;
    const App = w.go.api.App;
    const ok = <T>(v: T) => Promise.resolve(v);
    const fail = (code: string, message: string) => Promise.reject(new Error(JSON.stringify({ code, message, detail: '' })));
    const listeners: Record<string, ((...data: unknown[]) => void)[]> = {};
    const emit = (name: string, ...data: unknown[]) => (listeners[name] ?? []).forEach(cb => cb(...data));
    w.runtime.EventsOn = (name: string, cb: (...data: unknown[]) => void) => {
      (listeners[name] ??= []).push(cb);
      return () => (listeners[name] = listeners[name].filter(x => x !== cb));
    };
    const jobs: Record<string, () => void> = {};
    const cancel = App.CancelQuery;
    App.CancelQuery = (id: string) => (jobs[id] ? ok(jobs[id]()) : cancel(id));
    const later = (job: string, steps: (() => void)[], done: () => unknown) =>
      new Promise((resolve, reject) => {
        let i = 0;
        const timer = setInterval(() => {
          if (i < steps.length) return steps[i++]();
          clearInterval(timer);
          delete jobs[job];
          resolve(done());
        }, w.dumpPause ?? 60);
        jobs[job] = () => {
          clearInterval(timer);
          delete jobs[job];
          reject(new Error(JSON.stringify({ code: 'canceled', message: 'Stopped.', detail: '' })));
        };
      });

    App.ExportDump = (_: string, job: string, name: string, opts: unknown) => {
      w.lastDump = { name, opts };
      const p = (phase: string, done: number, rows: number, object = '') => () => emit('dump:progress', job, { phase, object, tables: 7, done, rows, bytes: rows * 52 });
      return later(job, [p('structure', 0, 0), p('data', 2, 8120, 'public.orders'), p('data', 6, 24813, 'public.shipments')], () => ({
        path: `/Users/demo/Documents/${name}`, bytes: 1290240, rows: 24813, tables: 7,
      }));
    };
    App.PickDumpFile = () => ok(w.nextDumpFile ?? { path: '/Users/demo/Downloads/shop.sql.gz', name: 'shop.sql.gz', size: 48213 });
    App.ImportDump = (_: string, job: string, path: string, opts: unknown) => {
      w.lastImport = { path, opts };
      if (path.endsWith('.dump')) return fail('bad_value', 'This is a pg_dump archive in custom format.');
      const p = (bytes: number, statements: number) => () => emit('import:progress', job, { bytes, total: 48213, statements, errors: 0, current: 'COPY public.orders (id, customer_id, total) FROM stdin' });
      const broken = path.includes('broken');
      return later(job, [p(12000, 40), p(48213, 112)], () => ({
        statements: 112, rows: broken ? 0 : 24813, skipped: ['\\restrict 3c2SycFE', '\\unrestrict 3c2SycFE'], stopped: broken, rolledBack: broken,
        errors: broken ? [{ index: 57, statement: "CREATE TYPE public.mood AS ENUM ('sad', 'ok', 'happy')", message: 'type "mood" already exists' }] : [],
      }));
    };
  });
}
