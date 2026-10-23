<script setup lang="ts">
import type { UiNode } from '../types'
import { send } from '../bridge'

defineProps<{ node: UiNode }>()
</script>

<template>
  <div class="pal w-bevel-window">
    <template v-for="(it, i) in node.p?.items" :key="it.id">
      <div v-if="it.sep" class="pal-sep" />
      <button
        class="pal-btn w-bevel-raised"
        :class="{ active: it.active, off: !it.enabled }"
        :data-id="it.id"
        :title="it.tip"
        @click="it.enabled && send({ id: it.id, op: 'click' })"
      >
        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.6">
          <path v-if="it.icon === 'pick'" d="M6 3l12 8-6 1.5L9 19z" fill="currentColor" />
          <rect v-else-if="it.icon === 'rect'" x="4" y="6" width="16" height="12" stroke-dasharray="3 2" />
          <ellipse v-else-if="it.icon === 'ellipse'" cx="12" cy="12" rx="8" ry="6" stroke-dasharray="3 2" />
          <path v-else-if="it.icon === 'polygon'" d="M5 17l3-11 9 3 2 8z" stroke-dasharray="3 2" />
          <g v-else-if="it.icon === 'focus'"><path d="M4 9V4h5M15 4h5v5M20 15v5h-5M9 20H4v-5" /></g>
        </svg>
      </button>
    </template>
  </div>
</template>

<style scoped>
.pal { position: absolute; top: 8px; right: 8px; padding: 4px; background: var(--w-face); display: flex; flex-direction: column; gap: 2px; z-index: 5; }
.pal-btn { width: 34px; height: 34px; padding: 0; border: 0; background: var(--w-face); color: var(--w-text); display: flex; align-items: center; justify-content: center; cursor: default; }
.pal-btn.active { background: var(--w-light); box-shadow: inset 1px 1px 0 var(--w-dkshadow), inset -1px -1px 0 var(--w-hilight), inset 2px 2px 0 var(--w-shadow); }
.pal-btn.off { color: var(--w-disabled); }
.pal-sep { height: 6px; border-top: 1px solid var(--w-shadow); border-bottom: 1px solid var(--w-hilight); margin: 2px 0; }
</style>
