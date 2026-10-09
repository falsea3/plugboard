import { expect, test } from './setup';

test('edits Cassandra rows but leaves sorting and structure to CQL', async ({ page }) => {
  await page.getByRole('listbox', { name: 'Connections' }).getByRole('option').filter({ hasText: 'Ledger' }).getByRole('button', { name: 'Connect' }).click();
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await expect(grid.getByText('ann.novak1@example.com')).toBeVisible();
  await expect(page.locator('.ro-reason')).toHaveCount(0);

  await grid.getByRole('columnheader', { name: /email/ }).click();
  await expect(page.getByText('This database returns rows in primary key order and can’t sort a table', { exact: false })).toBeVisible();

  await page.getByRole('tab', { name: 'Structure' }).click();
  await expect(page.getByText('Change the structure in the SQL editor')).toBeVisible();
});
