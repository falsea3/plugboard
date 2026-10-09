import { expect, test } from './setup';

test('opens ClickHouse tables read-only and leaves changes to the SQL editor', async ({ page }) => {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Warehouse' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await expect(grid.getByText('ann.novak1@example.com')).toBeVisible();
  await expect(page.getByText('Rows can’t be edited here — change them in the SQL editor')).toBeVisible();
  await grid.getByText('ann.novak1@example.com').dblclick();
  await expect(page.locator('.cell-editor')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Add row' })).toHaveCount(0);
  await page.getByRole('tab', { name: 'Structure' }).click();
  await expect(page.getByText('Change the structure in the SQL editor')).toBeVisible();
});
