import { test, expect } from '@playwright/test';

test('register', async ({ page }) => {
    await page.goto('http://localhost:8080/');

    await page.getByRole('textbox', { name: 'Enter your API key' }).click();
    await page.getByRole('textbox', { name: 'Enter your API key' }).fill('my-super-secret-token-123');
    await page.getByRole('textbox', { name: 'alex@example.com' }).click();
    await page.getByRole('textbox', { name: 'alex@example.com' }).fill('test@gmail.com');
    await page.getByRole('textbox', { name: 'golang/go' }).click();
    await page.getByRole('textbox', { name: 'golang/go' }).fill('golang-migrate/migrate');
    await page.getByRole('button', { name: 'Subscribe' }).click();

    const message = page.locator('#message');
    await expect(message).toBeVisible();
    await expect(message).toHaveClass(/bg-green-50/);
    await expect(page.locator('#repository')).toBeEmpty();

    await page.getByRole('button', { name: 'My Subs' }).click();
    const subList = page.locator('#subList');
    await expect(subList).toBeVisible();
    const listItems = page.locator('#listItems');
    await expect(listItems).toContainText('golang-migrate/migrate');
});