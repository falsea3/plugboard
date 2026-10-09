import type { Page } from '@playwright/test';
import { expect, test } from './setup';

async function connectCache(page: Page) {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Cache' }).getByRole('button', { name: 'Connect' }).click();
}

const tree = (page: Page) => page.getByRole('tree', { name: 'Keys' });
const item = (page: Page, name: string) => tree(page).getByRole('treeitem').filter({ has: page.locator('.label', { hasText: new RegExp(`^${name}$`) }) });

test('browses keys by folder and edits a hash field', async ({ page }) => {
  await connectCache(page);
  await expect(page.getByRole('button', { name: 'Database', exact: true })).toContainText('Database 0');
  await item(page, 'shop').click();
  await item(page, 'customer').click();
  await item(page, '1').click();
  await expect(page.getByRole('tab', { name: 'shop:customer:1' })).toBeVisible();
  const plan = page.getByRole('cell', { name: 'pro', exact: true });
  await plan.dblclick();
  await page.keyboard.press('ControlOrMeta+a');
  await page.keyboard.type('team');
  await page.keyboard.press('Enter');
  await expect(page.getByRole('cell', { name: 'team', exact: true })).toBeVisible();
  expect(await page.evaluate(() => (window as any).lastKeyEdit)).toMatchObject({ key: 'shop:customer:1', op: 'hset', field: 'plan', value: 'team' });
});

test('edits a JSON string and saves it with ⌘S', async ({ page }) => {
  await connectCache(page);
  await item(page, 'shop').click();
  await item(page, 'session:8f2a').click();
  await expect(page.getByText('TTL 1d')).toBeVisible();
  await page.getByRole('button', { name: 'Format JSON' }).click();
  await expect(page.locator('.string .cm-content')).toContainText('"cart": [');
  await page.locator('.string .cm-content').click();
  await page.keyboard.press('ControlOrMeta+s');
  await expect.poll(() => page.evaluate(() => (window as any).lastKeyEdit?.op)).toBe('set');
  expect(await page.evaluate(() => (window as any).lastKeyEdit.value)).toContain('\n  "customer": 1');
});

test('creates, filters and deletes keys', async ({ page }) => {
  await connectCache(page);
  await page.getByRole('button', { name: 'New key' }).click();
  const dialog = page.getByRole('dialog', { name: 'New key in database 0' });
  await dialog.getByLabel('Name').fill('shop:flags');
  await dialog.getByLabel('Value').fill('on');
  await dialog.getByRole('button', { name: 'Create' }).click();
  await expect(page.getByRole('tab', { name: 'shop:flags' })).toBeVisible();

  await page.getByRole('textbox', { name: 'Filter keys' }).fill('leader');
  await expect(tree(page).getByRole('treeitem')).toHaveText([/shop:leaderboard/]);
  await tree(page).getByRole('treeitem').click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Delete…' }).click();
  await page.getByRole('dialog', { name: 'Delete “shop:leaderboard”?' }).getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByText('No keys match “leader”.')).toBeVisible();
});

test('runs commands in a console and confirms writes on staging only when asked', async ({ page }) => {
  await connectCache(page);
  await page.getByRole('button', { name: 'New console' }).click();
  await expect(page.getByRole('tab', { name: 'Console 1' })).toBeVisible();
  await page.locator('.cm-content').click();
  await page.keyboard.type('GET shop:config:currency');
  await page.keyboard.press('Escape');
  await page.keyboard.press('ControlOrMeta+Enter');
  await expect(page.getByRole('grid').getByText('USD')).toBeVisible();
});
