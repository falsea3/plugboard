import { expect, test } from './setup';

test('imports connections from DBeaver, leaving out what is already here or unsupported', async ({ page }) => {
  await page.getByRole('button', { name: 'Import from DBeaver' }).click();
  const dialog = page.getByRole('dialog', { name: 'Import from DBeaver' });
  const list = dialog.getByRole('group', { name: 'DBeaver connections' });
  await expect(list.getByText('billing.internal:5432/main')).toBeVisible();
  await expect(list.getByText('port 8123 (HTTP) becomes 9000')).toBeVisible();
  await expect(list.getByRole('checkbox')).toHaveCount(4);
  await expect(list.getByRole('checkbox').nth(2)).not.toBeChecked();
  await expect(list.getByRole('checkbox').nth(3)).toBeDisabled();
  await expect(list.getByText('already here')).toBeVisible();
  await expect(list.getByText('no saved password')).toBeVisible();
  await dialog.getByRole('button', { name: 'Import 2' }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByText('2 connections imported from DBeaver')).toBeVisible();
  expect(await page.evaluate(() => (window as any).lastImportKeys)).toEqual({ root: '/Users/demo/Library/DBeaverData/workspace6', keys: ['General/pg-1', 'General/ch-2'] });
});
