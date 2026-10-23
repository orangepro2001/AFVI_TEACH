import { chromium } from '@playwright/test'
const url = process.argv[2] || 'http://127.0.0.1:47701/'
const b = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox'] })
const p = await b.newPage({ viewport: { width: 1600, height: 1000 } })
await p.addInitScript(() => {
  const w = window
  w.__log = []
  const of = w.fetch
  w.fetch = async (u, o) => {
    const t0 = performance.now()
    const r = await of(u, o)
    const name = String(u).split('?')[0]
    if (name === 'img') { const buf = await r.clone().arrayBuffer(); w.__log.push({ k: 'img', t0, ms: performance.now() - t0, bytes: buf.byteLength }) }
    else w.__log.push({ k: name, t0, t1: performance.now(), status: r.status })
    return r
  }
})
await p.goto(url)
await p.waitForFunction(() => window.__viewer && window.__viewer().imgW > 0, null, { timeout: 60000 })
await p.waitForTimeout(1500)
console.log('stats idle', await (await fetch(url + 'stats')).text())

// 1. slider drag round trip: pointermove -> next poll response
const gain = p.locator('[data-id="gain"] .w-trackbar__track')
const box = await gain.boundingBox()
await p.evaluate(() => (window.__log.length = 0))
await p.mouse.move(box.x + box.width * 0.3, box.y + 3)
await p.mouse.down()
const t0 = Date.now()
for (let i = 0; i <= 30; i++) { await p.mouse.move(box.x + box.width * (0.3 + 0.4 * i / 30), box.y + 3); await p.waitForTimeout(16) }
await p.mouse.up()
await p.waitForTimeout(800)
const log1 = await p.evaluate(() => window.__log.slice())
const cmds = log1.filter((l) => l.k === 'cmd'), polls = log1.filter((l) => l.k === 'poll')
const lat = cmds.map((c) => { const n = polls.find((q) => q.t1 > c.t0); return n ? n.t1 - c.t0 : null }).filter((x) => x != null).sort((a, b) => a - b)
console.log(`slider: ${cmds.length} cmds, ${polls.length} frames; cmd->frame latency ms p50=${lat[lat.length >> 1]?.toFixed(1)} max=${lat[lat.length - 1]?.toFixed(1)}; imgs=${log1.filter(l => l.k === 'img').length}`)
console.log('img fetches', JSON.stringify(log1.filter((l) => l.k === 'img').map((l) => ({ ms: +l.ms.toFixed(0), MB: +(l.bytes / 1e6).toFixed(1) }))))

// 2. image change (display channel combo) -> time until canvas updated
await p.evaluate(() => { window.__log.length = 0 })
const tA = await p.evaluate(() => performance.now())
await p.locator('[data-id="showmat"] input').click()
await p.waitForFunction((t) => window.__log.some((l) => l.k === 'img'), tA, { timeout: 30000 })
await p.waitForTimeout(800)
const log2 = await p.evaluate(() => window.__log.slice())
console.log('toggle material view:', JSON.stringify(log2.filter(l => l.k === 'img').map((l) => ({ fetchMs: +l.ms.toFixed(0), MB: +(l.bytes / 1e6).toFixed(1) }))))

// 3. pan: drag on canvas, measure rAF intervals + draw cost
const cv = p.locator('canvas.viewer-canvas'); const cb = await cv.boundingBox()
await p.evaluate(() => { window.__frames = []; let last = performance.now(); const f = () => { const n = performance.now(); window.__frames.push(n - last); last = n; window.__raf = requestAnimationFrame(f) }; window.__raf = requestAnimationFrame(f) })
await p.mouse.move(cb.x + 300, cb.y + 300)
await p.mouse.down({ button: 'middle' })
for (let i = 0; i < 60; i++) { await p.mouse.move(cb.x + 300 + i * 4, cb.y + 300 + i * 2) }
await p.mouse.up({ button: 'middle' })
await p.waitForTimeout(300)
const fr = await p.evaluate(() => { cancelAnimationFrame(window.__raf); return window.__frames.slice() })
const s = fr.slice().sort((a, b) => a - b)
console.log(`pan: ${fr.length} rAF frames, median ${s[s.length >> 1]?.toFixed(1)}ms, p95 ${s[Math.floor(s.length * 0.95)]?.toFixed(1)}ms, max ${s[s.length - 1]?.toFixed(1)}ms`)

// 4. wheel zoom
await p.evaluate(() => { window.__frames = []; let last = performance.now(); const f = () => { const n = performance.now(); window.__frames.push(n - last); last = n; window.__raf = requestAnimationFrame(f) }; window.__raf = requestAnimationFrame(f) })
for (let i = 0; i < 20; i++) { await p.mouse.wheel(0, -100); await p.waitForTimeout(10) }
await p.waitForTimeout(300)
const fz = await p.evaluate(() => { cancelAnimationFrame(window.__raf); return window.__frames.slice() })
const sz = fz.slice().sort((a, b) => a - b)
console.log(`zoom: median ${sz[sz.length >> 1]?.toFixed(1)}ms p95 ${sz[Math.floor(sz.length * 0.95)]?.toFixed(1)}ms max ${sz[sz.length - 1]?.toFixed(1)}ms`)
console.log('stats end', await (await fetch(url + 'stats')).text())
await b.close()
