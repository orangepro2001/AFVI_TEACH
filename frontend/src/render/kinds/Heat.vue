<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import type { UiNode } from '../../types'

const props = defineProps<{ node: UiNode }>()
const cv = ref<HTMLCanvasElement>()

function paint() {
  const p = props.node.p ?? {}
  const c = cv.value
  if (!c) return
  const w: number = p.w, h: number = p.h
  const bin = atob(p.data as string)
  c.width = w
  c.height = h
  const ctx = c.getContext('2d')!
  const img = ctx.createImageData(w, h)
  for (let i = 0; i < w * h; i++) {
    const t = bin.charCodeAt(i) / 255
    let r = 0, g = 0, b = 0
    if (t < 0.5) b = 255 * t / 0.5
    else { const k = (t - 0.5) / 0.5; r = 255 * k; g = 255 * k; b = 255 * (1 - k) }
    img.data[i * 4] = r; img.data[i * 4 + 1] = g; img.data[i * 4 + 2] = b; img.data[i * 4 + 3] = 255
  }
  ctx.putImageData(img, 0, 0)
}
onMounted(paint)
watch(() => props.node, paint)
const size = () => (props.node.p?.size ?? 120) + 'px'
</script>

<template>
  <div class="n-heat w-bevel-sunken" :style="{ width: size(), height: size() }">
    <canvas ref="cv" style="width:100%;height:100%;image-rendering:pixelated" />
    <span
      class="n-heat-peak"
      :style="{ left: ((node.p!.peak % node.p!.w + 0.5) / node.p!.w * 100) + '%', top: ((Math.floor(node.p!.peak / node.p!.w) + 0.5) / node.p!.h * 100) + '%' }"
    />
  </div>
</template>

<style scoped>
.n-heat { position: relative; margin: 3px 0; }
.n-heat-peak { position: absolute; width: 10px; height: 10px; margin: -5px 0 0 -5px; border: 2px solid #0f0; border-radius: 50%; box-sizing: border-box; }
</style>
