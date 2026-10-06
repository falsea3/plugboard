import { connect, expect, test } from './setup';

type Page = import('@playwright/test').Page;
const tabs = (page: Page) => page.getByRole('navigation', { name: 'Open tabs' });
const script = (page: Page, name: string) => page.getByRole('complementary').getByRole('button', { name, exact: true });

test('asks before closing an edited query and saves it as a script', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.type('select 42');
  await expect(tabs(page).getByTitle('Unsaved changes')).toBeVisible();

  await page.keyboard.press('ControlOrMeta+w');
  const ask = page.getByRole('dialog', { name: 'Save changes to “Query 1”?' });
  await ask.getByRole('button', { name: 'Cancel' }).click();
  await expect(ask).toBeHidden();
  await expect(page.locator('.cm-content')).toHaveText('select 42');

  await page.keyboard.press('ControlOrMeta+w');
  await ask.getByRole('button', { name: 'Save as…' }).click();
  const naming = page.getByRole('dialog', { name: 'Save as script' });
  await expect(naming.getByLabel('Script name')).toHaveValue('Query 1');
  await naming.getByLabel('Script name').fill('monthly revenue');
  await expect(naming.getByRole('alert')).toHaveText('A script with this name already exists.');
  await naming.getByLabel('Script name').fill('answer');
  await naming.getByRole('button', { name: 'Save and close' }).click();

  await expect(tabs(page).getByRole('tab')).toHaveCount(0);
  await script(page, 'answer').click();
  await expect(tabs(page).getByRole('tab', { name: 'answer' })).toBeVisible();
  await expect(page.locator('.cm-content')).toHaveText('select 42');
});

test('closes unchanged queries at once and drops unsaved text on request', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.keyboard.press('ControlOrMeta+w');
  await expect(tabs(page).getByRole('tab')).toHaveCount(0);

  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.type('select 1');
  await tabs(page).getByRole('button', { name: 'Close tab' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Don’t save' }).click();
  await expect(tabs(page).getByRole('tab')).toHaveCount(0);
  await expect(script(page, 'Query 1')).toHaveCount(0);
});

test('saves an open script with ⌘S and keeps query tabs across restarts', async ({ page }) => {
  await connect(page);
  await script(page, 'monthly revenue').click();
  await page.locator('.cm-content').click();
  await page.keyboard.press('ControlOrMeta+End');
  await page.keyboard.type(' -- by month');
  await expect(tabs(page).getByTitle('Unsaved changes')).toBeVisible();
  await page.keyboard.press('ControlOrMeta+s');
  await expect(tabs(page).getByTitle('Unsaved changes')).toHaveCount(0);

  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').last().click();
  await page.keyboard.type('select now()');
  await expect(tabs(page).getByTitle('Unsaved changes')).toBeVisible();

  await page.reload();
  await connect(page);
  await expect(tabs(page).getByRole('tab')).toHaveText(['monthly revenue', 'Query 1']);
  await expect(page.locator('.pane:not(.hidden) .cm-content')).toHaveText('select now()');
  await expect(tabs(page).getByTitle('Unsaved changes')).toHaveCount(1);
  await tabs(page).getByRole('tab', { name: 'monthly revenue' }).getByRole('button').first().click();
  await expect(page.locator('.pane:not(.hidden) .cm-content')).toContainText('GROUP BY 1; -- by month');
});

test('renames and deletes scripts from the sidebar', async ({ page }) => {
  await connect(page);
  await script(page, 'monthly revenue').click();
  await script(page, 'monthly revenue').click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Rename…' }).click();
  const naming = page.getByRole('dialog', { name: 'Rename script' });
  await naming.getByLabel('Script name').fill('revenue');
  await naming.getByRole('button', { name: 'Rename' }).click();
  await expect(tabs(page).getByRole('tab', { name: 'revenue' })).toBeVisible();

  await script(page, 'revenue').click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Delete…' }).click();
  await page.getByRole('dialog', { name: 'Delete “revenue”?' }).getByRole('button', { name: 'Delete' }).click();
  await expect(script(page, 'revenue')).toHaveCount(0);
  await expect(tabs(page).getByTitle('Unsaved changes')).toBeVisible();
});
