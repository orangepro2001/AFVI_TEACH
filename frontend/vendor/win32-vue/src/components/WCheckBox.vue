<script setup lang="ts">
import { computed } from 'vue'
import { splitMnemonic } from '../utils/mnemonic'

const props = withDefaults(
  defineProps<{
    modelValue?: boolean | string | null
    label?: string
    triState?: boolean
    disabled?: boolean
    name?: string
  }>(),
  {
    modelValue: false,
    label: '',
    triState: false,
    disabled: false,
    name: '',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean | string | null): void
  (e: 'change', v: boolean | string | null): void
}>()

type Tri = false | true | 'indeterminate'

const state = computed<Tri>(() => {
  if (props.triState && props.modelValue === 'indeterminate') return 'indeterminate'
  return props.modelValue === true
})

function next(): Tri {
  if (!props.triState) return !state.value
  if (state.value === false) return true
  if (state.value === true) return 'indeterminate'
  return false
}

function onChange() {
  const v = next()
  emit('update:modelValue', v)
  emit('change', v)
}
</script>

<template>
  <label class="w-checkbox w-selectable" :class="{ 'is-disabled': disabled }">
    <input
      class="w-checkbox__input"
      type="checkbox"
      :name="name || undefined"
      :checked="state === true"
      :indeterminate="state === 'indeterminate'"
      :disabled="disabled"
      @change="onChange"
    />
    <span class="w-checkbox__box" aria-hidden="true">
      <span v-if="state === 'indeterminate'" class="w-checkbox__dash" />
      <span v-else-if="state === true" class="w-checkbox__check" />
    </span>
    <span class="w-checkbox__label">
      <template v-for="(c, i) in splitMnemonic(label)" :key="i">
        <u v-if="c.mnemonic" class="w-mnemonic">{{ c.ch }}</u>
        <template v-else>{{ c.ch }}</template>
      </template>
    </span>
  </label>
</template>

<style scoped>
.w-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: var(--w-row-h);
  cursor: default;
  position: relative;
}

.w-checkbox.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
  cursor: default;
}

.w-checkbox__input {
  position: absolute;
  opacity: 0;
  width: 13px;
  height: 13px;
  margin: 0;
  left: 0;
  top: 50%;
  margin-top: -6px;
  z-index: 1;
  cursor: default;
}

.w-checkbox__box {
  width: 13px;
  height: 13px;
  flex: 0 0 13px;
  background: var(--w-window);
  position: relative;
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-checkbox__input:focus-visible + .w-checkbox__box {
  outline: 1px dotted var(--w-text);
  outline-offset: 1px;
}

.w-checkbox__check {
  position: absolute;
  left: 3px;
  top: 1px;
  width: 6px;
  height: 10px;
  border-right: 2px solid var(--w-text);
  border-bottom: 2px solid var(--w-text);
  transform: rotate(40deg);
}

.w-checkbox__dash {
  position: absolute;
  left: 2px;
  top: 5px;
  width: 8px;
  height: 2px;
  background: var(--w-text);
}

.w-checkbox__label {
  white-space: nowrap;
}

.w-mnemonic {
  text-decoration: underline;
}
</style>
