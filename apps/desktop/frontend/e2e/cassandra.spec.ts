import { connect, expect, test } from './setup';

test('edits Cassandra rows but leaves sorting and structure to CQL', async ({ page }) => {
  await connect(page, 'ledger');
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await expect(grid.getByText('ann.novak1@example.com')).toBeVisible();
  await expect(page.locator('.ro-reason')).toHaveCount(0);

  await grid.getByRole('columnheader', { name: /email/ }).click();
  await expect(page.getByText('This database returns rows in primary key order and can’t sort a table', { exact: false })).toBeVisible();

  await page.getByRole('tab', { name: 'Structure' }).click();
  await expect(page.getByText('Change the structure in the SQL editor')).toBeVisible();
});

test('shows Cassandra roles and its settings read-only', async ({ page }) => {
  await connect(page, 'ledger');
  await page.getByRole('button', { name: 'Roles' }).click();
  await expect(page.getByRole('listbox', { name: 'Roles' }).getByRole('option')).toHaveText([/relay\s*you\s*login\s*superuser/, /readers/]);
  await expect(page.getByText('Connections')).toHaveCount(0);
  await page.getByRole('button', { name: 'Server config' }).click();
  await expect(page.locator('.config .cm-content')).toContainText('authenticator: PasswordAuthenticator');
  await page.locator('.config .cm-content').click();
  await page.keyboard.type('typed');
  await expect(page.locator('.config .cm-content')).not.toContainText('typed');
  await expect(page.getByRole('button', { name: 'Save & reload' })).toBeDisabled();
});
