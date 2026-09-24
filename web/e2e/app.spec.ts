// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { expect, test } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  // Start from a clean state (default program, default layout, Spanish).
  await page.addInitScript(() => {
    try {
      if (!sessionStorage.getItem('e2e-init')) {
        localStorage.clear();
        localStorage.setItem('wmips.lang', JSON.stringify('es'));
        sessionStorage.setItem('e2e-init', '1');
      }
    } catch {
      /* ignore */
    }
  });
});

test('assemble, single-step and switch language with the mock engine', async ({ page }) => {
  await page.goto('/?mock=1');
  await expect(page.getByTestId('engine-kind')).toHaveText('mock');

  // Assemble the default program from the editor panel.
  await page.getByTestId('assemble').click();
  await expect(page.getByTestId('status-text')).toContainText('Programa ensamblado');
  await expect(page.getByTestId('asm-errors')).toContainText('Sin errores');

  // F7 x5: the pipeline and cycles panels update.
  await page.getByTestId('status-text').click();
  for (let i = 0; i < 5; i++) await page.keyboard.press('F7');
  await expect(page.getByTestId('status-cycles')).toContainText('5');
  await expect(page.getByTestId('pipe-wb')).toHaveAttribute('data-active', '1');
  await expect(page.getByTestId('pipe-if')).toHaveAttribute('data-active', '1');

  await page.getByTestId('tab-cycles').click();
  const cycles = page.getByTestId('cycles-view');
  await expect(cycles).toBeVisible();
  await expect(cycles.locator('.cy-cell').first()).toBeVisible();
  await expect(cycles).toContainText('ld r1, len(r0)');
  await expect(cycles).toContainText('WB');

  // Switch language to English.
  await page.getByTestId('lang-select').selectOption('en');
  await expect(page.getByTestId('tab-registers')).toHaveText('Registers');
  await expect(page.getByTestId('tab-cycles')).toHaveText('Cycles');
  await expect(page.getByTestId('menu-file')).toHaveText('File');

  // Run to the end (F4) and check the terminal output.
  await page.keyboard.press('F4');
  await expect(page.getByTestId('status-text')).toContainText(/HALT|finished/);
  await page.getByTestId('tab-terminal').click();
  await expect(page.getByTestId('terminal-output')).toHaveText('3');
});

test('shows assembler errors inline and in the list', async ({ page }) => {
  await page.goto('/?mock=1');
  await expect(page.getByTestId('engine-kind')).toHaveText('mock');
  const editor = page.locator('.cm-content');
  await editor.click();
  await page.keyboard.press('ControlOrMeta+End');
  await page.keyboard.type('\n        bogus r1, r2\n');
  await page.getByTestId('assemble').click();
  await expect(page.getByTestId('asm-errors')).toContainText('Instrucción inválida');
  await expect(page.locator('.cm-lint-marker-error').first()).toBeVisible();
});
