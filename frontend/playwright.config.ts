import { defineConfig, devices } from '@playwright/test'

// Two suites: "mock" pushes recorded frames into a vite preview (no backend);
// "app" drives the real Go program started in -web mode (see e2e/app.spec.ts).
const MOCK_PORT = 4180
const STUDIO_PORT = 47701
const TEACH_PORT = 47702
const REF = process.env.AFVI_REF || '../reference'
const GO = process.env.AFVI_STUDIO_BIN || '../bin/camera-gui-web'
const TEACH = process.env.AFVI_TEACH_BIN || '../bin/teach-web'
export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  expect: { timeout: 10_000 },
  workers: 1,
  reporter: [['list']],
  outputDir: 'test-results',
  use: {
    viewport: { width: 1500, height: 900 },
    launchOptions: {
      executablePath: process.env.CHROME_PATH || '/usr/bin/google-chrome',
      args: ['--no-sandbox', '--disable-dev-shm-usage', '--font-render-hinting=none'],
    },
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chrome', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    {
      command: `npx vite build && npx vite preview --port ${MOCK_PORT} --strictPort --host 127.0.0.1`,
      url: `http://127.0.0.1:${MOCK_PORT}`,
      reuseExistingServer: true,
      timeout: 120_000,
    },
    // the real programs in -web mode (Linux, no window): they need ./reference and
    // bin/camera (build: scripts/e2e-wsl.sh builds both first)
    {
      command: `rm -rf /tmp/afvi-e2e-captures && ${GO} -ref ${REF} -web 127.0.0.1:${STUDIO_PORT} -link off -login admin:1 -model FM1/25fbo015-15`,
      url: `http://127.0.0.1:${STUDIO_PORT}`,
      reuseExistingServer: true,
      timeout: 120_000,
      env: { AFVI_CAPTURES: '/tmp/afvi-e2e-captures' },
    },
    {
      command: `rm -rf /tmp/afvi-e2e-captures-teach && ${TEACH} -ref ${REF} -web 127.0.0.1:${TEACH_PORT} -link off -model FM1/25fbo015-15`,
      url: `http://127.0.0.1:${TEACH_PORT}`,
      reuseExistingServer: true,
      timeout: 120_000,
      env: { AFVI_CAPTURES: '/tmp/afvi-e2e-captures-teach' },
    },
  ],
})
