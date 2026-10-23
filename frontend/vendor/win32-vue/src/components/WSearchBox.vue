<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    disabled?: boolean
    width?: string | number
  }>(),
  {
    modelValue: '',
    placeholder: 'Search',
    disabled: false,
    width: '180px',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
  (e: 'search', v: string): void
  (e: 'clear'): void
}>()

const hasText = computed(() => props.modelValue.length > 0)

function clear() {
  emit('update:modelValue', '')
  emit('clear')
  emit('search', '')
}
</script>

<template>
  <div
    class="w-searchbox"
    :class="{ 'is-disabled': disabled }"
    :style="{ width: typeof width === 'number' ? width + 'px' : width }"
  >
    <span class="w-searchbox__glass" aria-hidden="true" />
    <input
      class="w-searchbox__input"
      type="text"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      @keydown.enter="emit('search', modelValue)"
    />
    <button
      v-if="hasText"
      type="button"
      class="w-searchbox__clear"
      :disabled="disabled"
      tabindex="-1"
      aria-label="Clear"
      @click="clear"
    >
      ×
    </button>
  </div>
</template>

<style scoped>
.w-searchbox {
  display: inline-flex;
  align-items: center;
  height: var(--w-field-h);
  padding: 0 3px 0 4px;
  gap: 3px;
  background: var(--w-window);
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-searchbox__glass {
  flex: 0 0 12px;
  width: 12px;
  height: 12px;
  border: 1px solid var(--w-shadow);
  border-radius: 50%;
  position: relative;
}

.w-searchbox__glass::after {
  content: '';
  position: absolute;
  right: -3px;
  bottom: -2px;
  width: 5px;
  height: 1px;
  background: var(--w-shadow);
  transform: rotate(45deg);
}

.w-searchbox__input {
  flex: 1 1 auto;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  height: 100%;
}

.w-searchbox__clear {
  flex: 0 0 14px;
  width: 14px;
  height: 14px;
  line-height: 12px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--w-shadow);
  cursor: default;
  font-size: 13px;
}

.w-searchbox.is-disabled {
  background: var(--w-face);
}

.w-searchbox.is-disabled .w-searchbox__input {
  color: var(--w-disabled);
}
</style>
