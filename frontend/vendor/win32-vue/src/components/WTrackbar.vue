<script setup lang="ts">
import { computed, ref } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: number
    min?: number
    max?: number
    step?: number
    orientation?: 'horizontal' | 'vertical'
    ticks?: boolean
    disabled?: boolean
    label?: string
  }>(),
  {
    modelValue: 0,
    min: 0,
    max: 100,
    step: 1,
    orientation: 'horizontal',
    ticks: true,
    disabled: false,
    label: '',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: number): void
  (e: 'change', v: number): void
}>()

const track = ref<HTMLElement | null>(null)
const dragging = ref(false)

const ratio = computed(() => {
  const span = props.max - props.min
  if (span <= 0) return 0
  return Math.min(1, Math.max(0, (props.modelValue - props.min) / span))
})

const percent = computed(() => `${ratio.value * 100}%`)

const tickCount = computed(() => {
  // Classic trackbar: a handful of evenly spaced major ticks only.
  const span = props.max - props.min
  if (span <= 0) return 2
  const rough = Math.round(span / 10)
  return Math.min(11, Math.max(2, rough + 1))
})

function valueFromClientX(clientX: number) {
  const el = track.value
  if (!el) return props.modelValue
  const rect = el.getBoundingClientRect()
  const t = rect.width > 0 ? (clientX - rect.left) / rect.width : 0
  const raw = props.min + Math.min(1, Math.max(0, t)) * (props.max - props.min)
  return snap(raw)
}

function valueFromClientY(clientY: number) {
  const el = track.value
  if (!el) return props.modelValue
  const rect = el.getBoundingClientRect()
  const t = rect.height > 0 ? (rect.bottom - clientY) / rect.height : 0
  const raw = props.min + Math.min(1, Math.max(0, t)) * (props.max - props.min)
  return snap(raw)
}

function snap(raw: number) {
  const step = props.step > 0 ? props.step : 1
  const stepped = Math.round((raw - props.min) / step) * step + props.min
  const v = Math.min(props.max, Math.max(props.min, Number(stepped.toFixed(6))))
  return v
}

function onPointerDown(ev: PointerEvent) {
  if (props.disabled) return
  dragging.value = true
  ;(ev.currentTarget as HTMLElement).setPointerCapture?.(ev.pointerId)
  const v =
    props.orientation === 'horizontal' ? valueFromClientX(ev.clientX) : valueFromClientY(ev.clientY)
  emit('update:modelValue', v)
}

function onPointerMove(ev: PointerEvent) {
  if (!dragging.value || props.disabled) return
  const v =
    props.orientation === 'horizontal' ? valueFromClientX(ev.clientX) : valueFromClientY(ev.clientY)
  if (v !== props.modelValue) emit('update:modelValue', v)
}

function onPointerUp(ev: PointerEvent) {
  if (!dragging.value) return
  dragging.value = false
  ;(ev.currentTarget as HTMLElement).releasePointerCapture?.(ev.pointerId)
  emit('change', props.modelValue)
}

function onKeydown(ev: KeyboardEvent) {
  if (props.disabled) return
  let next: number | null = null
  const big = (props.max - props.min) / 10
  if (ev.key === 'ArrowLeft' || ev.key === 'ArrowDown') next = snap(props.modelValue - props.step)
  if (ev.key === 'ArrowRight' || ev.key === 'ArrowUp') next = snap(props.modelValue + props.step)
  if (ev.key === 'PageDown') next = snap(props.modelValue - big)
  if (ev.key === 'PageUp') next = snap(props.modelValue + big)
  if (ev.key === 'Home') next = props.min
  if (ev.key === 'End') next = props.max
  if (next !== null) {
    ev.preventDefault()
    emit('update:modelValue', next)
    emit('change', next)
  }
}
</script>

<template>
  <div
    class="w-trackbar"
    :class="[`w-trackbar--${orientation}`, { 'is-disabled': disabled }]"
  >
    <div
      ref="track"
      class="w-trackbar__track"
      role="slider"
      tabindex="0"
      :aria-valuemin="min"
      :aria-valuemax="max"
      :aria-valuenow="modelValue"
      :aria-label="label || undefined"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
      @keydown="onKeydown"
    >
      <div class="w-trackbar__thumb" :style="{ [orientation === 'horizontal' ? 'left' : 'bottom']: percent }" />
    </div>
    <div v-if="ticks" class="w-trackbar__ticks">
      <span
        v-for="i in tickCount"
        :key="i"
        class="w-trackbar__tick"
        :style="{ [orientation === 'horizontal' ? 'left' : 'bottom']: `${((i - 1) / Math.max(1, tickCount - 1)) * 100}%` }"
      />
    </div>
  </div>
</template>

<style scoped>
.w-trackbar {
  position: relative;
  padding: 6px 0;
}

.w-trackbar--horizontal {
  height: 24px;
  min-width: 120px;
}

.w-trackbar--vertical {
  width: 24px;
  height: 120px;
  padding: 0 6px;
}

.w-trackbar__track {
  position: relative;
  background: var(--w-face);
  height: 4px;
  cursor: default;
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-trackbar--vertical .w-trackbar__track {
  width: 4px;
  height: 100%;
}

.w-trackbar__thumb {
  position: absolute;
  top: 50%;
  width: 11px;
  height: 18px;
  margin-top: -9px;
  margin-left: -5px;
  background: var(--w-face);
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-trackbar--vertical .w-trackbar__thumb {
  margin-top: 0;
  margin-left: 0;
  left: 50%;
  transform: translateX(-50%);
}

.w-trackbar__track:focus-visible {
  outline: 1px dotted var(--w-text);
  outline-offset: 2px;
}

.w-trackbar__ticks {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.w-trackbar__tick {
  position: absolute;
  width: 1px;
  height: 3px;
  background: var(--w-shadow);
}

.w-trackbar--horizontal .w-trackbar__tick {
  top: 12px;
}

.w-trackbar--vertical .w-trackbar__tick {
  left: 12px;
  width: 3px;
  height: 1px;
}

.w-trackbar.is-disabled {
  color: var(--w-disabled);
}

.w-trackbar.is-disabled .w-trackbar__track,
.w-trackbar.is-disabled .w-trackbar__thumb {
  background: var(--w-face-alt);
}
</style>
