import path from "node:path";

import { defineConfig, devices } from "@playwright/test";

/**
 * E2E tests drive the whole stack: browser → Next.js → Go API → PostgreSQL.
 *
 * Playwright starts its own API and Next.js server on ports that differ from
 * the development ones (8080 and 3000), so the tests can run while the dev
 * servers are up. Only PostgreSQL must already be running (docker compose).
 *
 * The API uses a separate database, reset before every run, and logs the
 * confirmation emails instead of sending them through Resend.
 */

const API_PORT = 8081;
const WEB_PORT = 3001;

// Reads the repository .env, like the Go API does, so E2E_DATABASE_URL or
// DATABASE_URL can be defined there. Variables already set take precedence.
// The path is relative to this file, not to the working directory, because
// editors such as the VS Code Playwright extension load the config from the
// workspace root.
try {
  process.loadEnvFile(path.join(__dirname, "../.env"));
} catch {
  // No .env file: the variables come from the environment (CI).
}

/**
 * Returns the URL of the E2E database: E2E_DATABASE_URL when set, otherwise
 * DATABASE_URL pointing to the terminuler_test database on the same server.
 * backend/cmd/e2edb refuses any database whose name does not end with _test.
 */
function e2eDatabaseURL(): string {
  if (process.env.E2E_DATABASE_URL) {
    return process.env.E2E_DATABASE_URL;
  }

  if (!process.env.DATABASE_URL) {
    throw new Error("Set E2E_DATABASE_URL or DATABASE_URL to run the E2E tests");
  }

  const url = new URL(process.env.DATABASE_URL);
  url.pathname = "/terminuler_test";

  return url.toString();
}

export default defineConfig({
  testDir: "./tests/e2e",
  forbidOnly: !!process.env.CI,
  retries: 0,
  reporter: process.env.CI ? "github" : "list",

  use: {
    baseURL: `http://localhost:${WEB_PORT}`,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],

  webServer: [
    {
      name: "API",
      // e2edb creates, migrates and empties the E2E database before the API starts.
      command: "go run ./cmd/e2edb && go run ./cmd/api",
      cwd: "../backend",
      url: `http://localhost:${API_PORT}/health`,
      env: {
        DATABASE_URL: e2eDatabaseURL(),
        PORT: String(API_PORT),
        EMAIL_PROVIDER: "log",
        // Every booking comes from the same machine; the limit has its own tests.
        APPOINTMENT_RATE_LIMIT: "1000",
        // Required at startup; the E2E tests do not call the admin endpoints.
        CLERK_SECRET_KEY: "sk_test_e2e",
        ADMIN_CLERK_USER_ID: "user_e2e",
      },
      // Never reuse a running API: it could point to another database.
      reuseExistingServer: false,
      timeout: 120_000,
      stdout: "pipe",
    },
    {
      name: "Web",
      // A production build, because a second `next dev` cannot run while the
      // dev server is up; `next build` uses its own output directory.
      command: `npm run build && npm run start -- --port ${WEB_PORT}`,
      url: `http://localhost:${WEB_PORT}`,
      // The Clerk keys come from process.env (CI) or frontend/.env.local,
      // since Playwright passes process.env to every web server.
      env: {
        API_URL: `http://localhost:${API_PORT}`,
      },
      reuseExistingServer: false,
      timeout: 240_000,
    },
  ],
});
