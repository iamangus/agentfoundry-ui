import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  use: { baseURL: 'http://127.0.0.1:18081', headless: true },
  webServer: {
    command: 'npm run build && go build -o /tmp/opencode/agentfoundry-ui-test ../cmd/server && LISTEN=127.0.0.1:18081 BACKEND_URL=http://127.0.0.1:18082 /tmp/opencode/agentfoundry-ui-test',
    url: 'http://127.0.0.1:18081',
    reuseExistingServer: false,
    timeout: 120000,
  },
})
