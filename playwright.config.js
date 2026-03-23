const { defineConfig } = require("@playwright/test");

const livePort = process.env.PLAYWRIGHT_LIVE_PORT || "4173";
const liveBaseURL = process.env.PLAYWRIGHT_LIVE_BASE_URL || `http://127.0.0.1:${livePort}`;
const liveAuthToken = process.env.PLAYWRIGHT_E2E_AUTH_TOKEN;

if (!liveAuthToken) {
  throw new Error("PLAYWRIGHT_E2E_AUTH_TOKEN is required for live-server Playwright tests");
}

module.exports = defineConfig({
  testDir: "./tests/browser",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: "list",
  use: {
    headless: true,
  },
  webServer: {
    command: [
      "mkdir -p tmp",
      "rm -f tmp/playwright-e2e.db",
      `APP_ENV=test LISTEN_ADDR=127.0.0.1:${livePort} DB_PATH=tmp/playwright-e2e.db PLAYWRIGHT_E2E_AUTH_TOKEN=${liveAuthToken} WHISPER_BINARY_PATH= WHISPER_MODEL_PATH= go run -tags e2e ./cmd/server/main.go`,
    ].join(" && "),
    url: `${liveBaseURL}/auth`,
    timeout: 120_000,
    reuseExistingServer: false,
  },
  projects: [
    {
      name: "dom-fixtures",
      testIgnore: /.*\.live\.spec\.js$/,
    },
    {
      name: "live-server",
      testMatch: /.*\.live\.spec\.js$/,
      use: {
        baseURL: liveBaseURL,
      },
    },
  ],
});
