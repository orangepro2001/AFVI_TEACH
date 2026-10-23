import { reactive, ref, shallowRef } from 'vue'
import type { Cmd, Frame } from './types'

// One HTTP transport for both the Wails window (AssetServer handler) and the browser:
// commands are POSTed to /cmd, frames are long-polled from /poll, images come from /img.
export const frame = shallowRef<Frame | null>(null)
export const connected = ref(false)
// Locally edited values (sliders, text fields) shown until the host echoes them back.
export const local = reactive<Record<string, { v: any; t: number }>>({})

let queue: Cmd[] = []
let scheduled: 'micro' | 'timer' | null = null
let timer: ReturnType<typeof setTimeout> | undefined
const latestSet = new Map<string, number>()
const FRAME_MS = 16

// Commands that only carry the newest value (slider drags, pointer moves, hover, view) are
// batched once per animation frame; anything else (clicks, keys, tool presses) goes out at
// the end of the current task. Fewer, larger requests matter most in the Wails window,
// where every request crosses the WebView2 resource pipeline.
export function send(c: Cmd) {
  const coalesce = c.op === 'set' || c.op === 'view' || c.op === 'hover' || (c.op === 'ptr' && c.k === 'move')
  if (coalesce) {
    const key = c.op === 'set' ? `set:${c.id}` : c.op === 'ptr' ? 'ptr:move' : c.op
    const at = latestSet.get(key)
    if (at !== undefined) queue[at] = c
    else {
      latestSet.set(key, queue.length)
      queue.push(c)
    }
    if (!scheduled) {
      scheduled = 'timer'
      timer = setTimeout(flush, FRAME_MS)
    }
    return
  }
  queue.push(c)
  if (scheduled !== 'micro') {
    if (timer) clearTimeout(timer)
    scheduled = 'micro'
    queueMicrotask(flush)
  }
}

export function setLocal(id: string, v: any) {
  local[id] = { v, t: Date.now() }
}

export function shown<T>(id: string | undefined, serverValue: T): T {
  if (!id) return serverValue
  const l = local[id]
  if (!l) return serverValue
  if (l.v === serverValue || Date.now() - l.t > 600) {
    delete local[id]
    return serverValue
  }
  return l.v as T
}

async function flush() {
  scheduled = null
  timer = undefined
  const batch = queue
  queue = []
  latestSet.clear()
  if (!batch.length) return
  try {
    await fetch('cmd', { method: 'POST', body: JSON.stringify(batch), headers: { 'Content-Type': 'application/json' } })
  } catch {
    connected.value = false
  }
}

let started = false
export async function start() {
  if (started) return
  started = true
  let seq = 0
  for (;;) {
    try {
      const r = await fetch(`poll?seq=${seq}`, { cache: 'no-store' })
      connected.value = true
      if (r.status === 200) {
        const f = (await r.json()) as Frame
        seq = f.seq
        frame.value = f
      }
    } catch {
      connected.value = false
      await new Promise((res) => setTimeout(res, 800))
    }
  }
}

// Test hook: feed a frame without a backend.
export function injectFrame(f: Frame) {
  frame.value = f
  connected.value = true
}
