<script setup lang="ts">
import { computed } from 'vue'
import type { WTrendSeries } from '../types'

const props = withDefaults(
  defineProps<{
    series?: WTrendSeries[]
    min?: number
    max?: number
    height?: number
    width?: number
    grid?: boolean
    showLegend?: boolean
    fill?: boolean
  }>(),
  {
    series: () => [],
    min: 0,
    max: 100,
    height: 140,
    width: 320,
    grid: true,
    showLegend: true,
    fill: false,
  },
)

const span = computed(() => props.max - props.min || 1)

function xOf(i: number, n: number) {
  return n === 1 ? 0 : (props.width / (n - 1)) * i
}

function yOf(p: number) {
  return props.height - ((p - props.min) / span.value) * props.height
}

function linePoints(s: WTrendSeries) {
  const n = s.points.length
  return s.points.map((p, i) => `${xOf(i, n).toFixed(1)},${yOf(p).toFixed(1)}`).join(' ')
}

function areaPoints(s: WTrendSeries) {
  const n = s.points.length
  const pts = s.points.map((p, i) => `${xOf(i, n).toFixed(1)},${yOf(p).toFixed(1)}`).join(' ')
  return `0,${props.height} ${pts} ${props.width},${props.height}`
}
</script>

<template>
  <div class="w-trend">
    <div class="w-trend__canvas-wrap">
      <svg
        class="w-trend__svg"
        :viewBox="`0 0 ${width} ${height}`"
        :width="width"
        :height="height"
        preserveAspectRatio="none"
      >
        <rect x="0" y="0" :width="width" :height="height" fill="#ffffff" />

        <g v-if="grid" class="w-trend__grid">
          <line
            v-for="i in 5"
            :key="`h${i}`"
            x1="0"
            :y1="(height / 5) * (i - 1)"
            :x2="width"
            :y2="(height / 5) * (i - 1)"
          />
          <line
            v-for="i in 6"
            :key="`v${i}`"
            :x1="(width / 6) * (i - 1)"
            y1="0"
            :x2="(width / 6) * (i - 1)"
            :y2="height"
          />
        </g>

        <g v-for="(s, si) in series" :key="si">
          <template v-if="s.points.length">
            <polygon
              v-if="fill"
              :points="areaPoints(s)"
              :fill="s.color"
              fill-opacity="0.15"
            />
            <polyline
              :points="linePoints(s)"
              fill="none"
              :stroke="s.color"
              stroke-width="1.5"
            />
          </template>
        </g>

        <rect
          x="0"
          y="0"
          :width="width"
          :height="height"
          fill="none"
          stroke="#808080"
          stroke-width="1"
        />
      </svg>
    </div>
    <div v-if="showLegend" class="w-trend__legend">
      <span v-for="(s, i) in series" :key="i" class="w-trend__legend-item">
        <i class="w-trend__swatch" :style="{ background: s.color }" />
        {{ s.name }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.w-trend {
  display: inline-flex;
  flex-direction: column;
  gap: 3px;
}

.w-trend__canvas-wrap {
  background: var(--w-window);
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-trend__svg {
  display: block;
}

.w-trend__grid line {
  stroke: #c8c4bc;
  stroke-width: 1;
  shape-rendering: crispEdges;
}

.w-trend__legend {
  display: flex;
  gap: 10px;
  font-size: 10px;
}

.w-trend__legend-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.w-trend__swatch {
  display: inline-block;
  width: 9px;
  height: 3px;
}
</style>
