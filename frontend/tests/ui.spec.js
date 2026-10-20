import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

const agents = [{ agent_id: 'a1', name: 'Research assistant', kind: 'agent', scope: 'user', description: 'Find answers and summarize results.', model: 'primary-model', provider_id: 'p1', backup_provider_id: 'p2', backup_model: 'backup-model', system_prompt: 'Be helpful.', tools: [] }]
const providers = [{ id: 'p1', name: 'Primary cloud', provider_type: 'openai', default_model: 'primary-model', base_url: 'https://primary.example/v1', scope: 'user', schema_validation: true }, { id: 'p2', name: 'Backup cloud', provider_type: 'custom', default_model: 'backup-model', base_url: 'https://backup.example/v1', scope: 'user' }]

async function fixtures(page) {
  // Audit settled theme states, not a transient frame halfway through a
  // background-color hover transition. Also exercises reduced-motion support.
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.route('**/*', async route => {
    const url = new URL(route.request().url()), path = url.pathname
    if (path.startsWith('/assets/') || !['/agents/list', '/tools/servers/list', '/auth/me', '/api/keys'].includes(path) && !path.startsWith('/api/v1/') && !path.startsWith('/agents/Research')) return route.continue()
    let data = []
    if (path === '/auth/me') data = { username: 'Alex', is_admin: true }
    else if (path === '/agents/list') data = agents
    else if (path === '/api/v1/providers') data = providers
    else if (path === '/api/v1/teams') data = { teams: ['Engineering'] }
    else if (path === '/tools/servers/list') data = [{ id: 's1', name: 'Knowledge', url: 'https://tools.example/mcp', connected: true, scope: 'user', tools: [{ name: 'search', description: 'Search the knowledge base', scope: 'user' }] }]
    else if (path === '/api/keys') data = [{ id: 'k1', name: 'Development', key_prefix: 'af_dev', created_at: '2026-10-20T12:00:00Z' }]
    else if (path === '/api/v1/chat/sessions') data = [{ id: 'session1', agent_id: 'a1', agent_name: 'Research assistant', messages: [] }]
    else if (path === '/api/v1/chat/sessions/session1') data = { id: 'session1', messages: [{ role: 'user', content: 'Summarize the findings.' }, { role: 'assistant', content: 'Here is a concise summary.\n\n- Reliable inference\n- Accessible interfaces\n\n```go\nfmt.Println("Hello")\n```' }] }
    else if (path === '/api/v1/executions') data = [{ workflow_id: 'wf1', agent_name: 'Research assistant', status: 'Completed', start_time: '2026-10-20T12:00:00Z', close_time: '2026-10-20T12:00:10Z' }]
    else if (path.endsWith('/trace')) data = { events: [{ kind: 'model', model: 'backup-model', provider_id: 'p2', backup: true, attempts: 1, failover_reason: 'Primary provider failed after 2 attempts', input: {}, output: {} }] }
    else if (path === '/api/v1/executions/wf1') data = { workflow_id: 'wf1', status: 'Completed', agent_name: 'Research assistant', history: [], spans: [], start_time: '2026-10-20T12:00:00Z' }
    else if (path.endsWith('/versions')) data = { versions: [] }
    return route.fulfill({ json: data })
  })
}

for (const path of ['/chat', '/agents', '/providers', '/tools', '/api-keys', '/executions']) {
  test(`${path}: both themes, accessible structure, narrow layouts`, async ({ page }) => {
    await fixtures(page)
    await page.goto(path)
    await expect(page.locator('main')).toBeVisible()
    for (const theme of ['light', 'dark']) {
      await page.evaluate(theme => { document.documentElement.className = theme === 'dark' ? 'dark' : ''; document.documentElement.dataset.theme = theme }, theme)
      const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
      expect(results.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([])
      await page.screenshot({ path: `/tmp/opencode/agentfoundry-${path.slice(1)}-${theme}.png`, fullPage: true })
    }
    for (const width of [280, 320, 414, 1280]) {
      await page.setViewportSize({ width, height: 900 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy()
      if (width === 414) await page.screenshot({ path: `/tmp/opencode/agentfoundry-${path.slice(1)}-mobile.png`, fullPage: true })
    }
  })
}

test('agent editor: backup values, validation, save payload, unsaved guard', async ({ page }) => {
  await fixtures(page)
  await page.goto('/agents')
  await page.getByTitle('Edit', { exact: true }).click()
  await expect(page.getByLabel('Backup provider', { exact: false })).toHaveValue('p2')
  await page.getByLabel('Backup model', { exact: true }).fill('replacement')
  await page.locator('.form-body').evaluate(el => el.scrollTop = 0)
  await page.screenshot({ path: '/tmp/opencode/agentfoundry-editor.png', fullPage: true })
  await page.getByRole('button', { name: 'Cancel', exact: true }).click()
  await expect(page.getByRole('alertdialog')).toBeVisible()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Cancel', exact: true }).click()
  await expect(page.getByLabel('Backup model', { exact: true })).toHaveValue('replacement')
  await page.getByLabel('Name', { exact: true }).fill('')
  await page.getByRole('button', { name: 'Save agent' }).click()
  await expect(page.getByRole('alert')).toContainText('required')
  await page.getByLabel('Name', { exact: true }).fill('Research assistant')
  const request = page.waitForRequest(r => r.method() === 'PUT' && r.url().includes('/agents/'))
  await page.getByRole('button', { name: 'Save agent' }).click()
  expect((await request).postDataJSON()).toMatchObject({ backup_provider_id: 'p2', backup_model: 'replacement' })
})

test('search, dialogs, focus restoration, execution failover trace', async ({ page }) => {
  await fixtures(page)
  await page.goto('/agents')
  await page.getByRole('searchbox', { name: 'Search agents' }).fill('missing')
  await expect(page.getByText('No matching agents')).toBeVisible()
  await page.getByRole('searchbox', { name: 'Search agents' }).fill('')
  const trigger = page.getByTitle('Delete', { exact: true })
  await trigger.click()
  await expect(page.getByRole('alertdialog')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(trigger).toBeFocused()
  await page.getByRole('link', { name: 'Executions', exact: true }).click()
  await page.getByText('Research assistant', { exact: true }).click()
  await page.getByText('backup-model', { exact: true }).click()
  await expect(page.getByText('Primary provider failed after 2 attempts', { exact: true })).toBeVisible()
})

test('mobile navigation and provider dialog keyboard operation', async ({ page }) => {
  await fixtures(page)
  await page.setViewportSize({ width: 320, height: 800 })
  await page.goto('/providers')
  await page.getByRole('button', { name: 'Toggle navigation' }).click()
  await expect(page.getByRole('link', { name: 'API keys', exact: true })).toBeVisible()
  await page.getByRole('link', { name: 'Providers', exact: true }).click()
  await page.getByRole('button', { name: 'Add Provider' }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
  expect(results.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([])
  await page.keyboard.press('Escape')
  await expect(page.getByRole('button', { name: 'Add Provider' })).toBeFocused()
})

test('API errors are actionable, not empty success states', async ({ page }) => {
  await fixtures(page)
  await page.route('**/agents/list', route => route.fulfill({ status: 503, json: { error: 'Temporarily unavailable' } }))
  await page.goto('/agents')
  await expect(page.getByRole('alert')).toContainText('Temporarily unavailable')
  await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()
  await expect(page.getByText('Your first agent starts here')).toHaveCount(0)
})

test('embedded server serves deep links and strict API 404s', async ({ request }) => {
  const route = await request.get('/executions')
  expect(route.status()).toBe(200)
  expect(route.headers()['content-type']).toContain('text/html')
  const unknown = await request.get('/api/not-a-route')
  expect(unknown.status()).toBe(404)
  expect(unknown.headers()['content-type']).not.toContain('text/html')
})

test('expanded agent editor is accessible and usable on narrow screens', async ({ page }) => {
  await fixtures(page)
  await page.goto('/agents')
  await page.getByTitle('Edit', { exact: true }).click()
  await page.locator('.editor-section').evaluateAll(elements => elements.forEach(el => { if (el.tagName === 'DETAILS') el.open = true }))
  for (const theme of ['light', 'dark']) {
    await page.evaluate(theme => { document.documentElement.className = theme === 'dark' ? 'dark' : ''; document.documentElement.dataset.theme = theme }, theme)
    const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
    expect(results.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([])
  }
  await page.setViewportSize({ width: 320, height: 900 })
  await page.locator('.form-body').evaluate(el => el.scrollTop = 0)
  await page.screenshot({ path: '/tmp/opencode/agentfoundry-editor-mobile.png', fullPage: true })
  expect(await page.locator('.form-page').evaluate(el => el.scrollWidth <= el.clientWidth)).toBeTruthy()
})

test('chat preserves successful responses and discards failed partial streams', async ({ page }) => {
  await fixtures(page)
  await page.route('**/api/v1/agents/a1/run', route => route.fulfill({ json: { run_id: 'r1' } }))
  await page.route('**/runs/r1/events', route => route.fulfill({ contentType: 'text/event-stream', body: [
    'event: response_boundary\ndata: \n\n',
    'event: response_start\ndata: \n\n',
    'event: token\ndata: Completed tool explanation\n\n',
    'event: response_boundary\ndata: \n\n',
    'event: response_start\ndata: \n\n',
    'event: token\ndata: Failed partial response\n\n',
    'event: response_retry\ndata: Retrying\n\n',
    'event: response_start\ndata: \n\n',
    'event: token\ndata: Backup answer\n\n',
    'event: done\ndata: Backup answer\n\n',
  ].join('') }))
  await page.goto('/chat?session=session1')
  await expect(page.getByText('Reliable inference')).toBeVisible()
  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
  expect(results.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([])
  await page.screenshot({ path: '/tmp/opencode/agentfoundry-chat-conversation.png', fullPage: true })
  await page.getByRole('textbox', { name: 'Message agent' }).fill('Continue')
  await page.getByRole('button', { name: 'Send message' }).click()
  await expect(page.getByText('Backup answer', { exact: true })).toBeVisible()
  await expect(page.getByText('Completed tool explanation', { exact: true })).toBeVisible()
  await expect(page.getByText('Failed partial response', { exact: true })).toHaveCount(0)
})

for (const [path, button, name] of [['/tools', 'Register Server', 'MCP server'], ['/api-keys', 'Create Key', 'Create API key']]) {
  test(`${path}: dialog naming, contrast, and focus`, async ({ page }) => {
    await fixtures(page)
    await page.goto(path)
    await page.getByRole('button', { name: button, exact: true }).click()
    await expect(page.getByRole('dialog', { name })).toBeVisible()
    for (const theme of ['light', 'dark']) {
      await page.evaluate(theme => { document.documentElement.className = theme === 'dark' ? 'dark' : ''; document.documentElement.dataset.theme = theme }, theme)
      const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
      expect(results.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([])
    }
    await page.keyboard.press('Escape')
    await expect(page.getByRole('button', { name: button, exact: true })).toBeFocused()
  })
}
