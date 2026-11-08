import { expect, test } from '@playwright/test'

test('Antigravity shows grouped quota, zero balance, resets and refresh errors alongside Codex', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('cpa-helper-language', 'zh'))
  const now = new Date().toISOString()
  const emptyAccount = {
    name: 'ag.json', email: 'ag@example.test', provider: 'antigravity',
    account_type: 'oauth', disabled: false, unavailable: false, runtime_only: false,
    status: 'active', priority: 0, primary_used_percent: null, secondary_used_percent: null,
    primary_reset_at: null, secondary_reset_at: null, quota: null, model_quotas: null,
    primary_window_usage: null, secondary_window_usage: null,
  }
  const groups = [
    { label: 'Gemini Models', buckets: [
      { id: 'gemini-5h', label: 'Five Hour Limit', window: '5h', remaining_percent: 100, reset_at: '2027-01-08T17:02:08Z' },
      { id: 'gemini-weekly', label: 'Weekly Limit', window: 'weekly', remaining_percent: 0, reset_at: '2027-01-15T12:02:08Z' },
    ] },
    { label: 'Claude and GPT models', buckets: [
      { id: 'claude-5h', label: 'Five Hour Limit', window: '5h', remaining_percent: 45.7, reset_at: null },
    ] },
  ]
  let forcedRefreshes = 0
  let codexRefreshNames: string[] = []
  await page.route('**/api/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (!path.startsWith('/api/')) return route.continue()
    let data: unknown = {}
    if (path === '/api/auth/me') {
      data = { id: 1, username: 'admin', nickname: 'Admin', email: '', avatar: '', created_at: now,
        is_admin: true, cas_bound: true, can_change_password: false, must_change_password: false,
        can_view_account_status: true, can_view_usage_history: true }
    } else if (path === '/api/codex-keeper/accounts') {
      data = { items: [emptyAccount, { ...emptyAccount, name: 'codex.json', email: 'codex@example.test', provider: 'codex',
        account_type: 'plus', primary_used_percent: 25, secondary_used_percent: 50,
        primary_window_seconds: 18000, secondary_window_seconds: 604800 }], priority_rules: [], degraded: false }
    } else if (path === '/api/codex-keeper/status') {
      data = { running: false, daemon_running: false, running_modes: [], logs: [], stats: {} }
    } else if (path === '/api/codex-keeper/accounts/refresh') {
      codexRefreshNames = route.request().postDataJSON().auth_names
      data = { status: 'started' }
    } else if (path.endsWith('/ag.json/quota')) {
      const force = route.request().method() === 'POST'
      if (force) forcedRefreshes++
      data = { groups, checked_at: now, attempted_at: now, stale: force, error: force ? 'Antigravity 额度查询失败（HTTP 429）' : null }
    }
    await route.fulfill({ json: data })
  })
  await page.goto('/admin/credential-mgmt')
  const panel = page.locator('[data-antigravity-quota]')
  await expect(panel.getByText('Gemini Models', { exact: true })).toBeVisible()
  await expect(panel.getByRole('progressbar', { name: 'Gemini Models · 每周' })).toHaveAttribute('aria-valuenow', '0')
  await expect(panel.getByText('45.7%', { exact: true })).toBeVisible()
  await expect(panel.getByText(/2027-01-09 01:02:08/)).toBeVisible()
  await expect(panel.getByText(/上游未提供/)).toBeVisible()
  const codexRow = page.getByRole('row').filter({ hasText: 'codex@example.test' })
  await expect(codexRow.getByText('75%', { exact: true })).toBeVisible()
  await panel.getByRole('button', { name: '刷新 Antigravity 额度' }).click()
  await expect(panel.getByRole('status')).toHaveText('Antigravity 额度查询失败（HTTP 429）')
  await expect(panel.getByText('上次查询结果 · 已过期')).toBeVisible()
  await expect(panel.getByText('100%', { exact: true })).toBeVisible()
  expect(forcedRefreshes).toBe(1)
  await page.getByRole('checkbox', { name: '选择 ag@example.test', exact: true }).check()
  await page.getByRole('checkbox', { name: '选择 codex@example.test', exact: true }).check()
  await page.getByRole('button', { name: '刷新', exact: true }).click()
  await expect.poll(() => codexRefreshNames).toEqual(['codex.json'])
  expect(forcedRefreshes).toBe(2)
  await page.screenshot({ path: 'test-results/antigravity-quota.png', fullPage: true })
})
