import { connect, expect, shot, test } from './setup';

test('edits the server config, shows the server’s problems and saves', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'Server config' }).click();
  const editor = page.locator('.config .cm-content');
  await expect(editor).toContainText('max_connections = 100');
  await expect(page.getByText('via SQL')).toBeVisible();
  await editor.click();
  await page.keyboard.press('ControlOrMeta+End');
  await page.keyboard.type('work_mem = ');
  await expect(page.getByRole('tab', { name: 'Server config' }).locator('.dirty-dot')).toBeVisible();
  await page.keyboard.press('ControlOrMeta+s');
  const problems = page.getByRole('alert').filter({ hasText: 'Saved, but not reloaded' });
  await expect(problems).toContainText('line 5syntax error in file');
  await shot(page, 'server-config');
  await page.keyboard.type('4MB');
  await page.getByRole('button', { name: 'Save & reload' }).click();
  await expect(page.getByText('Saved and reloaded')).toBeVisible();
  await expect(problems).toHaveCount(0);
  await expect(page.getByRole('tab', { name: 'Server config' }).locator('.dirty-dot')).toHaveCount(0);
  expect(await page.evaluate(() => (window as any).lastConfig.text)).toContain('\nwork_mem = 4MB');
});

test('opens another file by path and asks before dropping edits', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'Server config' }).click();
  await page.locator('.config .cm-content').click();
  await page.keyboard.type('# mine\n');
  await page.getByRole('button', { name: 'Configuration file' }).click();
  await page.getByRole('option', { name: /pg_hba\.conf/ }).click();
  await expect(page.getByRole('dialog', { name: 'Discard your changes?' })).toBeVisible();
  await page.getByRole('button', { name: 'Discard' }).click();
  await expect(page.locator('.config .cm-content')).toContainText('scram-sha-256');
  await page.getByRole('button', { name: 'Configuration file' }).click();
  await page.getByRole('option', { name: 'Other file…' }).click();
  await page.getByRole('textbox', { name: 'File path' }).fill('/etc/missing.conf');
  await page.getByRole('button', { name: 'Open', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('could not open file "/etc/missing.conf"');
});

test('creates a role through the SQL the backend writes', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'Roles' }).click();
  const list = page.getByRole('listbox', { name: 'Roles' });
  await expect(list.getByRole('option')).toHaveText([/app\s*login/, /readers/, /relay\s*you\s*login\s*superuser/]);
  await page.getByText('Built-in roles').click();
  await expect(list.getByRole('option', { name: /pg_monitor/ })).toBeVisible();
  await page.getByRole('button', { name: 'New role' }).last().click();
  await expect(page.getByRole('textbox', { name: 'Name' })).toBeFocused();
  await page.keyboard.type('reporting');
  await page.getByLabel('Password').fill('s3cret');
  await page.getByText('Create databases').click();
  await page.getByRole('button', { name: 'Add to a role' }).click();
  await page.getByRole('option', { name: 'readers', exact: true }).last().click();
  await shot(page, 'roles');
  await page.getByRole('button', { name: 'Create role' }).click();
  await expect(page.getByRole('dialog', { name: 'Role changes' })).toContainText(`CREATE ROLE "reporting" WITH LOGIN PASSWORD '********'`);
  await page.getByRole('button', { name: 'Commit' }).click();
  await expect(list.getByRole('option', { name: /reporting/ })).toHaveAttribute('aria-selected', 'true');
  expect(await page.evaluate(() => (window as any).lastRole)).toMatchObject({ from: '', to: { name: 'reporting', password: 's3cret', flags: { createDB: true, login: true }, memberOf: ['readers'] } });
  await expect(page.getByRole('button', { name: 'Save' })).toBeDisabled();
});

test('manages MySQL users with a host, and opens their grants as SQL', async ({ page }) => {
  await connect(page, 'analytics');
  await page.getByRole('button', { name: 'Roles' }).click();
  const list = page.getByRole('listbox', { name: 'Roles' });
  await expect(list.getByRole('option')).toHaveText([/reporting@%/, /root@localhost\s*you\s*all privileges/]);
  await list.getByRole('option', { name: /reporting/ }).click();
  await expect(page.getByRole('heading', { name: 'reporting@%' })).toBeVisible();
  await expect(page.getByRole('textbox', { name: 'Host' })).toHaveValue('%');
  await expect(page.getByText('Valid until')).toHaveCount(0);
  await expect(page.getByText('GRANT SELECT ON `events`.*')).toBeVisible();
  await page.getByRole('button', { name: 'Open as SQL' }).click();
  await expect(page.locator('.cm-content')).toContainText('GRANT SELECT ON `events`.* TO `reporting`@`%`;');
});
