import { connect, expect, test } from './setup';

test('exports the structure of a database with progress', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'Dumps' }).click();
  await page.getByRole('menuitem', { name: 'Export dump…' }).click();
  const dialog = page.getByRole('dialog', { name: 'Export dump' });
  await dialog.getByRole('tab', { name: 'Structure only' }).click();
  await expect(dialog.getByText('Rows as INSERT statements')).toHaveCount(0);
  await dialog.getByText('Replace what’s there').click();
  await dialog.getByRole('button', { name: 'Export…' }).click();
  await expect(dialog.getByRole('status')).toContainText('Writing rows · public.');
  await expect(dialog.getByText('Saved 1.2 MB with 24,813 rows from 7 tables.')).toBeVisible();
  const last = await page.evaluate(() => (window as any).lastDump);
  expect(last.name).toMatch(/^shop-\d{4}-\d\d-\d\d-\d{4}\.sql$/);
  expect(last.opts).toEqual({ schemas: [], structure: true, data: false, clean: true, inserts: false, owners: false, format: 'sql' });
  await dialog.getByRole('button', { name: 'Done' }).click();
  await expect(dialog).toBeHidden();
});

test('exports a pg_restore archive', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'Dumps' }).click();
  await page.getByRole('menuitem', { name: 'Export dump…' }).click();
  const dialog = page.getByRole('dialog', { name: 'Export dump' });
  await dialog.getByRole('tab', { name: 'Custom' }).click();
  await expect(dialog.getByText('pg_restore can list it')).toBeVisible();
  await page.screenshot({ path: 'test-results/s-export-format.png' });
  await dialog.getByRole('button', { name: 'Export…' }).click();
  await expect(dialog.getByText(/^Saved/)).toBeVisible();
  const last = await page.evaluate(() => (window as any).lastDump);
  expect(last.name).toMatch(/^shop-.*\.dump$/);
  expect(last.opts.format).toBe('custom');
});

test('stops an export', async ({ page }) => {
  await connect(page);
  await page.evaluate(() => ((window as any).dumpPause = 400));
  await page.getByRole('button', { name: 'Dumps' }).click();
  await page.getByRole('menuitem', { name: 'Export dump…' }).click();
  const dialog = page.getByRole('dialog', { name: 'Export dump' });
  await dialog.getByRole('button', { name: 'Export…' }).click();
  await dialog.getByRole('button', { name: 'Stop' }).click();
  await expect(dialog.getByRole('button', { name: 'Export…' })).toBeVisible();
  await expect(page.locator('.toast')).toHaveCount(0);
});

test('imports a dump and lists what went wrong', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'Dumps' }).click();
  await page.getByRole('menuitem', { name: 'Import dump…' }).click();
  const dialog = page.getByRole('dialog', { name: 'Import dump' });
  await expect(dialog.getByRole('button', { name: 'Import', exact: true })).toBeDisabled();
  await dialog.getByRole('button', { name: 'Choose a dump file…' }).click();
  await expect(dialog.getByText('shop.sql.gz')).toBeVisible();
  await dialog.getByRole('button', { name: 'Import', exact: true }).click();
  await expect(dialog.getByText('Imported: 112 statements, 24,813 rows loaded.')).toBeVisible();
  await expect(dialog.getByText('Skipped 2 psql commands')).toBeVisible();
  expect(await page.evaluate(() => (window as any).lastImport.opts)).toEqual({ oneTransaction: true, stopOnError: true });

  await dialog.getByRole('button', { name: 'Back' }).click();
  await page.evaluate(() => ((window as any).nextDumpFile = { path: '/tmp/broken.sql', name: 'broken.sql', size: 900 }));
  await dialog.getByRole('button', { name: /shop\.sql\.gz/ }).click();
  await dialog.getByRole('button', { name: 'Import', exact: true }).click();
  await expect(dialog.getByText('Nothing was imported — an error rolled everything back.')).toBeVisible();
  await expect(dialog.getByText('type "mood" already exists')).toBeVisible();
  await page.keyboard.press('ControlOrMeta+Shift+j');
  await expect(page.getByText('Import, statement 57: type "mood" already exists')).toBeVisible();
});

test('refuses to import on a read-only session and hides dumps where the engine has none', async ({ page }) => {
  await connect(page);
  await page.getByRole('switch', { name: 'Read only' }).click();
  await expect(page.getByRole('switch', { name: 'Read only' })).toHaveAttribute('aria-checked', 'true');
  await page.getByRole('button', { name: 'Dumps' }).click();
  await page.getByRole('menuitem', { name: 'Import dump…' }).click();
  const dialog = page.getByRole('dialog', { name: 'Import dump' });
  await dialog.getByRole('button', { name: 'Choose a dump file…' }).click();
  await expect(dialog.getByText('Read-only is on for this connection.')).toBeVisible();
  await expect(dialog.getByRole('button', { name: 'Import', exact: true })).toBeDisabled();
  await page.screenshot({ path: 'test-results/dump-import.png' });
  await dialog.getByRole('button', { name: 'Cancel' }).click();

  await page.getByRole('button', { name: 'Dumps' }).click();
  await page.getByRole('menuitem', { name: 'Export dump…' }).click();
  await page.screenshot({ path: 'test-results/dump-export.png' });
  await page.keyboard.press('Escape');
  await page.getByRole('button', { name: 'All connections' }).click();
  await connect(page, 'analytics');
  await expect(page.getByRole('button', { name: 'Dumps' })).toHaveCount(0);
});
