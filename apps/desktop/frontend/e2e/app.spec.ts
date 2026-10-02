import { expect, test, type Page } from '@playwright/test';
import { join } from 'node:path';
import { installBridge } from './bridge';

const shotsDir = process.env.RELAYDB_SCREENSHOTS ?? '';

async function shot(page: Page, name: string) {
  if (!shotsDir) return;
  await page.mouse.move(0, 0);
  await page.screenshot({ path: join(shotsDir, `${name}.png`), animations: 'disabled' });
}

test.beforeEach(async ({ page }) => {
  await installBridge(page);
  await page.goto('/');
});

test('connects, browses a table and runs a query', async ({ page }) => {
  const options = page.getByRole('listbox', { name: 'Connections' }).getByRole('option');
  await expect(options).toHaveCount(3);
  await shot(page, 'connections');

  await options.filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await expect(grid.getByText('ann.novak1@example.com')).toBeVisible();
  await expect(page.getByText(/Rows 1–300 of ~24,813/)).toBeVisible();
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

test('marks a typo while typing SQL', async ({ page }) => {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText('selec * from customers');
  await expect(page.locator('.cm-lintRange-error')).toHaveText('selec');
  await page.keyboard.press('ControlOrMeta+a');
  await expect(page.getByRole('button', { name: 'Run selection' })).toBeVisible();
});

test('edits an enum cell from its list', async ({ page }) => {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await page.getByRole('grid').getByText('team').first().dblclick();
  await page.getByRole('option', { name: 'pro' }).click();
  await expect(page.getByText('1 change')).toBeVisible();
});

test('adds and renames columns in the Structure tab', async ({ page }) => {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
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
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
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

test('keeps Run all after running the statement under the cursor', async ({ page }) => {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText('select 1;\nselect 2;');
  await page.keyboard.press('ControlOrMeta+Enter');
  await expect(page.locator('.cm-ran')).toHaveText('select 2;');
  await expect(page.getByRole('button', { name: 'Run all' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Run selection' })).toHaveCount(0);
});

test('leaves an enum cell when its value is picked again', async ({ page }) => {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await grid.getByText('team').first().dblclick();
  await page.getByRole('option', { name: 'team' }).click();
  await expect(grid.locator('.select-button')).toHaveCount(0);
  await expect(grid).toBeFocused();
  await expect(page.getByText('1 change')).toHaveCount(0);

  await grid.getByText('team').first().dblclick();
  await page.keyboard.press('Tab');
  await expect(grid.locator('.select-button')).toHaveCount(0);
  await expect(grid).toBeFocused();
});

test('marks what the last run ran, whatever ran before', async ({ page }) => {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText('select 1;\nselect 2;\nselect 3;');
  await page.getByRole('button', { name: 'Run all' }).click();
  await expect(page.locator('.cm-ran')).toHaveCount(3);
  await page.locator('.cm-content').click();
  await page.keyboard.press('ControlOrMeta+Home');
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Shift+End');
  await expect(page.locator('.cm-ran')).toHaveCount(0);
  await page.getByRole('button', { name: 'Run selection' }).click();
  await expect(page.locator('.cm-ran')).toHaveCount(1);
  await expect(page.locator('.cm-ran')).toHaveText('select 2;');
});

test('underlines what the server can\'t parse', async ({ page }) => {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Shop' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText('select 1;\nselect * fron customers');
  await expect(page.locator('.cm-lintRange-error')).toHaveText('fron');
});

test('says the app is up to date instead of offering the check again', async ({ page }) => {
  await page.getByRole('button', { name: 'Settings' }).click();
  await page.getByRole('dialog', { name: 'Settings' }).getByText('About', { exact: true }).click();
  await page.getByRole('button', { name: 'Check for updates' }).click();
  await expect(page.getByText('Relay DB is up to date')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Check for updates' })).toHaveCount(0);
});
