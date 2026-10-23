<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = withDefaults(
  defineProps<{
    value?: number
    max?: number
    indeterminate?: boolean
    segments?: number
    width?: string | number
    height?: number
  }>(),
  {
    value: 0,
    max: 100,
    indeterminate: false,
    segments: 0,
    width: '100%',
    height: 14,
  },
)

const track = ref<HTMLElement | null>(null)
const trackWidth = ref(0)
let observer: ResizeObserver | undefined

const ratio = computed(() => {
  if (props.indeterminate) return 1
  if (props.max <= 0) return 0
  return Math.min(1, Math.max(0, props.value / props.max))
})

const segmentCount = computed(() => (props.segments > 0 ? props.segments : 10))

const filled = computed(() => {
  if (props.indeterminate) return 1
  if (props.segments > 0) {
    return Math.round(ratio.value * props.segments) / props.segments
  }
  return ratio.value
})

/** Pixel pitch of one classic progress block (Win32 uses 10 by default). */
const segPx = computed(() => {
  const n = segmentCount.value
  if (!trackWidth.value) return 10
  return Math.max(4, Math.floor(trackWidth.value / n))
})

onMounted(() => {
  if (typeof ResizeObserver === 'undefined') return
  observer = new ResizeObserver(() => {
    trackWidth.value = track.value?.clientWidth ?? 0
  })
  if (track.value) observer.observe(track.value)
})

onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <div
    ref="track"
    class="w-progress"
    :class="{ 'is-indeterminate': indeterminate }"
    :style="{
      width: typeof width === 'number' ? width + 'px' : width,
      height: `${height}px`,
    }"
    role="progressbar"
    :aria-valuemin="0"
    :aria-valuemax="max"
    :aria-valuenow="indeterminate ? undefined : value"
  >
    <div
      class="w-progress__fill"
      :style="{
        width: `${filled * 100}%`,
        backgroundImage: `repeating-linear-gradient(90deg, var(--w-progress) 0 ${segPx - 1}px, transparent ${segPx - 1}px ${segPx}px)`,
      }"
    />
  </div>
</template>

<style scoped>
.w-progress {
  position: relative;
  overflow: hidden;
  background: var(--w-face);
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-progress__fill {
  height: 100%;
  /* Face colour underneath so the 1px gaps read like the classic segmented bar. */
  background-color: var(--w-face);
  background-repeat: no-repeat;
}

.w-progress.is-indeterminate .w-progress__fill {
  width: 30% !important;
  animation: w-progress-slide 1.2s linear infinite;
}

@keyframes w-progress-slide {
  from {
    transform: translateX(-100%);
  }
  to {
    transform: translateX(333%);
  }
}
</style>
