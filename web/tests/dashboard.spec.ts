import { expect, test } from '@playwright/test'

const user = { id: 7, email: 'tester@example.com', created_at: '2026-09-21T10:00:00Z' }
const session = { user, token: 'fly_session_test', expires_at: '2099-09-21T10:00:00Z' }
const project = { id: 12, owner_id: 7, name: 'Checkout API', created_at: '2026-09-21T10:00:00Z' }
const issue = {
  id: 42, project_id: 12, fingerprint: 'abc', exception_type: 'TypeError',
  message: 'Cannot read properties of undefined', stacktrace: 'cart.ts:19',
  status: 'open', event_count: 4, first_seen: '2026-09-20T10:00:00Z',
  last_seen: '2026-09-21T10:00:00Z', created_at: '2026-09-20T10:00:00Z',
}
const secondIssue = {
  ...issue,
  id: 43,
  fingerprint: 'def',
  exception_type: 'RangeError',
  message: 'Maximum call stack size exceeded',
}

test('demo mode is labeled local-only and remains usable on narrow reduced-motion screens', async ({ page }, testInfo) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.goto('/')
  await page.getByRole('button', { name: 'Explore the interactive demo' }).click()

  await expect(page.getByText('Demo · local data')).toBeVisible()
  await expect(page.getByRole('heading', { name: 'DatabaseTimeoutError' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Exit demo' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'ignored', exact: true })).toBeVisible()
  await expect.poll(() => page.locator('.terrarium__image').evaluate((image: HTMLImageElement) => image.naturalWidth)).toBeGreaterThan(0)

  const habitatBugs = page.locator('.terrarium .bug')
  await expect(habitatBugs).toHaveCount(3)
  await habitatBugs.nth(1).click()
  await expect(page.getByRole('heading', { name: 'UpstreamConnectionError' })).toBeVisible()
  await habitatBugs.nth(1).click()
  await expect(page.getByRole('heading', { name: 'UpstreamConnectionError' })).toBeVisible()

  await page.locator('#project-picker').selectOption('')
  await expect(habitatBugs).toHaveCount(0)
  await page.locator('#project-picker').selectOption('1')
  await expect(habitatBugs).toHaveCount(3)
  await expect(page.getByRole('heading', { name: 'DatabaseTimeoutError' })).toBeVisible()

  await page.getByRole('button', { name: 'resolved', exact: true }).click()
  await expect(habitatBugs).toHaveCount(3)
  await expect(page.getByText('No resolved issues')).toBeVisible()

  await page.getByRole('button', { name: 'Terrarium', exact: true }).click()
  await expect(page.locator('.flytrap-main h1')).toHaveText('Terrarium')
  await expect(page.getByRole('button', { name: 'Terrarium', exact: true })).toHaveAttribute('aria-current', 'page')

  await page.getByRole('button', { name: 'Projects', exact: true }).click()
  await expect(page.locator('.flytrap-main h1')).toHaveText('Projects')
  await page.getByRole('button', { name: /Checkout API/ }).last().click()
  await expect(page.locator('.flytrap-main h1')).toHaveText('Issues')
  await expect(habitatBugs).toHaveCount(3)

  await page.getByRole('button', { name: 'Open in Editor', exact: true }).click()
  await expect(page.getByRole('dialog', { name: 'Open source in your editor' })).toBeVisible()
  await page.getByLabel('Local source root for this project').fill('/work/checkout')
  await page.getByRole('button', { name: 'Save settings' }).click()
  await expect(page.getByRole('link', { name: 'Open in Editor' })).toHaveAttribute(
    'href',
    'vscode://file//work/checkout/db/client.go:42:1',
  )

  await page.getByRole('button', { name: 'Editor settings' }).click()
  await page.getByLabel('Editor', { exact: true }).selectOption('jetbrains')
  await page.getByLabel('JetBrains IDE', { exact: true }).selectOption('goland')
  await expect(page.getByLabel('IDE project name', { exact: true })).toHaveValue('checkout')
  await page.getByRole('button', { name: 'Save settings' }).click()
  await expect(page.getByRole('link', { name: 'Open in Editor' })).toHaveAttribute(
    'href',
    'jetbrains://goland/navigate/reference?project=checkout&path=%2Fwork%2Fcheckout%2Fdb%2Fclient.go:42:1',
  )

  await page.getByRole('button', { name: 'Editor settings' }).click()
  await page.getByLabel('Editor', { exact: true }).selectOption('zed')
  await page.getByRole('button', { name: 'Save settings' }).click()
  await expect(page.getByRole('link', { name: 'Open in Editor' })).toHaveAttribute(
    'href',
    'zed://file/work/checkout/db/client.go:42:1',
  )
  await page.screenshot({ path: testInfo.project.name === 'mobile' ? '/tmp/flytrap-frontend-mobile.png' : '/tmp/flytrap-frontend-desktop.png', fullPage: true })
})

test('login, project loading, pagination, issue detail, and status update follow the API contract', async ({ page }) => {
  let listCalls = 0
  let updateCalls = 0
  let realtimeConnections = 0
  let currentStatus = 'open'
  await page.route('**/api/v1/auth/login', async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(session) }))
  await page.route('**/api/v1/projects/12/issues**', async (route) => {
    if (route.request().method() === 'PATCH') {
      updateCalls += 1
      currentStatus = 'resolved'
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ...issue, status: currentStatus }) })
    }
    listCalls += 1
    const hasCursor = new URL(route.request().url()).searchParams.has('cursor')
    const issues = hasCursor ? [secondIssue] : [{ ...issue, status: currentStatus }]
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ issues, next_cursor: hasCursor ? undefined : 'next-page' }) })
  })
  await page.route('**/api/v1/projects/12/issues/42', async (route) => {
    if (route.request().method() === 'PATCH') {
      updateCalls += 1
      currentStatus = 'resolved'
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ...issue, status: currentStatus }) })
    }
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ...issue, status: currentStatus, environments: ['production'], releases: ['1.2.0'] }) })
  })
  await page.routeWebSocket('**/api/v1/projects/12/realtime', (socket) => {
    realtimeConnections += 1
    socket.onMessage((raw) => {
      const message = JSON.parse(String(raw)) as { token?: string }
      if (message.token) socket.send(JSON.stringify({ type: 'realtime.ready', project_id: 12 }))
    })
  })
  await page.route('**/api/v1/projects', async (route) => {
    if (route.request().method() === 'GET') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([project]) })
    return route.continue()
  })

  await page.goto('/')
  await page.getByLabel('Email').fill(user.email)
  await page.getByLabel('Password').fill('correct horse battery staple')
  await page.getByRole('button', { name: 'Log in' }).click()
  await expect(page.locator('select').first()).toBeEnabled()
  await page.locator('select').first().selectOption('12')
  await expect.poll(() => listCalls).toBeGreaterThan(0)
  await expect(page.getByText('TypeError')).toBeVisible()
  await expect.poll(() => realtimeConnections).toBeGreaterThan(0)
  const connectionsAfterReady = realtimeConnections
  await page.locator('.issue-list__item').filter({ hasText: 'TypeError' }).click()
  await expect(page.getByRole('heading', { name: 'TypeError' })).toBeVisible()
  await expect(page.getByText('production')).toBeVisible()
  await page.getByRole('button', { name: 'Load more' }).click()
  await expect.poll(() => listCalls).toBeGreaterThan(1)
  await expect(page.getByText('RangeError')).toBeVisible()
  await page.getByRole('button', { name: 'Resolve', exact: true }).click()
  await expect.poll(() => updateCalls).toBe(1)
  await expect(page.getByText('resolved').first()).toBeVisible()
  await expect.poll(() => realtimeConnections).toBe(connectionsAfterReady)
})
