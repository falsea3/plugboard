import { expect, test, type Page } from '@playwright/test';
import { join } from 'node:path';
import { installBridge } from './bridge';

const shotsDir = process.env.RELAYDB_SCREENSHOTS ?? '';

async function shot(page: Page, name: string) {
  if (!shotsDir) return;
  await page.mouse.move(0, 0); // no hover highlights in the picture
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
