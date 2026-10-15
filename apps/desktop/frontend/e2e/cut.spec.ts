import { expect, test } from './setup';
import { connect } from './setup';

test('shows a cut value as cut, and loads it whole before editing', async ({ page }) => {
  await page.addInitScript(() => {
    const App = (window as any).go.api.App;
    const fetch = App.FetchTablePage;
    App.FetchTablePage = async (s: string, q: unknown) => {
      const p = await fetch(s, q);
      p.result.rows[1][2] = 'start of a very long name';
      p.result.cut = [{ row: 1, col: 2, size: 3 * 1024 * 1024 }];
      return p;
    };
  });
  await page.reload();
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  const cell = grid.getByText('start of a very long name…');
  await expect(cell).toBeVisible();
  await cell.dblclick();
  await expect(page.locator('.cell-editor, textarea.cell-input')).toHaveCount(0);

  await page.getByRole('button', { name: /^(Value|row \d+ ·)/ }).click();
  await cell.click();
  const panel = page.getByRole('region', { name: 'Selected cell value' });
  await expect(panel).toContainText('Showing the first 1 KB of 3.0 MB');
  await expect(panel.getByRole('textbox')).toHaveCount(0);
  await panel.getByRole('button', { name: 'Load whole value' }).click();
  await expect(panel.getByRole('textbox')).toHaveValue(/^whole value of name in row 2: x+$/);
  expect(await page.evaluate(() => (window as any).lastReadCell)).toMatchObject({ schema: 'public', table: 'customers', column: 'name', key: { id: 2 } });

  await cell.click({ button: 'right' });
  await page.getByRole('menuitem', { name: /Copy row as SQL|Copy as SQL/ }).first().hover();
  await page.getByRole('menuitem', { name: 'INSERT', exact: true }).click();
  await expect(page.getByText('Some of these values are too long to show whole')).toBeVisible();
});
