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
  await page.getByRole('button', { name: 'Database', exact: true }).click();
  await expect(page.getByRole('option')).toHaveText([/Database 0\s*6 keys/, /Database 9\s*3 keys/, /Database 1\s*empty/, ...Array(13).fill(/empty/)]);
  await page.keyboard.press('Escape');
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

test('adds a Redis user with ACL rules and changes a live setting', async ({ page }) => {
  await connectCache(page);
  await page.getByRole('button', { name: 'Roles' }).click();
  await expect(page.getByRole('listbox', { name: 'Roles' }).getByRole('option')).toHaveText([/default\s*you/]);
  await expect(page.getByText('Member of')).toHaveCount(0);
  await page.getByRole('button', { name: 'New user' }).last().click();
  await page.keyboard.type('app');
  await page.getByLabel('Password', { exact: true }).fill('pw');
  await page.getByLabel('Rules').fill('~cache:* +@read');
  await page.getByRole('button', { name: 'Create user' }).click();
  await expect(page.getByRole('dialog')).toContainText('ACL SETUSER app on resetpass >******** ~cache:* +@read');
  await page.getByRole('button', { name: 'Commit' }).click();
  expect(await page.evaluate(() => (window as any).lastRole.to)).toMatchObject({ name: 'app', password: 'pw', rules: '~cache:* +@read' });

  await page.getByRole('button', { name: 'Server config' }).click();
  await expect(page.getByText('live', { exact: true })).toBeVisible();
  const editor = page.locator('.config .cm-content');
  await expect(editor).toContainText('maxmemory-policy noeviction');
  await editor.click();
  await page.keyboard.press('ControlOrMeta+End');
  await page.keyboard.type('no-such-setting 1');
  await page.keyboard.press('ControlOrMeta+s');
  await expect(page.getByRole('alert').filter({ hasText: 'The server refused a change' })).toContainText('line 4ERR Unknown option');
  await page.keyboard.press('Backspace');
  for (let i = 0; i < 16; i++) await page.keyboard.press('Backspace');
  await page.keyboard.press('ControlOrMeta+s');
  await expect(page.getByText('Applied, but not written to a config file')).toBeVisible();
});
