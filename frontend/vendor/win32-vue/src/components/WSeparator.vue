<script setup lang="ts">
withDefaults(
  defineProps<{
    orientation?: 'horizontal' | 'vertical'
    inset?: boolean
    label?: string
  }>(),
  {
    orientation: 'horizontal',
    inset: false,
    label: '',
  },
)
</script>

<template>
  <div
    class="w-separator"
    :class="[`w-separator--${orientation}`, { 'is-inset': inset }]"
    role="separator"
    :aria-orientation="orientation"
  >
    <span v-if="label" class="w-separator__label">{{ label }}</span>
  </div>
</template>

<style scoped>
.w-separator--horizontal {
  height: 2px;
  border-top: 1px solid var(--w-shadow);
  border-bottom: 1px solid var(--w-hilight);
  margin: 3px 0;
}

.w-separator--vertical {
  width: 2px;
  align-self: stretch;
  border-left: 1px solid var(--w-shadow);
  border-right: 1px solid var(--w-hilight);
  margin: 0 3px;
}

.w-separator--horizontal.is-inset {
  margin-left: 6px;
  margin-right: 6px;
}

.w-separator__label {
  position: absolute;
  margin-left: 6px;
  background: var(--w-face);
  padding: 0 3px;
  line-height: 1;
  top: -5px;
}
</style>
