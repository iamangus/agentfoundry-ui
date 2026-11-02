import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

const providers = [
  { id: 'p1', name: 'DeepSeek', provider_type: 'deepseek', base_url: 'https://api.deepseek.com', default_model: 'primary-model', scope: 'user' },
  { id: 'p2', name: 'Backup', provider_type: 'openai', base_url: 'https://api.openai.com/v1', default_model: 'backup-model', scope: 'user' },
]
const profile = { id: 'profile1', name: 'Reliable', provider_id: 'p1', model: 'primary-model', backup_provider_id: 'p2', backup_model: 'backup-model', scope: 'user', created_by: 'alice' }

async function fixtures(page) {
  await page.route('**/*', async route => {
    const path = new URL(route.request().url()).pathname
    let data
    if (path === '/auth/me') data = { sub: 'alice', username: 'Alice', is_admin: false }
    else if (path === '/api/v1/providers') data = providers
    else if (path === '/api/v1/inference-profiles') data = [profile]
    else if (path === '/api/v1/teams') data = { teams: ['Engineering'] }
    else if (path === '/agents/list') data = [{ agent_id: 'a1', name: 'Assistant', kind: 'agent', scope: 'user', model: 'old-model', provider_id: 'p2', system_prompt: 'Help.', tools: [] }]
    else if (path === '/tools/servers/list') data = []
    else if (path.endsWith('/models')) data = { data: [{ id: 'primary-model', name: 'Primary' }, { id: 'backup-model' }] }
    else if (path.startsWith('/api/v1/')) data = {}
    else return route.continue()
    return route.fulfill({ json: data })
  })
}

test('DeepSeek creation discovers models before saving and keeps manual entry', async ({ page }) => {
  await fixtures(page)
  let discovery
  await page.route('**/api/v1/models/discover', async route => {
    discovery = route.request().postDataJSON()
    await route.fulfill({ json: { data: [{ id: 'live-model', name: 'Live model' }] } })
  })
  await page.goto('/providers')
  await page.getByRole('button', { name: 'Add Provider' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Provider Type').selectOption('deepseek')
  await expect(dialog.getByLabel('Base URL')).toHaveValue('https://api.deepseek.com')
  await expect(dialog.getByLabel('Schema Validation')).not.toBeChecked()
  await dialog.getByLabel('Name', { exact: true }).fill('New DeepSeek')
  await dialog.getByLabel('API Key').fill('test-secret')
  await dialog.getByRole('button', { name: 'Fetch models' }).click()
  await expect(dialog.getByRole('status')).toContainText('1 models available')
  expect(discovery).toMatchObject({ provider_type: 'deepseek', base_url: 'https://api.deepseek.com', api_key: 'test-secret' })
  await expect(dialog.locator('datalist option')).toHaveAttribute('value', 'live-model')
  await dialog.getByLabel('Default Model').fill('custom-model')
  const request = page.waitForRequest(r => r.url().endsWith('/api/v1/providers') && r.method() === 'POST')
  await dialog.getByRole('button', { name: 'Add Provider', exact: true }).click()
  expect((await request).postDataJSON()).toMatchObject({ provider_type: 'deepseek', default_model: 'custom-model', schema_validation: false })
})

test('profile application changes only the draft until agent save', async ({ page }) => {
  await fixtures(page)
  const mutations = []
  page.on('request', r => { if (['POST', 'PUT', 'DELETE'].includes(r.method())) mutations.push(r.url()) })
  await page.goto('/agents')
  await page.getByTitle('Edit', { exact: true }).click()
  await page.getByLabel('Inference profile', { exact: true }).selectOption('profile1')
  await page.getByRole('button', { name: 'Apply profile', exact: true }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Apply profile', exact: true }).click()
  await expect(page.getByLabel('Primary provider', { exact: true })).toHaveValue('p1')
  await expect(page.getByLabel('Primary model', { exact: true })).toHaveValue('primary-model')
  await expect(page.getByLabel('Backup provider', { exact: false })).toHaveValue('p2')
  await expect(page.getByLabel('Backup model', { exact: true })).toHaveValue('backup-model')
  expect(mutations).toEqual([])
  const request = page.waitForRequest(r => r.method() === 'PUT' && r.url().includes('/agents/'))
  await page.getByRole('button', { name: 'Save agent' }).click()
  const payload = (await request).postDataJSON()
  expect(payload).toMatchObject({ provider_id: 'p1', model: 'primary-model', backup_provider_id: 'p2', backup_model: 'backup-model' })
  expect(payload.inference_profile_id).toBeUndefined()
})

test('profile management edits and deletes templates without updating agents', async ({ page }) => {
  await fixtures(page)
  let saved
  let deleted = false
  await page.route('**/api/v1/inference-profiles/profile1', async route => {
    if (route.request().method() === 'DELETE') { deleted = true; return route.fulfill({ status: 204 }) }
    saved = route.request().postDataJSON()
    await route.fulfill({ json: { ...profile, ...saved } })
  })
  await page.goto('/inference-profiles')
  await expect(page.getByText('Reliable', { exact: false }).first()).toBeVisible()
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Name', { exact: true }).fill('Updated')
  for (const theme of ['light', 'dark']) {
    await page.evaluate(theme => { document.documentElement.className = theme === 'dark' ? 'dark' : ''; document.documentElement.dataset.theme = theme }, theme)
    const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
    expect(results.violations.map(v => v.id)).toEqual([])
  }
  await page.setViewportSize({ width: 320, height: 800 })
  expect(await dialog.evaluate(el => el.scrollWidth <= el.clientWidth)).toBeTruthy()
  await page.screenshot({ path: '/tmp/opencode/agentfoundry-inference-profile.png', fullPage: true })
  await dialog.getByRole('button', { name: 'Save profile', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  expect(saved).toMatchObject({ name: 'Updated', provider_id: 'p1', backup_provider_id: 'p2' })
  await page.getByRole('button', { name: 'Create profile' }).click()
  await expect(page.getByRole('dialog').getByLabel('Name', { exact: true })).toHaveValue('')
  await expect(page.getByRole('dialog').getByLabel('Name', { exact: true })).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await page.getByRole('button', { name: 'Delete', exact: true }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete profile', exact: true }).click()
  await expect.poll(() => deleted).toBeTruthy()
})

test('discovery failures do not block manual model entry', async ({ page }) => {
  await fixtures(page)
  await page.route('**/api/v1/providers/*/models', route => route.fulfill({ status: 502, json: { error: 'Provider unreachable' } }))
  await page.goto('/agents')
  await page.getByTitle('Edit', { exact: true }).click()
  await expect(page.getByRole('status').filter({ hasText: 'Provider unreachable' })).toBeVisible()
  await page.getByLabel('Primary model', { exact: true }).fill('manual-model')
  await expect(page.getByLabel('Primary model', { exact: true })).toHaveValue('manual-model')
})
