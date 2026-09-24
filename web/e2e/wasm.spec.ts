import { expect, test } from '@playwright/test';

// Real Go/WASM engine against a running container (docker compose up -d):
//   npx playwright test -c playwright.docker.config.ts

test('real engine: assemble, step, run to halt, no CSP errors', async ({ page }) => {
  const problems: string[] = [];
  page.on('console', (m) => {
    if (m.type() === 'error') problems.push(m.text());
  });
  page.on('pageerror', (e) => problems.push(e.message));
  await page.addInitScript(() => {
    try {
      localStorage.clear();
      localStorage.setItem('wmips.lang', JSON.stringify('es'));
    } catch {
      /* ignore */
    }
  });
  await page.goto('/');
  await expect(page.getByTestId('engine-kind')).toHaveText('wasm', { timeout: 20_000 });

  await page.getByTestId('assemble').click();
  await expect(page.getByTestId('asm-errors')).toContainText('Sin errores');

  await page.getByTestId('status-text').click();
  for (let i = 0; i < 5; i++) await page.keyboard.press('F7');
  await expect(page.getByTestId('status-cycles')).toContainText('5');
  await expect(page.getByTestId('pipe-wb')).toHaveAttribute('data-active', '1');

  await page.keyboard.press('F4');
  await expect(page.getByTestId('status-text')).toContainText(/HALT|detenida|finished/i, { timeout: 20_000 });
  // Regression: after a long run the Cycles grid must render the scrolled-to columns.
  const newest = await page
    .getByTestId('cycles-view')
    .evaluate((v) => Math.max(...[...v.querySelectorAll('.cy-head')].map((h) => Number(h.textContent))));
  expect(newest).toBeGreaterThan(40);
  await page.screenshot({ path: 'test-results/wasm-after-run.png', fullPage: true });
  expect(problems, problems.join('\n')).toEqual([]);
});
