<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: number
    min?: number
    max?: number
    label?: string
    unit?: string
    size?: number
    startAngle?: number
    endAngle?: number
    disabled?: boolean
  }>(),
  {
    modelValue: 0,
    min: 0,
    max: 100,
    label: '',
    unit: '',
    size: 64,
    startAngle: -135,
    endAngle: 135,
    disabled: false,
  },
)

const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()

const canvas = ref<HTMLCanvasElement | null>(null)
const dragging = ref(false)

const ratio = computed(() => {
  const span = props.max - props.min
  if (span <= 0) return 0
  return Math.min(1, Math.max(0, (props.modelValue - props.min) / span))
})

function angleAt(ratio: number) {
  return props.startAngle + (props.endAngle - props.startAngle) * ratio
}

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
  const r = s / 2 - 3

  // Dial plate
  ctx.fillStyle = '#d4d0c8'
  ctx.beginPath()
  ctx.arc(cx, cy, r, 0, Math.PI * 2)
  ctx.fill()
  ctx.strokeStyle = '#808080'
  ctx.lineWidth = 2
  ctx.stroke()
  ctx.strokeStyle = '#ffffff'
  ctx.lineWidth = 1
  ctx.beginPath()
  ctx.arc(cx, cy, r - 1, 0, Math.PI * 2)
  ctx.stroke()

  // Tick ring
  for (let i = 0; i <= 20; i++) {
    const a = angleAt(i / 20)
    const major = i % 5 === 0
    const r0 = r - (major ? 9 : 5)
    ctx.strokeStyle = '#000000'
    ctx.lineWidth = major ? 1.5 : 1
    ctx.beginPath()
    ctx.moveTo(cx + Math.cos((a * Math.PI) / 180) * r0, cy + Math.sin((a * Math.PI) / 180) * r0)
    ctx.lineTo(cx + Math.cos((a * Math.PI) / 180) * (r - 3), cy + Math.sin((a * Math.PI) / 180) * (r - 3))
    ctx.stroke()
  }

  // Pointer
  const a = (angleAt(ratio.value) * Math.PI) / 180
  ctx.strokeStyle = '#000000'
  ctx.lineWidth = 2
  ctx.beginPath()
  ctx.moveTo(cx + Math.cos(a) * 3, cy + Math.sin(a) * 3)
  ctx.lineTo(cx + Math.cos(a) * (r - 10), cy + Math.sin(a) * (r - 10))
  ctx.stroke()

  // Knurled grip ring
  ctx.strokeStyle = '#808080'
  ctx.lineWidth = 1
  for (let i = 0; i < 24; i++) {
    const ga = (i / 24) * Math.PI * 2
    ctx.beginPath()
    ctx.moveTo(cx + Math.cos(ga) * (r - 3), cy + Math.sin(ga) * (r - 3))
    ctx.lineTo(cx + Math.cos(ga) * r, cy + Math.sin(ga) * r)
    ctx.stroke()
  }
}

function valueFromPointer(ev: PointerEvent) {
  const el = canvas.value
  if (!el) return props.modelValue
  const rect = el.getBoundingClientRect()
  const dx = ev.clientX - (rect.left + rect.width / 2)
  const dy = ev.clientY - (rect.top + rect.height / 2)
  let deg = (Math.atan2(dy, dx) * 180) / Math.PI + 90
  if (deg < props.startAngle) deg = props.startAngle
  if (deg > props.endAngle) deg = props.endAngle
  const t = (deg - props.startAngle) / (props.endAngle - props.startAngle)
  const v = props.min + t * (props.max - props.min)
  return Math.round(v * 100) / 100
}

function onPointerDown(ev: PointerEvent) {
  if (props.disabled) return
  dragging.value = true
  ;(ev.currentTarget as HTMLElement).setPointerCapture?.(ev.pointerId)
  emit('update:modelValue', valueFromPointer(ev))
}

function onPointerMove(ev: PointerEvent) {
  if (!dragging.value || props.disabled) return
  emit('update:modelValue', valueFromPointer(ev))
}

function onPointerUp(ev: PointerEvent) {
  if (!dragging.value) return
  dragging.value = false
  ;(ev.currentTarget as HTMLElement).releasePointerCapture?.(ev.pointerId)
}

function onKeydown(ev: KeyboardEvent) {
  if (props.disabled) return
  const span = props.max - props.min
  let next: number | null = null
  if (ev.key === 'ArrowUp' || ev.key === 'ArrowRight') next = props.modelValue + span / 50
  if (ev.key === 'ArrowDown' || ev.key === 'ArrowLeft') next = props.modelValue - span / 50
  if (next !== null) {
    ev.preventDefault()
    emit('update:modelValue', Math.min(props.max, Math.max(props.min, next)))
  }
}

onMounted(draw)
watch(() => [props.modelValue, props.min, props.max, props.size], draw)
</script>

<template>
  <div class="w-knob">
    <canvas
      ref="canvas"
      class="w-knob__canvas"
      tabindex="0"
      role="slider"
      :aria-valuemin="min"
      :aria-valuemax="max"
      :aria-valuenow="modelValue"
      :aria-label="label || undefined"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
      @keydown="onKeydown"
    />
    <div class="w-knob__caption">
      <span class="w-knob__value w-mono">{{ modelValue }}{{ unit }}</span>
      <span v-if="label" class="w-knob__label">{{ label }}</span>
    </div>
  </div>
</template>

<style scoped>
.w-knob {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
}

.w-knob__canvas {
  display: block;
  cursor: default;
}

.w-knob__caption {
  display: flex;
  flex-direction: column;
  align-items: center;
  line-height: 1.25;
}

.w-knob__value {
  font-size: 11px;
}

.w-knob__label {
  font-size: 10px;
  color: var(--w-disabled);
}
</style>
