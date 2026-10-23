<script setup lang="ts">
withDefaults(
  defineProps<{
    width?: number | string
    align?: 'left' | 'center' | 'right'
  }>(),
  {
    width: 'auto',
    align: 'left',
  },
)
</script>

<template>
  <div
    class="w-statuspane"
    :class="`is-${align}`"
    :style="{
      width: typeof width === 'number' ? width + 'px' : width,
      flex: width === 'auto' ? '1 1 auto' : '0 0 auto',
    }"
  >
    <slot />
  </div>
</template>

<style scoped>
.w-statuspane {
  padding: 0 6px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  display: flex;
  align-items: center;
  border-right: 1px solid var(--w-shadow);
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-shadow);
}

.w-statuspane.is-center {
  justify-content: center;
}

.w-statuspane.is-right {
  justify-content: flex-end;
}
</style>
