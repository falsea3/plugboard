import { expect, test as base, type Page } from '@playwright/test';
import { installBridge } from './bridge';

export async function connect(page: Page, id = 'shop') {
  await page.locator(`.home [data-conn="${id}"]`).click();
  await page.getByRole('button', { name: /^(Connect|Switch to)/ }).click();
}

export const test = base.extend({
  page: async ({ page }, use) => {
    await installBridge(page);
    await page.goto('/');
    await use(page);
  },
});

export { expect };
