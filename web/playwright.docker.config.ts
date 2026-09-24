import { defineConfig, devices } from '@playwright/test';

// Runs the real-engine specs against a running container: `docker compose up -d`.
export default defineConfig({
  testDir: './e2e',
  testMatch: /wasm\.spec\.ts/,
  timeout: 60_000,
  use: { baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:8080', viewport: { width: 1280, height: 800 } },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 800 } } }],
});
