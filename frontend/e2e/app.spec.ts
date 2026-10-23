import { expect, test, type Page } from '@playwright/test'

const STUDIO = 'http://127.0.0.1:47701/'
const TEACH = 'http://127.0.0.1:47702/'

async function noErrors(page: Page) {
  const errs: string[] = []
  page.on('pageerror', (e) => errs.push(e.message))
  return errs
}

async function waitImage(page: Page) {
  await expect.poll(async () => page.evaluate(() => (window as any).__viewer?.().imgW ?? 0), { timeout: 60_000 }).toBeGreaterThan(0)
}

test.describe('camera studio (real backend)', () => {
  test('loads the model image and moves a slider', async ({ page }) => {
    const errs = await noErrors(page)
    await page.goto(STUDIO)
    await waitImage(page)
    const gain = page.locator('[data-id="gain"]')
    await expect(gain).toContainText('1.00')
    const box = (await gain.locator('.w-trackbar__track').boundingBox())!
    await page.mouse.click(box.x + box.width * 0.75, box.y + box.height / 2)
    await expect(gain.locator('.n-value')).not.toHaveText('1.00')
    expect(errs).toEqual([])
  })

  test('light channel combo and stats table follow the host', async ({ page }) => {
    await page.goto(STUDIO)
    await waitImage(page)
    await expect(page.locator('[data-id="light-stats"] tbody tr')).toHaveCount(1, { timeout: 30_000 })
    await page.locator('[data-id="showmat"] input').check()
    await expect(page.locator('[data-id="showmat"] input')).toBeChecked()
  })

  test('defect tab: paint a bright spot and save the scenario', async ({ page }) => {
    await page.goto(STUDIO)
    await waitImage(page)
    await page.locator('.n-tab', { hasText: '缺陷合成' }).click()
    await expect(page.locator('[data-id="sec-defect-brush"]')).toBeVisible()
    const rows = () => page.locator('[data-id^="defect-del-"]').count()
    const before = await rows()
    await page.locator('[data-id="defect-tool-0"]').click()
    const canvas = page.locator('canvas.viewer-canvas')
    const b = (await canvas.boundingBox())!
    await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2)
    await page.waitForTimeout(400) // hover reaches the host before the press (drag claim)
    await page.mouse.click(b.x + b.width / 2, b.y + b.height / 2)
    await expect.poll(rows).toBeGreaterThan(before)
  })
})

test.describe('teach program (real backend)', () => {
  test('operator pages, admin login and logout', async ({ page }) => {
    const errs = await noErrors(page)
    await page.goto(TEACH)
    await waitImage(page)
    for (const t of ['对位', '金板', '检测']) await expect(page.locator('.n-tab', { hasText: t })).toBeVisible()
    await expect(page.locator('.n-tab', { hasText: '对位试验台' })).toHaveCount(0)
    await page.getByText('登录…').first().click()
    await page.locator('[data-id="login-user"] input').fill('admin')
    await page.locator('[data-id="login-pass"] input').fill('1')
    await page.locator('[data-id="login-ok"]').click()
    await expect(page.locator('.n-tab', { hasText: '对位试验台' })).toBeVisible()
    await expect(page.locator('.n-tab', { hasText: '缺陷检出' })).toBeVisible()
    await page.getByText('注销').first().click()
    await expect(page.locator('.n-tab', { hasText: '对位试验台' })).toHaveCount(0)
    expect(errs).toEqual([])
  })

  test('operator confirms the alignment ROIs', async ({ page }) => {
    await page.goto(TEACH)
    await waitImage(page)
    await page.locator('[data-id="op-align-confirm"], [data-id*="confirm"]').first().click()
    await expect(page.getByText(/已对位|对位成功|对位完成/).first()).toBeVisible({ timeout: 60_000 })
  })
})
