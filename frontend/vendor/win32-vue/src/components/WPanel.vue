<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    variant?: 'face' | 'sunken' | 'raised' | 'window'
    padding?: number | string
    title?: string
  }>(),
  {
    variant: 'face',
    padding: 6,
    title: '',
  },
)

const pad = computed(() =>
  typeof props.padding === 'number' ? `${props.padding}px` : props.padding,
)
</script>

<template>
  <section
    class="w-panel"
    :class="`w-panel--${variant}`"
    :style="{ padding: pad }"
  >
    <header v-if="title" class="w-panel__title">{{ title }}</header>
    <div class="w-panel__body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.w-panel {
  background: var(--w-face);
  color: var(--w-text);
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.w-panel--sunken {
  background: var(--w-window);
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-panel--raised {
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-panel--window {
  background: var(--w-window);
}

.w-panel__title {
  font-weight: bold;
  margin-bottom: 4px;
  padding-bottom: 2px;
  border-bottom: 1px solid var(--w-shadow);
}

.w-panel__body {
  flex: 1 1 auto;
  min-height: 0;
}
</style>
