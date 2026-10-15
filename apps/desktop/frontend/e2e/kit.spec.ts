import { expect, test } from '@playwright/test';

test('shows every building block in both themes', async ({ page }) => {
  await page.goto('/#kit');
  await expect(page.getByRole('heading', { name: 'Plugboard UI kit' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save' })).toBeVisible();
  await page.getByRole('tab', { name: 'Structure' }).first().click();
  await expect(page.getByRole('tab', { name: 'Structure' }).first()).toHaveAttribute('aria-selected', 'true');
  await page.getByRole('tab', { name: 'Light' }).click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
});
