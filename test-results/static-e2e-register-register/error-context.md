# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: static\e2e\register.spec.js >> register
- Location: static\e2e\register.spec.js:3:5

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:8080/
Call log:
  - navigating to "http://localhost:8080/", waiting until "load"

```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | 
  3  | test('register', async ({ page }) => {
> 4  |     await page.goto('http://localhost:8080/');
     |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:8080/
  5  |     await page.getByRole('textbox', { name: 'Enter your API key' }).click();
  6  |     await page.getByRole('textbox', { name: 'Enter your API key' }).fill('my-super-secret-token-123');
  7  |     await page.getByRole('textbox', { name: 'alex@example.com' }).click();
  8  |     await page.getByRole('textbox', { name: 'alex@example.com' }).fill('test@gmail.com');
  9  |     await page.getByRole('textbox', { name: 'golang/go' }).click();
  10 |     await page.getByRole('textbox', { name: 'golang/go' }).fill('golang-migrate/migrate');
  11 |     await page.getByRole('button', { name: 'Subscribe' }).click();
  12 |     await page.getByRole('button', { name: 'My Subs' }).click();
  13 | });
```