import { connect, expect, test } from './setup';

test('opens ClickHouse tables read-only and leaves changes to the SQL editor', async ({ page }) => {
  await connect(page, 'warehouse');
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await expect(grid.getByText('ann.novak1@example.com')).toBeVisible();
  await expect(page.getByText('Rows can’t be edited here — change them in the SQL editor')).toBeVisible();
  await grid.getByText('ann.novak1@example.com').dblclick();
  await expect(page.locator('.cell-editor')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Add row' })).toHaveCount(0);
  await page.getByRole('button', { name: /^(Value|row \d+ ·)/ }).click();
  await expect(page.getByRole('region', { name: 'Selected cell value' }).locator('pre')).toHaveText('ann.novak1@example.com');
  await expect(page.getByRole('button', { name: 'Apply' })).toHaveCount(0);
  await page.getByRole('tab', { name: 'Structure' }).click();
  await expect(page.getByText('Change the structure in the SQL editor')).toBeVisible();
});

test('lists ClickHouse users and roles apart, and leaves xml users alone', async ({ page }) => {
  await connect(page, 'warehouse');
  await page.getByRole('button', { name: 'Roles' }).click();
  const list = page.getByRole('listbox', { name: 'Roles' });
  await expect(list.getByRole('option')).toHaveText([/analysts\s*role/]);
  await page.getByText('Built-in users').click();
  await list.getByRole('option', { name: /default/ }).click();
  await expect(page.getByRole('textbox', { name: 'Name' })).toBeDisabled();
  await expect(page.getByRole('button', { name: 'Drop user' })).toBeDisabled();
  await page.getByRole('button', { name: 'New user' }).first().click();
  await page.getByRole('button', { name: 'Add to a role' }).click();
  await expect(page.getByRole('option', { name: 'analysts', exact: true })).toBeVisible();
});
