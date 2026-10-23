<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { formatValue } from '../utils/format'
import type { WGaugeZone } from '../types'

const props = withDefaults(
  defineProps<{
    value?: number
    min?: number
    max?: number
    unit?: string
    label?: string
    zones?: WGaugeZone[]
    size?: number
  }>(),
  {
    value: 0,
    min: 0,
    max: 100,
    unit: '',
    label: '',
    zones: () => [],
    size: 120,
  },
)

const canvas = ref<HTMLCanvasElement | null>(null)

const ratio = computed(() => {
  const span = props.max - props.min
  if (span <= 0) return 0
  return Math.min(1, Math.max(0, (props.value - props.min) / span))
})

const START = Math.PI * 0.75
const SWEEP = Math.PI * 1.5

function draw() {
  const el = canvas.value
  if (!el) return
  const dpr = window.devicePixelRatio || 1
  const s = props.size
  el.width = s * dpr
  el.height = s * dpr
  el.style.width = `${s}px`
  el.style.height = `${s}px`
  const ctx = el.getContext('2d')
  if (!ctx) return
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, s, s)

  const cx = s / 2
  const cy = s / 2
  const r = s / 2 - 8

  // Face
  ctx.fillStyle = '#ffffff'
  ctx.beginPath()
  ctx.arc(cx, cy, r + 5, 0, Math.PI * 2)
  ctx.fill()

  // Bezel
  ctx.strokeStyle = '#808080'
  ctx.lineWidth = 2
  ctx.beginPath()
  ctx.arc(cx, cy, r + 5, 0, Math.PI * 2)
  ctx.stroke()
  ctx.strokeStyle = '#ffffff'
  ctx.lineWidth = 1
  ctx.beginPath()
  ctx.arc(cx, cy, r + 3.5, 0, Math.PI * 2)
  ctx.stroke()

  // Zones
  for (const z of props.zones) {
    const t0 = (z.from - props.min) / (props.max - props.min)
    const t1 = (z.to - props.min) / (props.max - props.min)
    ctx.strokeStyle = z.color
    ctx.lineWidth = 4
    ctx.beginPath()
    ctx.arc(cx, cy, r - 1, START + t0 * SWEEP, START + t1 * SWEEP)
    ctx.stroke()
  }

  // Ticks
  ctx.strokeStyle = '#000000'
  for (let i = 0; i <= 10; i++) {
    const a = START + (i / 10) * SWEEP
    const major = i % 2 === 0
    const r0 = r - (major ? 12 : 6)
    ctx.lineWidth = major ? 1.5 : 1
    ctx.beginPath()
    ctx.moveTo(cx + Math.cos(a) * r0, cy + Math.sin(a) * r0)
    ctx.lineTo(cx + Math.cos(a) * (r - 4), cy + Math.sin(a) * (r - 4))
    ctx.stroke()
  }

  // Needle
  const a = START + ratio.value * SWEEP
  ctx.strokeStyle = '#000000'
  ctx.lineWidth = 2
  ctx.beginPath()
  ctx.moveTo(cx - Math.cos(a) * 6, cy - Math.sin(a) * 6)
  ctx.lineTo(cx + Math.cos(a) * (r - 8), cy + Math.sin(a) * (r - 8))
  ctx.stroke()

  // Hub
  ctx.fillStyle = '#000000'
  ctx.beginPath()
  ctx.arc(cx, cy, 3, 0, Math.PI * 2)
  ctx.fill()

  // Readout
  ctx.fillStyle = '#000000'
  ctx.font = 'bold 11px Tahoma, sans-serif'
  ctx.textAlign = 'center'
  ctx.fillText(formatValue(props.value, props.max - props.min) + props.unit, cx, cy + r * 0.55)
  if (props.label) {
    ctx.font = '10px Tahoma, sans-serif'
    ctx.fillText(props.label, cx, cy + r * 0.55 + 13)
  }
}

/** Choose a sensible number of decimals for the dial readout. */


onMounted(draw)
watch(() => [props.value, props.min, props.max, props.size, props.label, props.unit], draw)
watch(() => props.zones, draw, { deep: true })
</script>

<template>
  <div class="w-gauge">
    <canvas ref="canvas" class="w-gauge__canvas" />
  </div>
</template>

<style scoped>
.w-gauge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.w-gauge__canvas {
  display: block;
}
</style>
