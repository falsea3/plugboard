import { connect, expect, test } from './setup';

test('edits an enum cell from its list', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await page.getByRole('grid').getByText('team').first().dblclick();
  await page.getByRole('option', { name: 'pro' }).click();
  await expect(page.getByText('1 change')).toBeVisible();
});

test('leaves an enum cell when its value is picked again', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await grid.getByText('team').first().dblclick();
  await page.getByRole('option', { name: 'team' }).click();
  await expect(grid.locator('.select-button')).toHaveCount(0);
  await expect(grid).toBeFocused();
  await expect(page.getByText('1 change')).toHaveCount(0);

  await grid.getByText('team').first().dblclick();
  await page.keyboard.press('Tab');
  await expect(grid.locator('.select-button')).toHaveCount(0);
  await expect(grid).toBeFocused();
});

test('turns rows into SQL from the context menu', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await page.getByRole('grid').getByText('bob.kim2@example.com').click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Open SQL in new query' }).hover();
  await page.getByRole('menuitem', { name: 'DELETE', exact: true }).click();
  await expect(page.locator('.cm-content')).toHaveText('DELETE FROM "public"."customers"WHERE "id" = 2;');
});

test('keeps rows under the viewport while the wheel jumps', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await expect(grid.getByText('ann.novak1@example.com')).toBeVisible();
  const misses = await grid.evaluate(async el => {
    const out: string[] = [];
    for (let i = 0; i < 30; i++) {
      const down = i < 20;
      const before = el.scrollTop;
      el.dispatchEvent(new WheelEvent('wheel', { deltaY: down ? 3000 : -2200, bubbles: true, cancelable: true }));
      const atEdge = down ? before >= el.scrollHeight - el.clientHeight - 1 : before <= 0;
      if (el.scrollTop === before && !atEdge) out.push(`jump ${i}: the grid didn't scroll`);
      const box = el.getBoundingClientRect();
      for (const y of [box.top + box.height / 2, box.bottom - 12]) {
        const hit = document.elementFromPoint(box.left + 200, y);
        const row = hit?.closest('[role="row"]');
        const number = row?.querySelector('.rn')?.textContent?.trim();
        const middle = row ? row.getBoundingClientRect().top + 13 : y;
        const want = String(Math.floor((el.scrollTop + middle - box.top - 30) / 26) + 1);
        if (number !== want) out.push(`jump ${i}: at ${Math.round(y - box.top)}px row ${number ?? 'none'}, want ${want}`);
      }
      await new Promise(r => setTimeout(r, 0));
    }
    return out;
  });
  expect(misses).toEqual([]);
});

test('loads more rows as the table scrolls, and the paging keys go where they should', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await grid.getByText('ann.novak1@example.com').click();
  await expect(page.getByTitle('How long the last fetch took')).toHaveText('· 14 ms');
  await page.keyboard.press('End');
  await expect(page.getByText('600 of ~24,813 rows loaded')).toBeVisible();
  await expect(page.getByTitle('How long the last fetch took')).toHaveText('· 10 ms');
  await page.keyboard.press('End');
  await expect(page.getByText('900 of ~24,813 rows loaded')).toBeVisible();
  await page.keyboard.press('End');
  await page.keyboard.press('End');
  await expect(page.getByText('1,200 rows', { exact: true })).toBeVisible();
  await expect(grid.locator('.rn.active')).toHaveText('1200');
  await page.keyboard.press('Home');
  await expect(grid.locator('.rn.active')).toHaveText('1');
  await page.keyboard.press('PageDown');
  await expect(grid.locator('.rn.active')).not.toHaveText('1');
  await expect(grid.getByText('ann.novak1@example.com')).toBeHidden();
  await page.keyboard.press('PageUp');
  await expect(grid.locator('.rn.active')).toHaveText('1');
});

test('picks rows with shift and ⌘ and turns them into one INSERT', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  const grid = page.getByRole('grid');
  await grid.getByText('bob.kim2@example.com').click();
  await grid.getByText('dana.ito4@example.com').click({ modifiers: ['Shift'] });
  await grid.getByText('fatima.berg6@example.com').click({ modifiers: ['ControlOrMeta'] });
  await expect(grid.locator('.rn.active')).toHaveText(['2', '3', '4', '6']);
  await grid.getByText('fatima.berg6@example.com').click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Open 4 rows as SQL' }).hover();
  await page.getByRole('menuitem', { name: 'INSERT', exact: true }).click();
  const sql = page.locator('.cm-content');
  await expect(sql).toContainText("(2, 'bob.kim2@example.com'");
  await expect(sql).toContainText("(4, 'dana.ito4@example.com'");
  await expect(sql).toContainText("(6, 'fatima.berg6@example.com'");
  await expect(sql).not.toContainText('eli.haddad5');

  await page.getByRole('tab', { name: 'customers' }).getByRole('button').first().click();
  await grid.getByText('bob.kim2@example.com').click();
  await page.keyboard.press('Shift+ArrowDown');
  await page.keyboard.press('Shift+ArrowDown');
  await expect(grid.locator('.rn.active')).toHaveText(['2', '3', '4']);
});

test('keeps a submenu that opens to the left while the mouse crosses over to it', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await expect(page.getByRole('grid').getByText('ann.novak1@example.com')).toBeVisible();
  const box = (await page.getByRole('grid').boundingBox())!;
  await page.mouse.click(box.x + box.width - 40, box.y + 60, { button: 'right' });
  const parent = page.getByRole('menuitem', { name: 'Open SQL in new query' });
  await parent.hover();
  await expect(page.locator('.menu.sub')).toHaveClass(/flip/);
  const target = (await page.getByRole('menuitem', { name: 'DELETE', exact: true }).boundingBox())!;
  await page.mouse.move(target.x + target.width / 2, target.y + target.height / 2, { steps: 12 });
  await page.mouse.down();
  await page.mouse.up();
  await expect(page.locator('.cm-content')).toContainText('DELETE FROM "public"."customers"');
});

test('reruns a filter as soon as its operator changes', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await page.getByRole('grid').getByRole('columnheader', { name: /country/ }).click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Equals…', exact: true }).click();
  await page.keyboard.type('DE');
  await page.keyboard.press('Enter');
  const filters = () => page.evaluate(() => (window as any).lastPage.filters);
  await expect.poll(filters).toEqual([{ column: 'country', op: '=', value: 'DE' }]);
  await page.getByRole('button', { name: 'Operator' }).click();
  await page.getByRole('option', { name: '≠' }).click();
  await expect.poll(filters).toEqual([{ column: 'country', op: '!=', value: 'DE' }]);
  await page.getByRole('button', { name: 'Column' }).click();
  await page.getByRole('option', { name: 'plan' }).click();
  await expect.poll(filters).toEqual([{ column: 'plan', op: '!=', value: 'DE' }]);
});

test('edits a cell from the Value panel and commits it like any edit', async ({ page }) => {
  await connect(page);
  await page.getByRole('complementary').getByText('customers', { exact: true }).click();
  await page.getByRole('button', { name: 'Value', exact: true }).click();
  const grid = page.getByRole('grid');
  await grid.getByText('ann.novak1@example.com').click();
  const box = page.getByRole('textbox', { name: 'Value of email' });
  await expect(box).toHaveValue('ann.novak1@example.com');
  await box.fill('ann@example.com');
  await box.press('ControlOrMeta+Enter');
  await expect(grid.getByText('ann@example.com', { exact: true })).toBeVisible();
  await expect(page.getByText('1 change')).toBeVisible();

  await grid.getByText('US').first().click();
  await page.getByRole('button', { name: 'Set NULL' }).click();
  await expect(page.getByRole('region', { name: 'Selected cell value' }).getByRole('textbox')).toHaveAttribute('placeholder', 'NULL');
  await page.keyboard.press('ControlOrMeta+s');
  await expect.poll(() => page.evaluate(() => (window as any).lastChanges?.changes?.length)).toBe(1);
  expect(await page.evaluate(() => (window as any).lastChanges.changes[0].values)).toEqual({ email: 'ann@example.com', country: null });
});
