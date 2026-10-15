import type { Page } from '@playwright/test';
import { join } from 'node:path';
import { connect, expect, test } from './setup';

const dir = process.env.PLUGBOARD_SCREENSHOTS ?? '';

test.skip(!dir, 'set PLUGBOARD_SCREENSHOTS to retake the README pictures');
test.use({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 2 });

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(screen, 'width', { get: () => 3000 });
    Object.defineProperty(screen, 'height', { get: () => 2000 });
  });
  await page.reload();
});

const LIGHTS = `body::after { content: ''; position: fixed; z-index: 9999; left: 20px; top: 20px; width: 12px; height: 12px;
  border-radius: 50%; background: #ff5f57; box-shadow: 20px 0 0 #febc2e, 40px 0 0 #28c840; pointer-events: none; }`;

async function snap(page: Page, name: string, themes: ('dark' | 'light')[] = ['dark']) {
  await page.addStyleTag({ content: LIGHTS });
  await page.mouse.move(720, 890);
  for (const theme of themes) {
    await page.evaluate(t => (document.documentElement.dataset.theme = t), theme);
    await page.waitForTimeout(250);
    await page.screenshot({ path: join(dir, theme === 'dark' ? `${name}.png` : `${name}-light.png`), animations: 'disabled' });
  }
}

const side = (page: Page) => page.getByRole('complementary').first();

async function openTable(page: Page, name = 'customers') {
  await connect(page);
  await side(page).getByText(name, { exact: true }).click();
  await expect(page.getByRole('grid').getByText('ann.novak1@example.com')).toBeVisible();
}

test('table', async ({ page }) => {
  await openTable(page);
  const grid = page.getByRole('grid');
  await grid.getByText('Fatima Berg').first().dblclick();
  await page.keyboard.press('ControlOrMeta+a');
  await page.keyboard.type('Fatima Berg-Lund');
  await page.keyboard.press('Enter');
  await grid.getByText('dana.ito4@example.com').click();
  await page.getByRole('button', { name: /^(Value|row \d+ ·)/ }).click();
  await snap(page, 'table', ['dark', 'light']);
});

test('home', async ({ page }) => {
  await openTable(page);
  await page.keyboard.press('ControlOrMeta+Shift+h');
  await page.locator('.home [data-conn="shop"]').click();
  await expect(page.getByText('2.4 GB')).toBeVisible();
  await snap(page, 'home', ['dark', 'light']);
});

test('switcher', async ({ page }) => {
  await openTable(page);
  await page.getByRole('button', { name: /^Switch connection — / }).click();
  await page.locator('[data-conn="ledger"]').hover();
  await snap(page, 'switcher');
});

test('query', async ({ page }) => {
  await openTable(page);
  await page.getByRole('button', { name: 'New query', exact: true }).click();
  await page.locator('.cm-content').last().click();
  await page.keyboard.insertText(
    '-- revenue by country, active customers only\nSELECT c.country, count(*) AS customers, sum(o.total) AS revenue\nFROM customers c\nJOIN orders o ON o.customer_id = c.id\nWHERE c.is_active\nGROUP BY c.country\nORDER BY revenue DESC;\n\nSELECT * FROM refunds WHERE amount > 100;',
  );
  await page.keyboard.press('ControlOrMeta+Home');
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('End');
  await page.keyboard.press('ControlOrMeta+Enter');
  await expect(page.getByRole('grid').getByText('1284390.50')).toBeVisible();
  await snap(page, 'query', ['dark', 'light']);
});

test('structure', async ({ page }) => {
  await openTable(page);
  await page.getByRole('tab', { name: 'Structure' }).click();
  await page.getByRole('grid').getByText('character(2)').dblclick();
  await page.keyboard.press('ControlOrMeta+a');
  await page.keyboard.type('varchar(3)');
  await page.keyboard.press('Enter');
  await snap(page, 'structure');
});

test('filters', async ({ page }) => {
  await openTable(page);
  await page.getByRole('button', { name: /^Filter/ }).click();
  const bar = page.locator('.filters');
  await bar.getByRole('button', { name: 'Column' }).click();
  await page.getByRole('option', { name: 'country' }).click();
  await bar.locator('input.value').fill('DE');
  await bar.getByRole('button', { name: /Condition/ }).click();
  await bar.getByRole('button', { name: 'Column' }).last().click();
  await page.getByRole('option', { name: 'plan' }).click();
  await bar.locator('input.value').last().fill('pro');
  await snap(page, 'filters');
});

test('json', async ({ page }) => {
  await openTable(page);
  await page.getByRole('grid').getByText('{"source": "ads", "visits": 0}').click();
  await page.keyboard.press('Shift+Enter');
  await page.getByRole('button', { name: 'Format' }).click();
  await snap(page, 'json');
});

test('diagram', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'Schema diagram' }).click();
  await expect(page.locator('.card[data-table="orders"]')).toBeVisible();
  await page.locator('.card[data-table="orders"]').click();
  await snap(page, 'diagram', ['dark', 'light']);
});

test('connection form', async ({ page }) => {
  await page.locator('.home [data-conn="shop"]').click();
  await page.getByRole('button', { name: 'More' }).click();
  await page.getByRole('menuitem', { name: 'Edit…' }).click();
  await page.getByRole('button', { name: 'Test' }).click();
  await expect(page.getByText(/Connected via SSH/)).toBeVisible();
  await snap(page, 'connection-form');
});

test('redis', async ({ page }) => {
  await connect(page, 'cache');
  const tree = page.getByRole('tree', { name: 'Keys' });
  for (const n of ['shop', 'customer', '1']) await tree.getByRole('treeitem').filter({ has: page.locator('.label', { hasText: new RegExp(`^${n}$`) }) }).click();
  await expect(page.getByRole('tab', { name: 'shop:customer:1' })).toBeVisible();
  await snap(page, 'redis');
  await page.getByRole('button', { name: 'New console' }).click();
  await page.locator('.cm-content').last().click();
  await page.keyboard.type('HGETALL shop:customer:1');
  await page.keyboard.press('ControlOrMeta+Enter');
  await page.waitForTimeout(300);
  await snap(page, 'redis-console');
});

test('roles and config', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'Roles' }).click();
  await page.getByRole('listbox', { name: 'Roles' }).getByRole('option').first().click();
  await snap(page, 'roles');
  await page.getByRole('button', { name: 'Server config' }).click();
  await expect(page.locator('.config .cm-content')).toContainText('max_connections');
  await snap(page, 'server-config');
});

test('agents', async ({ page }) => {
  await page.getByRole('button', { name: 'Settings' }).click();
  await page.getByRole('dialog', { name: 'Settings' }).getByText('AI agents', { exact: true }).click();
  await snap(page, 'mcp');
});
