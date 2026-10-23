<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Shape, ViewerOut } from '../types'
import { send } from '../bridge'

const props = defineProps<{ viewer: ViewerOut | undefined; background?: string }>()

const wrap = ref<HTMLDivElement>()
const cv = ref<HTMLCanvasElement>()

// ---- image pyramid ---------------------------------------------------------------
// The host serves the image at levels 0..5 (1/2^k size). We draw from the coarsest level
// that still has a pixel per screen pixel and only crop what is visible, so pan/zoom never
// scale the full 23-megapixel image, and while a slider streams new images only the small
// levels cross the wire (the sharp level follows once things settle).
interface Lvl { c: HTMLCanvasElement; w: number; h: number }
let levels = new Map<number, Lvl>() // levels of `shownRev`
let shownRev = -1
let wantRev = -1
let imgW = 0
let imgH = 0
let zoom = 1
let panX = 0
let panY = 0
let cw = 0
let ch = 0
let lastFit = -1
let lastZoomOn = 0
let raf = 0
let ctl: AbortController | null = null
let sharpTimer: ReturnType<typeof setTimeout> | undefined
const MAXL = 5

function sendView() {
  if (viewTimer) return
  viewTimer = setTimeout(() => {
    viewTimer = undefined
    send({ id: 'viewer', op: 'view', v: { zoom, px: panX, py: panY, w: cw, h: ch } })
  }, 60)
}
let viewTimer: ReturnType<typeof setTimeout> | undefined

function schedule() {
  if (!raf) raf = requestAnimationFrame(() => { raf = 0; draw() })
}

function fit() {
  if (!imgW || !cw) return
  zoom = Math.min(cw / imgW, ch / imgH) * 0.98
  panX = (cw - imgW * zoom) / 2
  panY = (ch - imgH * zoom) / 2
  sendView()
  schedule()
  wantSharp()
}

function zoomAt(sx: number, sy: number, factor: number) {
  const nz = Math.min(64, Math.max(0.02, zoom * factor))
  const ix = (sx - panX) / zoom
  const iy = (sy - panY) / zoom
  zoom = nz
  panX = sx - ix * zoom
  panY = sy - iy * zoom
  sendView()
  schedule()
  wantSharp()
}

// level whose resolution is just enough for the current zoom
function levelFor(z: number) {
  if (z >= 1) return 0
  return Math.min(MAXL, Math.max(0, Math.floor(Math.log2(1 / z))))
}

async function fetchLevel(rev: number, k: number, signal: AbortSignal): Promise<Lvl | null> {
  const r = await fetch(`img?rev=${rev}&lvl=${k}`, { cache: 'no-store', signal })
  if (!r.ok) return null
  const w = +(r.headers.get('X-W') ?? 0)
  const h = +(r.headers.get('X-H') ?? 0)
  const gray = r.headers.get('X-Fmt') === 'gray'
  const buf = new Uint8Array(await r.arrayBuffer())
  let rgba: Uint8ClampedArray
  if (gray) {
    rgba = new Uint8ClampedArray(w * h * 4)
    const d = new Uint32Array(rgba.buffer)
    for (let i = 0; i < w * h; i++) { const v = buf[i]; d[i] = 0xff000000 | (v << 16) | (v << 8) | v }
  } else {
    rgba = new Uint8ClampedArray(buf.buffer, buf.byteOffset, w * h * 4)
  }
  const c = document.createElement('canvas')
  c.width = w
  c.height = h
  c.getContext('2d')!.putImageData(new ImageData(rgba as any, w, h), 0, 0)
  return { c, w, h }
}

// Load the level the view needs for revision `rev`: a quick coarse preview first (so a
// stream of new images stays cheap), the sharp level once the image has been stable.
async function loadRev(rev: number, sharpDelay: number) {
  ctl?.abort()
  ctl = new AbortController()
  const signal = ctl.signal
  wantRev = rev
  const want = levelFor(zoom)
  const preview = Math.max(want, Math.min(MAXL, 3))
  try {
    if (!levels.has(preview) || shownRev !== rev) {
      const l = await fetchLevel(rev, preview, signal)
      if (l && wantRev === rev) adopt(rev, preview, l)
    }
    if (want < preview) {
      await new Promise((res) => (sharpTimer = setTimeout(res, sharpDelay)))
      if (wantRev !== rev || signal.aborted) return
      const k = levelFor(zoom)
      if (!levels.has(k)) {
        const l = await fetchLevel(rev, k, signal)
        if (l && wantRev === rev) adopt(rev, k, l)
      }
    }
  } catch { /* aborted or retried by the next frame */ }
}

function adopt(rev: number, k: number, l: Lvl) {
  if (shownRev !== rev) { levels = new Map(); shownRev = rev }
  levels.set(k, l)
  const first = imgW === 0
  imgW = props.viewer?.w ?? imgW
  imgH = props.viewer?.h ?? imgH
  if (first || props.viewer?.fitSeq !== lastFit) { lastFit = props.viewer?.fitSeq ?? lastFit; fit() }
  schedule()
}

// the zoom changed: make sure the level it needs is loaded (debounced)
function wantSharp() {
  if (shownRev < 0) return
  const k = levelFor(zoom)
  if (levels.has(k)) return
  clearTimeout(sharpTimer)
  sharpTimer = setTimeout(async () => {
    const rev = shownRev
    const kk = levelFor(zoom)
    if (levels.has(kk) || rev !== wantRev) return
    ctl?.abort()
    ctl = new AbortController()
    try {
      const l = await fetchLevel(rev, kk, ctl.signal)
      if (l && shownRev === rev && wantRev === rev) { levels.set(kk, l); schedule() }
    } catch { /* aborted */ }
  }, 150)
}

// best loaded level for the current zoom: the coarsest one that is at least as sharp as
// the screen, else the sharpest available
function pickLevel(): { k: number; l: Lvl } | null {
  const want = levelFor(zoom)
  let best: { k: number; l: Lvl } | null = null
  for (const [k, l] of levels) {
    if (k <= want) { if (!best || k > best.k || best.k > want) best = { k, l } }
  }
  if (best) return best
  for (const [k, l] of levels) if (!best || k < best.k) best = { k, l }
  return best
}

function strokeStyle(ctx: CanvasRenderingContext2D, s: Shape) {
  ctx.strokeStyle = s.col || '#00ff00'
  ctx.lineWidth = s.th || 1.5
  ctx.setLineDash(s.dash ? [5, 4] : [])
}

function draw() {
  const c = cv.value
  if (!c) return
  const dpr = window.devicePixelRatio || 1
  if (c.width !== Math.round(cw * dpr) || c.height !== Math.round(ch * dpr)) {
    c.width = Math.round(cw * dpr)
    c.height = Math.round(ch * dpr)
  }
  const ctx = c.getContext('2d')!
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.fillStyle = props.background || '#808080'
  ctx.fillRect(0, 0, cw, ch)
  const pl = pickLevel()
  if (pl && imgW) {
    // crop to the visible part of the level so large zooms only touch a few pixels
    const f = pl.l.w / imgW // level pixels per image pixel
    const vx0 = Math.max(0, -panX / zoom), vy0 = Math.max(0, -panY / zoom)
    const vx1 = Math.min(imgW, (cw - panX) / zoom), vy1 = Math.min(imgH, (ch - panY) / zoom)
    if (vx1 > vx0 && vy1 > vy0) {
      const sx = Math.floor(vx0 * f), sy = Math.floor(vy0 * f)
      const sx1 = Math.min(pl.l.w, Math.ceil(vx1 * f)), sy1 = Math.min(pl.l.h, Math.ceil(vy1 * f))
      ctx.imageSmoothingEnabled = zoom * (imgW / pl.l.w) < 1
      ctx.drawImage(pl.l.c, sx, sy, sx1 - sx, sy1 - sy,
        panX + (sx / f) * zoom, panY + (sy / f) * zoom, ((sx1 - sx) / f) * zoom, ((sy1 - sy) / f) * zoom)
    }
    ctx.strokeStyle = '#404040'
    ctx.lineWidth = 1
    ctx.setLineDash([])
    ctx.strokeRect(panX - 0.5, panY - 0.5, imgW * zoom + 1, imgH * zoom + 1)
  }
  const X = (x: number) => panX + x * zoom
  const Y = (y: number) => panY + y * zoom
  for (const s of props.viewer?.shapes ?? []) {
    ctx.save()
    switch (s.t) {
      case 'rect': {
        strokeStyle(ctx, s)
        let w = (s.w ?? 0) * zoom
        let h = (s.h ?? 0) * zoom
        let x = X(s.x)
        let y = Y(s.y)
        if (w < 3) { x -= (3 - w) / 2; w = 3 }
        if (h < 3) { y -= (3 - h) / 2; h = 3 }
        ctx.strokeRect(x, y, w, h)
        break
      }
      case 'fillrect':
        ctx.fillStyle = s.fill || 'rgba(255,0,0,0.25)'
        ctx.fillRect(X(s.x), Y(s.y), (s.w ?? 0) * zoom, (s.h ?? 0) * zoom)
        break
      case 'ellipse':
        strokeStyle(ctx, s)
        ctx.beginPath()
        ctx.ellipse(X(s.x + (s.w ?? 0) / 2), Y(s.y + (s.h ?? 0) / 2), Math.abs((s.w ?? 0) * zoom) / 2, Math.abs((s.h ?? 0) * zoom) / 2, 0, 0, Math.PI * 2)
        ctx.stroke()
        break
      case 'poly':
      case 'polyline': {
        const pts = s.pts ?? []
        if (!pts.length) break
        ctx.beginPath()
        pts.forEach((q, i) => (i ? ctx.lineTo(X(q[0]), Y(q[1])) : ctx.moveTo(X(q[0]), Y(q[1]))))
        if (s.t === 'poly') ctx.closePath()
        if (s.fill) { ctx.fillStyle = s.fill; ctx.fill() }
        strokeStyle(ctx, s)
        ctx.stroke()
        break
      }
      case 'line':
        strokeStyle(ctx, s)
        ctx.beginPath()
        ctx.moveTo(X(s.x), Y(s.y))
        ctx.lineTo(X(s.x2 ?? 0), Y(s.y2 ?? 0))
        ctx.stroke()
        break
      case 'dot':
        ctx.fillStyle = s.col || '#ffff00'
        ctx.beginPath()
        ctx.arc(X(s.x), Y(s.y), s.r || 3, 0, Math.PI * 2)
        ctx.fill()
        break
      case 'circle':
        strokeStyle(ctx, s)
        ctx.beginPath()
        ctx.arc(X(s.x), Y(s.y), Math.max(1.5, (s.r ?? 0) * zoom), 0, Math.PI * 2)
        ctx.stroke()
        break
      case 'text': {
        ctx.font = '12px Tahoma, sans-serif'
        ctx.textBaseline = 'top'
        const tx = X(s.x) + (s.x2 ?? 0)
        const ty = Y(s.y) + (s.y2 ?? 0)
        ctx.fillStyle = '#000'
        ctx.fillText(s.text ?? '', tx + 1, ty + 1)
        ctx.fillStyle = s.col || '#ffff00'
        ctx.fillText(s.text ?? '', tx, ty)
        break
      }
    }
    ctx.restore()
  }
}

// ---- input ----------------------------------------------------------------------
let mode: '' | 'pan' | 'tool' = ''
let button = -1
let startX = 0
let startY = 0
let moved = 0
let lastX = 0
let lastY = 0

function pos(e: MouseEvent) {
  const r = cv.value!.getBoundingClientRect()
  return { sx: e.clientX - r.left, sy: e.clientY - r.top }
}
function img(sx: number, sy: number) {
  return { x: (sx - panX) / zoom, y: (sy - panY) / zoom }
}

function onDown(e: PointerEvent) {
  const { sx, sy } = pos(e)
  cv.value!.setPointerCapture(e.pointerId)
  button = e.button
  startX = lastX = sx
  startY = lastY = sy
  moved = 0
  if (e.shiftKey && (e.button === 0 || e.button === 2)) {
    zoomAt(sx, sy, e.button === 0 ? 1.5 : 1 / 1.5)
    mode = ''
    button = -1
    return
  }
  const i = img(sx, sy)
  if (e.button === 0 && props.viewer && (props.viewer as any).claim) {
    mode = 'tool'
    send({ id: 'viewer', op: 'ptr', k: 'down', x: i.x, y: i.y, ctrl: e.ctrlKey, shift: e.shiftKey })
  } else {
    mode = 'pan'
    // a panel may still react to a press it does not own (point pick); it gets no move/up
    if (e.button === 0) send({ id: 'viewer', op: 'ptr', k: 'down', x: i.x, y: i.y, ctrl: e.ctrlKey, shift: e.shiftKey })
  }
}
function onMove(e: PointerEvent) {
  const { sx, sy } = pos(e)
  const i = img(sx, sy)
  if (mode === 'pan') {
    panX += sx - lastX
    panY += sy - lastY
    moved += Math.abs(sx - lastX) + Math.abs(sy - lastY)
    sendView()
    schedule()
  } else if (mode === 'tool') {
    send({ id: 'viewer', op: 'ptr', k: 'move', x: i.x, y: i.y, ctrl: e.ctrlKey, shift: e.shiftKey })
  } else {
    send({ id: 'viewer', op: 'hover', x: i.x, y: i.y })
  }
  lastX = sx
  lastY = sy
}
function onUp(e: PointerEvent) {
  const { sx, sy } = pos(e)
  const i = img(sx, sy)
  if (mode === 'tool') send({ id: 'viewer', op: 'ptr', k: 'up', x: i.x, y: i.y, ctrl: e.ctrlKey, shift: e.shiftKey })
  if (button === 2 && Math.hypot(sx - startX, sy - startY) < 4 && !e.shiftKey)
    send({ id: 'viewer', op: 'ptr', k: 'rclick', x: i.x, y: i.y })
  mode = ''
  button = -1
}
function onWheel(e: WheelEvent) {
  e.preventDefault()
  const { sx, sy } = pos(e)
  zoomAt(sx, sy, e.deltaY < 0 ? 1.2 : 1 / 1.2)
}
function onDbl() {
  if (mode !== 'tool' && !props.viewer?.noDblFit) fit()
}
function onLeave() {
  send({ id: 'viewer', op: 'hover', k: 'leave', x: 0, y: 0 })
}

let ro: ResizeObserver | null = null
onMounted(() => {
  ro = new ResizeObserver(() => {
    const r = wrap.value!.getBoundingClientRect()
    const first = cw === 0
    cw = Math.max(1, Math.floor(r.width))
    ch = Math.max(1, Math.floor(r.height))
    if (first && imgW && shownRev >= 0) fit()
    sendView()
    schedule()
  })
  ro.observe(wrap.value!)
  ;(window as any).__viewer = () => ({ zoom, panX, panY, cw, ch, imgW, imgH })
})
onBeforeUnmount(() => ro?.disconnect())

watch(
  () => props.viewer,
  (v) => {
    if (!v) return
    if (v.w === 0) { levels = new Map(); shownRev = -1; wantRev = -1; imgW = 0; imgH = 0; schedule(); return }
    imgW = v.w
    imgH = v.h
    if (v.rev !== wantRev) loadRev(v.rev, 250)
    if (v.fitSeq !== lastFit && shownRev >= 0) { lastFit = v.fitSeq; fit() }
    if (v.zoomOn && v.zoomOnSeq !== lastZoomOn) {
      lastZoomOn = v.zoomOnSeq
      if (v.zoomOn[2] > 0) zoom = v.zoomOn[2]
      panX = cw / 2 - v.zoomOn[0] * zoom
      panY = ch / 2 - v.zoomOn[1] * zoom
      sendView()
      wantSharp()
    }
    schedule()
  },
  { immediate: true },
)
</script>

<template>
  <div ref="wrap" class="viewer-wrap">
    <canvas
      ref="cv"
      class="viewer-canvas"
      :style="{ cursor: viewer?.cursor || 'default' }"
      @pointerdown="onDown"
      @pointermove="onMove"
      @pointerup="onUp"
      @pointerleave="onLeave"
      @wheel="onWheel"
      @dblclick="onDbl"
      @contextmenu.prevent
    />
  </div>
</template>

<style scoped>
.viewer-wrap { position: absolute; inset: 0; }
.viewer-canvas { width: 100%; height: 100%; display: block; touch-action: none; }
</style>
