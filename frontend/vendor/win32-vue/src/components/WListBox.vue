<script setup lang="ts">
import { computed } from 'vue'
import type { WComboOption } from '../types'

const props = withDefaults(
  defineProps<{
    modelValue?: string | number | Array<string | number> | null
    options?: WComboOption[]
    multiple?: boolean
    height?: number | string
    disabled?: boolean
    width?: string | number
  }>(),
  {
    modelValue: null,
    options: () => [],
    multiple: false,
    height: '120px',
    disabled: false,
    width: '100%',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string | number | Array<string | number>): void
  (e: 'change', v: string | number | Array<string | number>): void
}>()

const selected = computed<Array<string | number>>(() => {
  if (props.multiple) return Array.isArray(props.modelValue) ? props.modelValue : []
  const v = props.modelValue
  if (v === null || v === undefined || Array.isArray(v)) return []
  return [v]
})

function isSelected(v: string | number) {
  return selected.value.includes(v)
}

function toggle(o: WComboOption) {
  if (props.disabled || o.disabled) return
  let next: Array<string | number>
  if (props.multiple) {
    next = isSelected(o.value)
      ? selected.value.filter((v) => v !== o.value)
      : [...selected.value, o.value]
  } else {
    next = [o.value]
  }
  emit('update:modelValue', props.multiple ? next : next[0])
  emit('change', props.multiple ? next : next[0])
}
</script>

<template>
  <ul
    class="w-listbox"
    :class="{ 'is-disabled': disabled }"
    :style="{
      height: typeof height === 'number' ? height + 'px' : height,
      width: typeof width === 'number' ? width + 'px' : width,
    }"
    role="listbox"
    :aria-multiselectable="multiple || undefined"
  >
    <li
      v-for="(o, i) in options"
      :key="i"
      class="w-listbox__item"
      :class="{
        'is-selected': isSelected(o.value),
        'is-disabled': o.disabled,
      }"
      role="option"
      :aria-selected="isSelected(o.value)"
      @click="toggle(o)"
    >
      {{ o.label }}
    </li>
    <li v-if="options.length === 0" class="w-listbox__item is-empty">(empty)</li>
  </ul>
</template>

<style scoped>
.w-listbox {
  margin: 0;
  padding: 1px;
  list-style: none;
  background: var(--w-window);
  overflow: auto;
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-listbox__item {
  height: var(--w-row-h);
  line-height: calc(var(--w-row-h) - 2px);
  padding: 0 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: default;
}

.w-listbox__item.is-selected {
  background: var(--w-select-bg);
  color: var(--w-select-text);
}

.w-listbox__item.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-listbox__item.is-empty {
  color: var(--w-disabled);
}

.w-listbox.is-disabled {
  background: var(--w-face);
}
</style>
