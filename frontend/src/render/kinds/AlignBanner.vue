<script setup lang="ts">
import type { UiNode } from '../../types'

defineProps<{ node: UiNode }>()
</script>

<template>
  <div class="n-abanner w-bevel-sunken" :class="'lv-' + (node.p?.level ?? 'ok')">{{ node.l }}</div>
</template>

<style scoped>
.n-abanner {
  margin: 4px 0;
  padding: 6px 8px;
  text-align: center;
  font-weight: bold;
  font-size: 1.5em;
}
.lv-ok { background: #00c000; color: #000; }
.lv-manual { background: #ffd800; color: #000; }
.lv-ng { background: #ff0000; color: #fff; }
</style>
