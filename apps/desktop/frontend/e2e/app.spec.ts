import { connect, expect, test } from './setup';

test('says the app is up to date instead of offering the check again', async ({ page }) => {
  await page.getByRole('button', { name: 'Settings' }).click();
  await page.getByRole('dialog', { name: 'Settings' }).getByText('About', { exact: true }).click();
  await page.getByRole('button', { name: 'Check for updates' }).click();
  await expect(page.getByText('Plugboard is up to date')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Check for updates' })).toHaveCount(0);
});

test('opens a range of tables at once and counts them on the connection', async ({ page }) => {
  await connect(page);
  const sidebar = page.getByRole('complementary');
  await sidebar.getByText('customers', { exact: true }).click();
  await sidebar.getByText('orders', { exact: true }).click({ modifiers: ['Shift'] });
  await expect(sidebar.getByText('4 selected')).toBeVisible();
  await sidebar.getByRole('button', { name: 'Open 4' }).click();
  await expect(page.locator('[data-ws] .count')).toHaveText('4');
  await expect(sidebar.getByText('4 selected')).toHaveCount(0);
});

test('shows a mapped error in plain words', async ({ page }) => {
  await connect(page);
  await page.getByLabel('Schema', { exact: true }).click();
  await page.getByRole('option', { name: 'analytics' }).click();
  await page.getByRole('button', { name: 'Schema diagram' }).click();
  await expect(page.getByRole('alert').filter({ hasText: "db.internal:5432 didn't answer in time." })).toBeVisible();
});

test('says when the last run crashed and offers the log', async ({ page }) => {
  await page.addInitScript(() => ((window as any).crashed = true));
  await page.reload();
  const toast = page.getByRole('alert').filter({ hasText: 'Plugboard quit unexpectedly last time' });
  await expect(toast).toBeVisible();
  await expect(toast).toContainText('logs/crash.log');
  await toast.getByRole('button', { name: 'Show log' }).click();
  await expect(toast).toBeHidden();
});

test('keeps a dialog open when a drag inside it ends outside', async ({ page }) => {
  await page.getByRole('button', { name: /New connection/ }).first().click();
  const dialog = page.getByRole('dialog', { name: 'New connection' });
  const name = dialog.getByLabel('Name');
  await name.fill('a long connection name');
  const box = (await name.boundingBox())!;
  await page.mouse.move(box.x + box.width - 4, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(5, 300, { steps: 6 });
  await page.mouse.up();
  await expect(dialog).toBeVisible();
  await page.mouse.click(5, 300);
  await expect(dialog).toBeHidden();
});

test('flips a settings switch on every click, even while saving is slow', async ({ page }) => {
  await page.addInitScript(() => {
    const app = (window as any).go.api.App;
    const save = app.SaveSettings;
    app.SaveSettings = (s: unknown) => new Promise(resolve => setTimeout(() => resolve(save(s)), 400));
  });
  await page.reload();
  await page.getByRole('button', { name: 'Settings' }).click();
  const box = page.getByRole('checkbox', { name: 'Install updates automatically' });
  const sw = (await box.locator('..').boundingBox())!;
  const spots = [
    { x: sw.width - 3, y: sw.height / 2 },
    { x: 3, y: 3 },
    { x: sw.width / 2, y: sw.height - 2 },
  ];
  for (const [i, want] of [true, false, true].entries()) {
    await box.locator('..').click({ position: spots[i] });
    await expect(box).toBeChecked({ checked: want });
    await page.waitForTimeout(120);
  }
  await page.waitForTimeout(1500);
  await expect(box).toBeChecked();

});

test('opens What’s new from the update banner every time', async ({ page }) => {
  await page.addInitScript(() => {
    const app = (window as any).go.api.App;
    app.CheckForUpdate = () => Promise.resolve({ available: { version: '9.9.9', notes: '### Added\n\n- **Thing.** It works.', publishedAt: '', releaseUrl: 'https://example.com' } });
  });
  await page.reload();
  const banner = page.getByRole('status').filter({ hasText: 'is available' });
  await expect(banner).toBeVisible({ timeout: 8000 });
  for (let i = 0; i < 5; i++) {
    await banner.getByRole('button', { name: 'What’s new' }).click();
    await expect(page.getByRole('dialog', { name: 'What’s new in 9.9.9' })).toBeVisible();
    await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click();
    await expect(page.getByRole('dialog')).toHaveCount(0);
  }
});

test('shows the selected cell in the Value panel of a table tab only', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const panel = page.getByRole('region', { name: 'Selected cell value' });
  await expect(panel).toContainText('Select a cell to view its value');
  await page.getByRole('grid').getByText('ann.novak1@example.com').click();
  await expect(panel.locator('pre')).toHaveText('ann.novak1@example.com');
  await page.getByRole('button', { name: 'Value', exact: true }).click();
  await expect(panel).toBeHidden();
  await page.getByRole('button', { name: 'Value', exact: true }).click();
  await expect(panel.locator('pre')).toHaveText('ann.novak1@example.com');
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await expect(panel).toBeHidden();
  await expect(page.getByRole('button', { name: 'Value', exact: true })).toHaveCount(0);
});

test('reorders tabs by dragging and keeps a click a click', async ({ page }) => {
  await connect(page);
  const tabs = page.getByRole('navigation', { name: 'Open tabs' });
  for (let i = 0; i < 3; i++) await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await expect(tabs.getByRole('tab')).toHaveText(['Query 1', 'Query 2', 'Query 3']);
  const from = (await tabs.getByRole('tab', { name: 'Query 1' }).boundingBox())!;
  const to = (await tabs.getByRole('tab', { name: 'Query 3' }).boundingBox())!;
  await page.mouse.move(from.x + 30, from.y + from.height / 2);
  await page.mouse.down();
  await page.mouse.move(to.x + to.width - 10, to.y + to.height / 2, { steps: 10 });
  await page.mouse.up();
  await expect(tabs.getByRole('tab')).toHaveText(['Query 2', 'Query 3', 'Query 1']);
  await expect(tabs.getByRole('tab', { name: 'Query 3' })).toHaveAttribute('aria-selected', 'true');
  await tabs.getByRole('tab', { name: 'Query 2' }).getByRole('button').first().click();
  await expect(tabs.getByRole('tab', { name: 'Query 2' })).toHaveAttribute('aria-selected', 'true');
});
