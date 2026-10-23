import { expect, test } from '@playwright/test'

const base = 'http://127.0.0.1:4180/?mock'

const frame = {
  seq: 1,
  title: 'Color AFVI Camera Studio',
  meta: { profile: 'studio', theme: 'light', lang: 'zh', fontPx: 16, role: 2 },
  slots: {
    menu: { k: 'root', c: [{ k: 'menu', id: 'm-file', l: '文件(F)', c: [{ k: 'mitem', id: 'menu:exit', l: '退出' }] }] },
    tabs: { k: 'root', c: [{ k: 'tabs', id: 'tabs', v: 0, p: { items: ['光照与 GV', '缺陷合成'] } }] },
    left: {
      k: 'root',
      c: [
        { k: 'section', id: 'sec-a', l: '光源', v: true, c: [
          { k: 'slider', id: 'g', l: '增益', v: 1.5, p: { min: 0, max: 4, fmt: '%.2f' } },
          { k: 'check', id: 'c', l: '显示材料图', v: true },
          { k: 'combo', id: 'cb', l: '显示', v: 1, p: { items: ['Red', 'Green', 'Blue'] } },
        ] },
        { k: 'group', id: 'gb-x', l: '分组', c: [{ k: 'button', id: 'b', l: '恢复默认', p: { enabled: true } }] },
      ],
    },
    right: { k: 'root', c: [{ k: 'table', id: 't', v: 1, p: { headers: ['通道', '均值'], rows: [[{ t: 'Red' }, { t: '12.50' }], [{ t: 'Green' }, { t: '33.10' }]], height: 4 } }] },
    status: { k: 'root', c: [{ k: 'status', l: '就绪', c: [{ k: 'text', l: '缩放 100%', p: { w: 100 } }] }] },
    palette: { k: 'root', c: [{ k: 'palette', p: { items: [{ id: 'pal-rect', icon: 'rect', active: true, enabled: true, tip: '矩形框' }] } }] },
  },
}

test('renders a recorded frame with the win32 components', async ({ page }) => {
  await page.goto(base)
  await page.evaluate((f) => (window as any).__inject(f), frame)
  await expect(page.locator('.w-titlebar')).toContainText('Camera Studio')
  await expect(page.locator('[data-id="g"]')).toContainText('增益')
  await expect(page.locator('.n-sec-head')).toContainText('光源')
  await expect(page.locator('.n-table tbody tr')).toHaveCount(2)
  await expect(page.locator('.n-table tr.sel')).toContainText('Green')
  await expect(page.locator('.pal-btn.active')).toHaveCount(1)
  // design constitution: no rounded corners / blurred shadows on the controls we render
  const bad = await page.evaluate(() => {
    const out: string[] = []
    document.querySelectorAll('.shell *').forEach((el) => {
      const cs = getComputedStyle(el)
      if (el.tagName === 'CANVAS' || el.classList.contains('n-heat-peak')) return
      if (cs.borderRadius !== '0px') out.push(el.className + ':' + cs.borderRadius)
    })
    return out
  })
  expect(bad).toEqual([])
  await page.screenshot({ path: 'test-results/mock-studio.png' })
})

test('commands go to /cmd as coalesced batches', async ({ page }) => {
  const posts: any[] = []
  await page.route('**/cmd', async (route) => {
    posts.push(JSON.parse(route.request().postData() || '[]'))
    await route.fulfill({ status: 204 })
  })
  await page.goto(base)
  await page.evaluate((f) => (window as any).__inject(f), frame)
  await page.locator('[data-id="b"]').click()
  await page.locator('.n-tab', { hasText: '缺陷合成' }).click()
  await expect.poll(() => posts.flat().some((c: any) => c.id === 'tabs')).toBe(true)
  const all = posts.flat()
  expect(all).toContainEqual({ id: 'b', op: 'click' })
  expect(all).toContainEqual({ id: 'tabs', op: 'set', v: 1 })
})
