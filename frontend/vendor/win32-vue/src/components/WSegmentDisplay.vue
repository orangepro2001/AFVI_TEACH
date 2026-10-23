<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    value?: string | number
    digits?: number
    pad?: boolean
    color?: 'red' | 'green' | 'amber' | 'blue'
    size?: number
    showSign?: boolean
  }>(),
  {
    value: '',
    digits: 4,
    pad: true,
    color: 'red',
    size: 16,
    showSign: false,
  },
)

const SEGMENTS: Record<string, string[]> = {
  '0': ['a', 'b', 'c', 'd', 'e', 'f'],
  '1': ['b', 'c'],
  '2': ['a', 'b', 'g', 'e', 'd'],
  '3': ['a', 'b', 'g', 'c', 'd'],
  '4': ['f', 'g', 'b', 'c'],
  '5': ['a', 'f', 'g', 'c', 'd'],
  '6': ['a', 'f', 'g', 'e', 'c', 'd'],
  '7': ['a', 'b', 'c'],
  '8': ['a', 'b', 'c', 'd', 'e', 'f', 'g'],
  '9': ['a', 'b', 'c', 'd', 'f', 'g'],
  '-': ['g'],
  ' ': [],
}

const ALL = ['a', 'b', 'c', 'd', 'e', 'f', 'g']

const RECT: Record<string, Record<string, number>> = {
  a: { x: 1, y: 0, width: 8, height: 2 },
  b: { x: 8, y: 1, width: 2, height: 7 },
  c: { x: 8, y: 10, width: 2, height: 7 },
  d: { x: 1, y: 16, width: 8, height: 2 },
  e: { x: 0, y: 10, width: 2, height: 7 },
  f: { x: 0, y: 1, width: 2, height: 7 },
  g: { x: 1, y: 8, width: 8, height: 2 },
}

const text = computed(() => {
  let t = String(props.value ?? '')
  if (props.showSign && !/^-/.test(t)) t = `-${t}`
  if (props.pad) {
    const neg = t.startsWith('-')
    const body = neg ? t.slice(1) : t
    t = (neg ? '-' : '') + body.padStart(props.digits, ' ')
  }
  return t.split('')
})

function isOn(ch: string, seg: string) {
  const set = SEGMENTS[ch]
  if (!set) return false
  return set.includes(seg)
}

function segRect(seg: string) {
  return RECT[seg] ?? {}
}
</script>

<template>
  <div class="w-segdisplay" :class="`is-${color}`" :style="{ '--seg-size': `${size}px` }">
    <div v-for="(ch, i) in text" :key="i" class="w-segdisplay__digit">
      <svg
        class="w-segdisplay__svg"
        viewBox="0 0 10 18"
        preserveAspectRatio="none"
        shape-rendering="crispEdges"
      >
        <g class="w-segdisplay__off">
          <rect x="1" y="0" width="8" height="2" />
          <rect x="0" y="1" width="2" height="7" />
          <rect x="8" y="1" width="2" height="7" />
          <rect x="1" y="8" width="8" height="2" />
          <rect x="0" y="10" width="2" height="7" />
          <rect x="8" y="10" width="2" height="7" />
          <rect x="1" y="16" width="8" height="2" />
        </g>
        <g class="w-segdisplay__on">
          <template v-for="seg in ALL" :key="seg">
            <rect v-if="isOn(ch, seg)" v-bind="segRect(seg)" />
          </template>
        </g>
      </svg>
    </div>
  </div>
</template>

<style scoped>
.w-segdisplay {
  display: inline-flex;
  gap: 3px;
  padding: 2px 3px;
  background: #101010;
  border: 1px solid var(--w-dkshadow);
}

.w-segdisplay__digit {
  width: var(--seg-size);
  height: calc(var(--seg-size) * 1.8);
}

.w-segdisplay__svg {
  width: 100%;
  height: 100%;
  display: block;
}

.w-segdisplay__off rect {
  fill: #2a0000;
}

.w-segdisplay__on rect {
  fill: currentColor;
}

.w-segdisplay.is-red {
  color: #ff3b30;
}
.w-segdisplay.is-green {
  color: #33d17a;
}
.w-segdisplay.is-amber {
  color: #ffb300;
}
.w-segdisplay.is-blue {
  color: #4da6ff;
}
</style>
