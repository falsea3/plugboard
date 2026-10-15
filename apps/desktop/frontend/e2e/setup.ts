import { expect, test as base, type Page } from '@playwright/test';
import { join } from 'node:path';
import { installBridge } from './bridge';

const shotsDir = process.env.PLUGBOARD_SCREENSHOTS ?? '';

export async function shot(page: Page, name: string) {
  if (!shotsDir) return;
  await page.mouse.move(0, 0);
  await page.screenshot({ path: join(shotsDir, `${name}.png`), animations: 'disabled' });
}

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
