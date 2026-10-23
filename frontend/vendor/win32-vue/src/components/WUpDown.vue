<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: number
    min?: number
    max?: number
    step?: number
    disabled?: boolean
    width?: string | number
  }>(),
  {
    modelValue: 0,
    min: 0,
    max: 100,
    step: 1,
    disabled: false,
    width: '52px',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: number): void
  (e: 'change', v: number): void
}>()

const value = computed(() => props.modelValue)

function clamp(v: number) {
  return Math.min(props.max, Math.max(props.min, v))
}

function bump(delta: number) {
  if (props.disabled) return
  const next = clamp(Number((value.value + delta).toFixed(6)))
  if (next === value.value) return
  emit('update:modelValue', next)
  emit('change', next)
}
</script>

<template>
  <div
    class="w-updown"
    :class="{ 'is-disabled': disabled }"
    :style="{ width: typeof width === 'number' ? width + 'px' : width }"
  >
    <div class="w-updown__field">
      <span class="w-updown__text w-mono">{{ value }}</span>
    </div>
    <div class="w-updown__buttons">
      <button
        type="button"
        class="w-updown__btn w-updown__btn--up"
        :disabled="disabled"
        tabindex="-1"
        aria-label="Increase"
        @click="bump(step)"
      >
        <span class="w-updown__glyph w-updown__glyph--up" />
      </button>
      <button
        type="button"
        class="w-updown__btn w-updown__btn--down"
        :disabled="disabled"
        tabindex="-1"
        aria-label="Decrease"
        @click="bump(-step)"
      >
        <span class="w-updown__glyph w-updown__glyph--down" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.w-updown {
  display: inline-flex;
  height: var(--w-field-h);
  background: var(--w-window);
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-updown__field {
  flex: 1 1 auto;
  min-width: 0;
  padding: 0 3px;
  overflow: hidden;
}

.w-updown__text {
  display: block;
  line-height: calc(var(--w-field-h) - 2px);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.w-updown__buttons {
  flex: 0 0 16px;
  display: flex;
  flex-direction: column;
}

.w-updown__btn {
  flex: 1 1 0;
  background: var(--w-face);
  border: none;
  padding: 0;
  cursor: default;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-updown__btn:active:not(:disabled) {
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-updown__glyph {
  width: 0;
  height: 0;
  border-left: 3px solid transparent;
  border-right: 3px solid transparent;
}

.w-updown__glyph--up {
  border-bottom: 4px solid var(--w-text);
}

.w-updown__glyph--down {
  border-top: 4px solid var(--w-text);
}

.w-updown.is-disabled {
  background: var(--w-face);
  color: var(--w-disabled);
}
</style>
