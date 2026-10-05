import { connect, expect, test } from './setup';

test('marks a typo while typing SQL', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText('selec * from customers');
  await expect(page.locator('.cm-lintRange-error')).toHaveText('selec');
  await page.keyboard.press('ControlOrMeta+a');
  await expect(page.getByRole('button', { name: 'Run selection' })).toBeVisible();
});

test('keeps Run all after running the statement under the cursor', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText('select 1;\nselect 2;');
  await page.keyboard.press('ControlOrMeta+Enter');
  await expect(page.locator('.cm-ran')).toHaveText('select 2;');
  await expect(page.getByRole('button', { name: 'Run all' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Run selection' })).toHaveCount(0);
});

test('marks what the last run ran, whatever ran before', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText('select 1;\nselect 2;\nselect 3;');
  await page.getByRole('button', { name: 'Run all' }).click();
  await expect(page.locator('.cm-ran')).toHaveCount(3);
  await page.locator('.cm-content').click();
  await page.keyboard.press('ControlOrMeta+Home');
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Shift+End');
  await expect(page.locator('.cm-ran')).toHaveCount(0);
  await page.getByRole('button', { name: 'Run selection' }).click();
  await expect(page.locator('.cm-ran')).toHaveCount(1);
  await expect(page.locator('.cm-ran')).toHaveText('select 2;');
});

test('underlines what the server can\'t parse', async ({ page }) => {
  await connect(page);
  await page.getByRole('button', { name: 'New query (⌘T)' }).click();
  await page.locator('.cm-content').click();
  await page.keyboard.insertText('select 1;\nselect * fron customers');
  await expect(page.locator('.cm-lintRange-error')).toHaveText('fron');
});

test('edits a JSON cell in the JSON editor and keeps its style', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  const cell = grid.getByRole('gridcell').filter({ hasText: '"visits": 3}' }).first();
  await cell.dblclick();
  const dialog = page.getByRole('dialog', { name: 'customers.meta' });
  const editor = dialog.locator('.cm-content');
  await expect(editor).toContainText('"source": "blog",');
  await expect(dialog.getByText('Valid JSON')).toBeVisible();

  await editor.click();
  await page.keyboard.press('ControlOrMeta+a');
  await page.keyboard.insertText('{\n  "source": "blog"\n  "visits": 4\n}');
  await expect(dialog.locator('.cm-lintRange-error')).toHaveText('"');
  await expect(dialog.getByText('Line 3, column 3: Expected “,” or “}”')).toBeVisible();
  await expect(dialog.getByRole('button', { name: /Save/ })).toBeDisabled();
  await page.keyboard.press('ControlOrMeta+a');
  await page.keyboard.insertText('{"source": "blog", "visits": 4}');
  await dialog.getByRole('button', { name: /Save/ }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByText('1 change')).toBeVisible();
  await expect(grid.getByRole('gridcell').filter({ hasText: '{"source":"blog","visits":4}' })).toBeVisible();
});

test('opens a long value in its own editor and saves it to the cell', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const cell = page.getByRole('grid').getByText('bob.kim2@example.com');
  await cell.click();
  await page.keyboard.press('Shift+Enter');
  const dialog = page.getByRole('dialog', { name: 'customers.email' });
  await expect(dialog).toContainText('20 characters · 1 line');
  await dialog.locator('.cm-content').click();
  await page.keyboard.press('End');
  await page.keyboard.insertText('\nsecond line');
  await page.keyboard.press('Meta+s');
  await expect(dialog).toBeHidden();
  await expect(page.getByText('1 change')).toBeVisible();
  expect(await page.evaluate(() => (window as any).lastChanges ?? null)).toBeNull();
});
