<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    orientation?: 'horizontal' | 'vertical'
    size?: number
    modelValue?: number
    min?: number
    max?: number
    disabled?: boolean
  }>(),
  {
    orientation: 'vertical',
    size: 4,
    modelValue: 50,
    min: 0,
    max: 100,
    disabled: false,
  },
)

const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()

const pct = computed(() => {
  const span = props.max - props.min
  if (span <= 0) return 50
  return ((props.modelValue - props.min) / span) * 100
})

function onPointerDown(ev: PointerEvent) {
  if (props.disabled) return
  const el = ev.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  const t =
    props.orientation === 'horizontal'
      ? (ev.clientX - rect.left) / Math.max(1, rect.width)
      : (ev.clientY - rect.top) / Math.max(1, rect.height)
  const v = Math.round((props.min + Math.min(1, Math.max(0, t)) * (props.max - props.min)) * 100) / 100
  emit('update:modelValue', v)
}
</script>

<template>
  <div
    class="w-splitter"
    :class="[`w-splitter--${orientation}`, { 'is-disabled': disabled }]"
    role="separator"
    :aria-orientation="orientation"
    :aria-valuenow="modelValue"
    tabindex="0"
    @pointerdown="onPointerDown"
  >
    <span class="w-splitter__grip" aria-hidden="true"><i /><i /><i /></span>
  </div>
</template>

<style scoped>
.w-splitter {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--w-face);
  cursor: default;
  flex: 0 0 auto;
}

.w-splitter--vertical {
  width: 4px;
  cursor: col-resize;
}

.w-splitter--horizontal {
  height: 4px;
  cursor: row-resize;
}

.w-splitter__grip {
  display: flex;
  gap: 2px;
  flex-direction: column;
  align-items: center;
}

.w-splitter--horizontal .w-splitter__grip {
  flex-direction: row;
}

.w-splitter__grip i {
  display: block;
  width: 2px;
  height: 14px;
  background: var(--w-hilight);
  box-shadow: 1px 0 0 var(--w-shadow);
}

.w-splitter--horizontal .w-splitter__grip i {
  width: 14px;
  height: 2px;
  box-shadow: 0 1px 0 var(--w-shadow);
}

.w-splitter.is-disabled {
  cursor: default;
}
</style>
