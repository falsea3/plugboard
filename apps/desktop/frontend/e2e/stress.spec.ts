import { expect, test, type Page } from '@playwright/test';
import { installBridge } from './bridge';
import { connect } from './setup';

test.skip(!process.env.PLUGBOARD_STRESS, 'set PLUGBOARD_STRESS=1 to run the stress walkthrough');
test.setTimeout(90_000);

async function start(page: Page, overrides: string) {
  await installBridge(page);
  await page.addInitScript(overrides);
  await page.goto('/');
}

async function timed(label: string, fn: () => Promise<unknown>) {
  const t = Date.now();
  await fn();
  console.log(`${label.padEnd(44)} ${String(Date.now() - t).padStart(7)} ms`);
}

const busy = (page: Page) =>
  page.evaluate(
    () =>
      new Promise<number>(resolve => {
        const t = performance.now();
        setTimeout(() => resolve(performance.now() - t), 0);
      }),
  );

test('10 000 tables in the sidebar', async ({ page }) => {
  await start(page, `
    const App = window.go.api.App;
    App.ListTables = () => Promise.resolve(Array.from({ length: 10000 }, (_, i) => ({ schema: 'public', name: 't' + i, kind: 'table' })));
  `);
  await timed('connect + render 10k tables', async () => {
    await connect(page);
    await expect(page.getByText('9,500 more — show them, or filter')).toBeVisible({ timeout: 60_000 });
  });
  const filter = page.getByPlaceholder('Filter tables');
  await timed('type a 4-letter filter', async () => {
    await filter.click();
    await page.keyboard.type('t123', { delay: 0 });
    await expect(page.getByText('t1234', { exact: true })).toBeVisible();
  });
  console.log('DOM nodes', await page.evaluate(() => document.querySelectorAll('*').length));
});

test('2000 query tabs restored', async ({ page }) => {
  await start(page, `
    const App = window.go.api.App;
    App.QueryTabs = () => Promise.resolve(Array.from({ length: ${Number(process.env.STRESS_TABS ?? 2000)} }, (_, i) => ({ title: 'Query ' + i, sql: 'select ' + i + ' from orders where id > 10;\\n'.repeat(20), script: '', saved: '', active: i === 0 })));
  `);
  await timed('connect + restore the query tabs', async () => {
    await connect(page);
    await expect(page.getByRole('tab', { name: 'Query ' + (Number(process.env.STRESS_TABS ?? 2000) - 1) })).toBeAttached({ timeout: 60_000 });
  });
  await timed('switch to the next tab (⌘⇧])', async () => {
    await page.keyboard.press('ControlOrMeta+Shift+BracketRight');
    await expect(page.locator('.cm-content').filter({ hasText: 'select 1 from' })).toBeVisible();
  });
  await timed('type in that tab', async () => {
    await page.locator('.cm-content').filter({ hasText: 'select 1 from' }).click();
    await page.keyboard.type('-- typed');
    await expect(page.locator('.cm-content').filter({ hasText: '-- typed' })).toBeVisible();
  });
  console.log('DOM nodes', await page.evaluate(() => document.querySelectorAll('*').length), 'busy', await busy(page));
});

test('a table with 1500 columns', async ({ page }) => {
  await start(page, `
    const App = window.go.api.App;
    const cols = Array.from({ length: 1500 }, (_, i) => ({ name: i ? 'c' + i : 'id', type: i ? 'text' : 'bigint', nullable: true, default: null, primaryKey: i === 0, enum: null, kind: i ? 'text' : 'number' }));
    App.DescribeTable = () => Promise.resolve(cols);
    App.FetchTablePage = () => Promise.resolve({
      result: { statement: '', columns: cols.map(c => ({ name: c.name, type: c.type, kind: c.kind })), rows: Array.from({ length: 300 }, (_, r) => cols.map((c, i) => i ? 'v' + i : r + 1)), rowsAffected: 0, hasRows: true, truncated: false, pageable: false, hasMore: false, offset: 0, durationMs: 5 },
      hasMore: false, defaultOrder: ['id'], hasPrev: false, keyset: true, offset: -1,
    });
  `);
  await connect(page);
  await timed('open the wide table', async () => {
    await page.getByRole('complementary').getByText('customers', { exact: true }).click();
    await expect(page.getByRole('grid').getByText('v3', { exact: true }).first()).toBeVisible({ timeout: 60_000 });
  });
  await timed('scroll far right', async () => {
    await page.getByRole('grid').hover();
    for (let i = 0; i < 60; i++) await page.mouse.wheel(4000, 0);
    await expect(page.getByRole('grid').getByText('v1499', { exact: true }).first()).toBeVisible();
  });
  console.log('DOM nodes', await page.evaluate(() => document.querySelectorAll('*').length));
});

test('a page of 300 rows of big JSON, cut by the backend', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await start(page, `
    const App = window.go.api.App;
    const big = JSON.stringify(Array.from({ length: 30000 }, (_, i) => ({ i, hash: 'abcdef0123456789abcdef0123456789' }))).slice(0, 65536);
    const cols = [{ name: 'id', type: 'int', kind: 'number' }, { name: 'doc', type: 'jsonb', kind: 'json' }];
    App.DescribeTable = () => Promise.resolve(cols.map((c, i) => ({ ...c, nullable: true, default: null, primaryKey: i === 0, enum: null })));
    App.FetchTablePage = () => Promise.resolve({
      result: { statement: '', columns: cols, rows: Array.from({ length: 300 }, (_, r) => [r + 1, big]), rowsAffected: 0, hasRows: true, truncated: false, pageable: false, hasMore: false, offset: 0, durationMs: 5, cut: Array.from({ length: 300 }, (_, row) => ({ row, col: 1, size: 0 })) },
      hasMore: false, defaultOrder: ['id'], hasPrev: false, keyset: true, offset: -1,
    });
  `);
  await connect(page);
  await timed('open the table (300 cells of 64 KB)', async () => {
    await page.getByRole('complementary').getByText('customers', { exact: true }).click();
    await expect(page.getByRole('grid').getByRole('gridcell').filter({ hasText: /^\[\{"i":0/ }).first()).toBeVisible({ timeout: 60_000 });
  });
  await timed('select a big cell', async () => {
    await page.getByRole('grid').getByRole('row').nth(2).getByRole('gridcell').nth(1).click();
  });
});

test('a diagram of 10 000 tables in one chain', async ({ page }) => {
  await start(page, `
    const App = window.go.api.App;
    const col = (name, pk) => ({ name, type: 'int', nullable: !pk, default: null, primaryKey: pk, enum: null, kind: 'number' });
    App.Diagram = (_, schema) => Promise.resolve({
      schema,
      tables: Array.from({ length: 10000 }, (_, i) => ({ name: 't' + i, kind: 'table', columns: [col('id', true), col('parent', false), col('name', false)] })),
      relations: Array.from({ length: 9999 }, (_, i) => ({ name: 'fk' + i, table: 't' + (i + 1), columns: ['parent'], refSchema: schema, refTable: 't' + i, refColumns: ['id'], onDelete: 'NO ACTION', onUpdate: 'NO ACTION' })),
    });
  `);
  await connect(page);
  await timed('open the diagram', async () => {
    await page.getByRole('button', { name: 'Schema diagram' }).click();
    await expect(page.locator('.card[data-table="t0"]')).toBeAttached({ timeout: 60_000 });
  });
  await timed('zoom in once', async () => {
    await page.getByRole('button', { name: 'Zoom in' }).click();
  });
  console.log('DOM nodes', await page.evaluate(() => document.querySelectorAll('*').length), 'busy', await busy(page));
});
