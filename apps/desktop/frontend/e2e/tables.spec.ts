import { connect, expect, shot, test } from './setup';

test('connects, browses a table and runs a query', async ({ page }) => {
  const options = page.getByRole('listbox', { name: 'Connections' }).getByRole('option');
  await expect(options).toHaveCount(3);
  await shot(page, 'connections');

  await options.filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await expect(grid.getByText('ann.novak1@example.com')).toBeVisible();
  await expect(page.getByText('300 of ~24,813 rows loaded')).toBeVisible();
  await grid.getByText('dana.ito4@example.com').click();
  await shot(page, 'table');

  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText(
    'SELECT c.country, count(*) AS customers, sum(o.total) AS revenue\nFROM customers c\nJOIN orders o ON o.customer_id = c.id\nWHERE c.is_active\nGROUP BY c.country\nORDER BY revenue DESC;',
  );
  await page.getByRole('button', { name: 'Run all' }).click();
  await expect(page.getByRole('grid').getByText('1284390.50')).toBeVisible();
  await shot(page, 'query');
});

test('asks before writing to a Production connection', async ({ page }) => {
  await page
    .getByRole('listbox', { name: 'Connections' })
    .getByRole('option')
    .filter({ hasText: 'Analytics' })
    .getByRole('button', { name: 'Connect' })
    .click();
  await expect(page.getByRole('switch', { name: /Read-only/ })).toHaveAttribute('aria-checked', 'true');
  await page.getByRole('switch', { name: /Read-only/ }).click();
  await expect(page.getByRole('dialog', { name: 'Allow writes on Production?' })).toBeVisible();
  await page.getByRole('button', { name: 'Keep read-only' }).click();
  await expect(page.getByRole('switch', { name: /Read-only/ })).toHaveAttribute('aria-checked', 'true');
});

test('adds and renames columns in the Structure tab', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await page.getByRole('tab', { name: 'Structure' }).click();
  await page.getByRole('button', { name: 'Column' }).click();
  await page.keyboard.type('nickname');
  await page.keyboard.press('Tab');
  await page.keyboard.press('Enter');
  await page.keyboard.type('varchar(40)');
  await page.keyboard.press('Enter');
  await expect(page.getByText('1 change')).toBeVisible();
  await page.getByRole('button', { name: 'Preview SQL' }).click();
  await expect(page.getByText('ADD COLUMN "nickname" varchar(40)', { exact: false })).toBeVisible();
  await page.getByRole('button', { name: 'Commit', exact: true }).last().click();
  await expect(page.getByText('Saved 1 change to the structure of customers.')).toBeVisible();
});

test('won’t change the structure under uncommitted row edits', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await page.getByRole('grid').getByText('team').first().dblclick();
  await page.getByRole('option', { name: 'pro' }).click();
  await page.getByRole('tab', { name: 'Structure' }).click();
  await page.getByRole('button', { name: 'Column' }).click();
  await page.keyboard.type('nickname');
  await page.keyboard.press('Tab');
  await page.keyboard.press('Enter');
  await page.keyboard.type('varchar(40)');
  await page.keyboard.press('Enter');
  await page.getByRole('button', { name: /^Commit/ }).click();
  await expect(page.getByText('Commit or discard the 1 row change in Data first', { exact: false })).toBeVisible();
});

test('lists the indexes of a table under its columns', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await page.getByRole('tab', { name: 'Structure' }).click();
  const indexes = page.getByRole('region', { name: 'Indexes' });
  await expect(indexes.getByText('3 indexes')).toBeVisible();
  await expect(indexes.getByRole('row').filter({ hasText: 'customers_pkey' })).toContainText('Primary key');
  await expect(indexes.getByRole('row').filter({ hasText: 'customers_email_key' })).toContainText('Unique');
  await expect(indexes.getByRole('row').filter({ hasText: 'customers_active_country' })).toContainText('country, plan');
  await expect(indexes.getByRole('row').filter({ hasText: 'customers_active_country' })).toContainText('is_active');
});

test('draws the schema and explains a relation when it is clicked', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'Schema diagram' }).click();
  const canvas = page.locator('.canvas');
  await expect(canvas.locator('[data-table="orders"]')).toBeVisible();
  await expect(page.getByText('8 tables · 6 relations')).toBeVisible();
  await shot(page, 'diagram');

  await page.locator('[data-relation="orders.orders_customer_id_fkey"]').dispatchEvent('click');
  await expect(page.locator('.detail code')).toHaveText('orders.customer_id → customers.id');
  await expect(page.locator('.detail')).toContainText('on delete restrict');
  await expect(canvas.locator('[data-table="products"]')).toHaveClass(/dim/);
  await expect(canvas.locator('[data-table="customers"] .row.marked')).toHaveText(/id/);

  await canvas.locator('[data-table="customers"] .head').dblclick();
  await expect(page.getByRole('grid').getByText('ann.novak1@example.com')).toBeVisible();
});

test('follows a foreign key from a cell to the row it points at', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  const cell = grid.locator('.td.link').first();
  await expect(cell).toHaveText('US');
  await cell.hover();
  await cell.getByRole('button', { name: 'Go to the referenced row' }).click();
  await expect(page.getByRole('tab', { name: 'countries' })).toHaveAttribute('aria-selected', 'true');
  expect(await page.evaluate(() => (window as any).lastPage.filters)).toEqual([{ column: 'code', op: '=', value: 'US' }]);

  await page.getByRole('tab', { name: 'customers' }).getByRole('button').first().click();
  await grid.locator('.td.link').nth(1).click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Go to countries' }).click();
  await expect(page.getByRole('tab', { name: 'countries' })).toHaveAttribute('aria-selected', 'true');
  await expect.poll(() => page.evaluate(() => (window as any).lastPage.filters)).toEqual([{ column: 'code', op: '=', value: 'DE' }]);
});

test('shows that a table and a query are still loading', async ({ page }) => {
  await page.addInitScript(() => {
    const app = (window as any).go.api.App;
    for (const name of ['FetchTablePage', 'RunQuery']) {
      const real = app[name];
      app[name] = (...args: unknown[]) => new Promise(resolve => setTimeout(() => resolve(real(...args)), 1200));
    }
  });
  await page.reload();
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await expect(page.getByText('Loading rows…')).toBeVisible();
  await expect(page.getByRole('progressbar', { name: 'Loading customers' })).toBeAttached();
  await expect(page.getByRole('grid').getByText('ann.novak1@example.com')).toBeVisible();
  await expect(page.getByRole('progressbar')).toHaveCount(0);

  await page.getByRole('grid').hover();
  await page.mouse.wheel(0, 8000);
  await expect(page.getByText('Loading more…')).toBeVisible();
  await expect(page.getByRole('progressbar', { name: 'Loading customers' })).toBeAttached();
  await expect(page.getByText('600 of')).toBeVisible();
  await expect(page.getByRole('progressbar')).toHaveCount(0);

  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText('select 1');
  await page.getByRole('button', { name: 'Run', exact: true }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Running…' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Stop' })).toBeVisible();
  await expect(page.getByRole('grid').getByText('1284390.50')).toBeVisible();
});
