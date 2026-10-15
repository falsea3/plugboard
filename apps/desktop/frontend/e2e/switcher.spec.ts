import type { Page } from '@playwright/test';
import { connect, expect, test } from './setup';

const pop = (page: Page) => page.getByRole('dialog', { name: 'Switch connection' });

async function openSwitcher(page: Page) {
  await page.getByRole('button', { name: /^Switch connection — / }).click();
  await expect(pop(page)).toBeVisible();
}

test('lists connections by group and switches from the keyboard', async ({ page }) => {
  await connect(page);
  await openSwitcher(page);
  const tree = pop(page).getByRole('tree', { name: 'Connections' });
  await expect(tree.getByRole('group')).toHaveText([/Team.*1 open.*2/, /Data.*2/, /Ungrouped.*2/]);
  await expect(tree.getByRole('treeitem', { name: /Shop/ }).first()).toContainText('● open');

  await pop(page).getByLabel('Find a connection').fill('ledg');
  await expect(tree.getByRole('group')).toHaveCount(1);
  await page.keyboard.press('Enter');
  await expect(pop(page)).toBeHidden();
  await expect(page.getByRole('button', { name: 'Switch connection — Ledger' })).toBeVisible();

  await page.keyboard.press('Control+Tab');
  await expect(page.getByRole('button', { name: 'Switch connection — Shop' })).toBeVisible();
});

test('collapses groups and drags connections and groups into a new order', async ({ page }) => {
  await connect(page);
  await openSwitcher(page);
  const tree = pop(page).getByRole('tree', { name: 'Connections' });
  const head = (name: string) => tree.locator(`[data-head="${name}"]`);
  const row = (id: string) => tree.locator(`[data-conn="${id}"]`);

  await head('Data').click();
  await expect(row('ledger')).toHaveCount(0);
  await expect(head('Data')).toHaveAttribute('aria-expanded', 'false');

  const from = (await row('music').boundingBox())!;
  const to = (await row('shop').boundingBox())!;
  await page.mouse.move(from.x + 40, from.y + from.height / 2);
  await page.mouse.down();
  await page.mouse.move(to.x + 40, to.y + 6, { steps: 8 });
  await page.mouse.up();
  await expect.poll(() => tree.locator('[data-section="Team"] [data-conn]').evaluateAll(els => els.map(e => e.getAttribute('data-conn')))).toEqual(['music', 'shop', 'analytics']);

  const g = (await head('Data').boundingBox())!;
  const t = (await head('Team').boundingBox())!;
  await page.mouse.move(g.x + 60, g.y + g.height / 2);
  await page.mouse.down();
  await page.mouse.move(t.x + 60, t.y + 4, { steps: 8 });
  await page.mouse.up();
  await expect.poll(() => tree.locator('[data-section]').evaluateAll(els => els.map(e => e.getAttribute('data-section')))).toEqual(['Data', 'Team', '']);
  await expect(head('Data')).toHaveAttribute('aria-expanded', 'false');

});

test('creates, renames and removes a group', async ({ page }) => {
  await connect(page);
  await openSwitcher(page);
  await pop(page).getByRole('button', { name: 'Group', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'New group' });
  await dialog.getByRole('textbox').fill('Team');
  await dialog.getByRole('button', { name: 'Create' }).click();
  await expect(dialog).toContainText('A group with this name exists');
  await dialog.getByRole('textbox').fill('Later');
  await dialog.getByRole('button', { name: 'Create' }).click();
  const tree = pop(page).getByRole('tree', { name: 'Connections' });
  await expect(tree.locator('[data-head="Later"]')).toBeVisible();

  await tree.locator('[data-head="Team"]').hover();
  await tree.getByRole('button', { name: 'Remove group Team' }).click();
  await expect(tree.locator('[data-section=""] [data-conn]')).toHaveCount(4);
});
