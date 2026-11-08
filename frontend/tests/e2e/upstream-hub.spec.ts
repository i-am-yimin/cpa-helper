import { expect, test, type Page } from '@playwright/test'

// All accounts and traffic in these captures are synthetic test fixtures.
async function fixture(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('cpa-helper-language', 'zh')
    localStorage.setItem('cpa-helper-theme', 'light')
  })
  const now = new Date().toISOString()
  const reset = new Date(Date.now() + 3 * 3600000).toISOString()
  const usage = { records: 18, total_tokens: 152000, estimated_cost_usd: 0.42, unpriced_records: 0, reset_at: reset, stale: false }
  const accounts = [
    { name: 'codex-team.json', email: 'codex-team@example.test', provider: 'codex', account_type: 'plus', priority: 4, disabled: false, primary_used_percent: 24, secondary_used_percent: 50, primary_window_usage: usage },
    { name: 'antigravity-team.json', email: 'antigravity-team@example.test', provider: 'antigravity', priority: 0, disabled: false },
  ]
  let upstreamItems = [{ name: 'Team gateway', 'base-url': 'https://gateway.example.test/v1', 'api-key': 'test-only-key', priority: 8, disabled: false, custom_setting: 'preserved', _cpa_v8: { group: 0, key: 0 } }]
  const groups = [
    { id: 'gemini', label: 'Gemini Models', priority: 5, effective_priority: 5, degraded: false, model_patterns: ['gemini-*'], buckets: [{ id: 'g5h', window: '5h', remaining_percent: 82, reset_at: reset, usage }] },
    { id: 'claude-gpt', label: 'Claude and GPT models', priority: 8, effective_priority: -1, degraded: true, model_patterns: ['claude-*', 'gpt-*'], buckets: [{ id: 'c5h', window: '5h', remaining_percent: 0, reset_at: reset, usage }] },
  ]
  let savedGroup: Record<string, unknown> | undefined
  let savedUpstream: Record<string, unknown> | undefined
  let upstreamReads = 0
  await page.route('**/api/**', async route => {
    const path = new URL(route.request().url()).pathname
    if (!path.startsWith('/api/')) return route.continue()
    let data: unknown = {}
    if (path === '/api/auth/me') data = { id: 1, username: 'admin', nickname: 'Admin', email: '', avatar: '', is_admin: true, must_change_password: false, created_at: now, cas_bound: false, can_change_password: true, can_view_account_status: true, can_view_usage_history: true }
    else if (path === '/api/upstreams/group-scheduler') {
      if (route.request().method() === 'PUT') {
        savedGroup = route.request().postDataJSON()
        const group = groups.find(group => group.id === savedGroup?.group_id)!
        group.priority = Number(savedGroup?.priority)
        group.model_patterns = savedGroup?.model_patterns as string[]
        data = { groups, checked_at: now, stale: false }
      } else data = { active: true, last_pick_at: now }
    } else if (path === '/api/upstreams') { upstreamReads++; data = { sections: { 'openai-compatibility': upstreamItems }, api_version: 8, revision: '"revision-1"' } }
    else if (path === '/api/upstreams/openai-compatibility') {
      savedUpstream = route.request().postDataJSON()
      upstreamItems = savedUpstream?.items as typeof upstreamItems
    } else if (path === '/api/codex-keeper/accounts') data = { items: accounts, priority_rules: [{ account_type: 'plus', priority: 4 }], degraded: false }
    else if (path.endsWith('/antigravity-team.json/quota')) data = { groups, checked_at: now, attempted_at: now, stale: false }
    else if (path === '/api/codex-keeper/status') data = { logs: [], stats: {}, running_modes: [], running: false }
    await route.fulfill({ json: data })
  })
  return { groups, getSavedGroup: () => savedGroup, getSavedUpstream: () => savedUpstream, getUpstreamReads: () => upstreamReads }
}

test('unified upstreams edit API priorities with revision and independent Antigravity group rules', async ({ page }) => {
  const state = await fixture(page)
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.goto('/admin/upstreams')
  await expect(page.getByRole('heading', { name: '上游与凭证', exact: true })).toBeVisible()
  const allRows = page.locator('[data-upstream-row]')
  await expect(allRows).toHaveCount(3)
  await expect(allRows.first()).toContainText('Team gateway')
  const api = page.locator('[data-upstream-row="openai-compatibility:0"]')
  await api.getByRole('spinbutton').fill('21')
  await api.getByRole('button', { name: '保存', exact: true }).click()
  await expect.poll(() => state.getSavedUpstream()).toMatchObject({ api_version: 8, revision: '"revision-1"', items: [{ priority: 21, custom_setting: 'preserved', _cpa_v8: { group: 0, key: 0 } }] })
  const ag = page.locator('[data-upstream-row="oauth:antigravity-team.json"]')
  await ag.getByRole('button', { name: '设置分组优先级' }).click()
  await expect(page.getByText('当前生效').last()).toContainText('-1')
  await page.getByRole('spinbutton', { name: 'Claude and GPT models 分组优先级', exact: true }).fill('12')
  await page.locator('.group-policy').filter({ hasText: 'Claude and GPT models' }).getByRole('button', { name: '保存分组' }).click()
  await expect.poll(() => state.getSavedGroup()).toMatchObject({ group_id: 'claude-gpt', priority: 12, model_patterns: ['claude-*', 'gpt-*'] })
  expect(state.groups[0]!.priority).toBe(5)
  await expect(page.locator('[data-antigravity-quota]')).toContainText('152K')
  await page.getByRole('button', { name: '查看 antigravity-team@example.test 详情' }).click()
  await page.screenshot({ path: '../.impeccable/review/upstream-hub-desktop.png', fullPage: true })
  await ag.getByRole('button', { name: '设置分组优先级' }).click()
  await page.locator('.group-rules-header').scrollIntoViewIfNeeded()
  await page.screenshot({ path: '../.impeccable/review/upstream-hub-groups.png', fullPage: true })
  await page.locator('[data-antigravity-quota]').scrollIntoViewIfNeeded()
  await page.screenshot({ path: '../.impeccable/review/upstream-hub-window-usage.png', fullPage: true })
  expect(errors).toEqual([])
})

test('native API toggles preserve exclusions and show distinct credential identities', async ({ page }) => {
  await fixture(page)
  let items: Array<Record<string, unknown>> = [
    { 'base-url': 'https://same.example.test/v1', 'auth-index': 'key-a', 'api-key': 'synthetic-secret-aaaa', 'excluded-models': ['*', 'gpt-blocked'], priority: 4 },
    { 'base-url': 'https://same.example.test/v1', 'auth-index': 'key-b', 'api-key': 'synthetic-secret-bbbb', 'excluded-models': ['gpt-other'], priority: 4 },
    { 'base-url': 'https://same.example.test/v1', 'auth-index': 'key-c', 'api-key': 'synthetic-secret-cccc', 'excluded-models': ['*'], priority: 4 },
  ]
  let saved: Array<Record<string, unknown>> | undefined
  await page.route('**/api/upstreams', route => route.fulfill({ json: { sections: { 'codex-api-key': items }, api_version: 8, revision: '"native-revision"' } }))
  await page.route('**/api/upstreams/codex-api-key', async route => { saved = route.request().postDataJSON().items; items = saved!; await route.fulfill({ json: {} }) })
  await page.goto('/admin/upstreams')
  const first = page.locator('[data-upstream-row="codex-api-key:0"]')
  const second = page.locator('[data-upstream-row="codex-api-key:1"]')
  await expect(first).toContainText('#key-a')
  await expect(second).toContainText('#key-b')
  await expect(page.locator('.hub-list')).not.toContainText('synthetic-secret')
  await expect(first.getByRole('switch')).not.toBeChecked()
  await expect(second.getByRole('switch')).toBeChecked()
  await first.getByRole('switch').click()
  await expect.poll(() => saved?.[0]?.['excluded-models']).toEqual(['gpt-blocked'])
  expect(saved?.[1]?.['excluded-models']).toEqual(['gpt-other'])
  await expect(first.getByRole('switch')).toBeChecked()
  await first.getByRole('switch').click()
  await expect.poll(() => saved?.[0]?.['excluded-models']).toEqual(['*', 'gpt-blocked'])
  const third = page.locator('[data-upstream-row="codex-api-key:2"]')
  await expect(third.getByRole('switch')).not.toBeChecked()
  await third.getByRole('switch').click()
  await expect.poll(() => saved?.[2]?.['excluded-models']).toBeUndefined()
  await expect(third.getByRole('switch')).toBeChecked()
})

test('an inline draft blocks reloading and switching editors until saved or discarded', async ({ page }) => {
  await page.clock.install()
  const state = await fixture(page)
  await page.goto('/admin/upstreams')
  const api = page.locator('[data-upstream-row="openai-compatibility:0"]')
  await api.getByRole('spinbutton').fill('21')
  await expect(page.getByRole('button', { name: '刷新总览', exact: true })).toBeDisabled()
  await expect(page.locator('[data-upstream-row="oauth:codex-team.json"]').getByRole('spinbutton')).toBeDisabled()
  const reads = state.getUpstreamReads()
  await page.clock.fastForward(61_000)
  expect(state.getUpstreamReads()).toBe(reads)
  await page.getByRole('tab', { name: 'API 接入配置', exact: true }).click()
  await expect(page.getByRole('tab', { name: '全部上游', exact: true })).toHaveAttribute('data-state', 'active')
  await page.getByRole('button', { name: '放弃更改', exact: true }).click()
  await expect.poll(() => state.getUpstreamReads()).toBeGreaterThan(reads)
  await expect(api.getByRole('spinbutton')).toHaveValue('8')
})

test('unified upstreams fit mobile and dark mode, retain filtering and old route', async ({ page }) => {
  await fixture(page)
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/admin/upstreams')
  await expect(page.locator('[data-upstream-row]')).toHaveCount(3)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await expect(page.locator('.hub-table tbody')).toHaveCSS('display', 'block')
  expect(await page.locator('.content-scroll').evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true)
  await page.screenshot({ path: '../.impeccable/review/upstream-hub-mobile.png', fullPage: true })
  await page.setViewportSize({ width: 768, height: 1000 })
  expect(await page.locator('.content-scroll').evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true)
  await page.screenshot({ path: '../.impeccable/review/upstream-hub-tablet.png', fullPage: true })
  await page.getByRole('textbox', { name: '搜索上游' }).fill('antigravity')
  await expect(page.locator('[data-upstream-row]')).toHaveCount(1)
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.evaluate(() => document.documentElement.classList.add('dark'))
  await page.evaluate(async () => {
    await new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
    await Promise.all(document.getAnimations().filter(animation => Number.isFinite(animation.effect?.getComputedTiming().iterations ?? 0)).map(animation => animation.finished.catch(() => {})))
  })
  await page.screenshot({ path: '../.impeccable/review/upstream-hub-dark.png', fullPage: true })
  await page.goto('/admin/credential-mgmt')
  await expect(page).toHaveURL(/\/admin\/upstreams\?tab=credentials$/)
  await expect(page.getByRole('button', { name: 'OAuth 登录', exact: true })).toBeVisible()
})
