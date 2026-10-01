import { defineConfig, devices } from '@playwright/test';

// The walkthrough in e2e/ runs the UI against a stand-in backend. With
// RELAYDB_SCREENSHOTS set to a folder it also saves the screenshots used in
// the README (make screenshots).
const screenshots = Boolean(process.env.RELAYDB_SCREENSHOTS);

export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: 'npm run dev -- --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
  },
  projects: [
    {
      // The app renders in WKWebView, so WebKit is the browser that matches it.
      name: 'webkit',
      use: { ...devices['Desktop Safari'], viewport: { width: 1280, height: 800 }, deviceScaleFactor: screenshots ? 2 : 1 },
    },
  ],
});
