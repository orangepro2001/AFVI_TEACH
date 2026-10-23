<script setup lang="ts">
import { computed } from 'vue'
import { splitMnemonic } from '../utils/mnemonic'

const props = withDefaults(
  defineProps<{
    modelValue?: string | number | null
    value: string | number
    label?: string
    name?: string
    disabled?: boolean
  }>(),
  {
    modelValue: null,
    label: '',
    name: '',
    disabled: false,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string | number): void
  (e: 'change', v: string | number): void
}>()

const checked = computed(() => props.modelValue === props.value)

function onChange() {
  emit('update:modelValue', props.value)
  emit('change', props.value)
}
</script>

<template>
  <label class="w-radio w-selectable" :class="{ 'is-disabled': disabled }">
    <input
      class="w-radio__input"
      type="radio"
      :name="name || undefined"
      :value="value"
      :checked="checked"
      :disabled="disabled"
      @change="onChange"
    />
    <span class="w-radio__dot" aria-hidden="true">
      <span v-if="checked" class="w-radio__fill" />
    </span>
    <span class="w-radio__label">
      <template v-for="(c, i) in splitMnemonic(label)" :key="i">
        <u v-if="c.mnemonic" class="w-mnemonic">{{ c.ch }}</u>
        <template v-else>{{ c.ch }}</template>
      </template>
    </span>
  </label>
</template>

<style scoped>
.w-radio {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: var(--w-row-h);
  cursor: default;
  position: relative;
}

.w-radio.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-radio__input {
  position: absolute;
  opacity: 0;
  width: 12px;
  height: 12px;
  margin: 0;
  left: 0;
  top: 50%;
  margin-top: -6px;
  z-index: 1;
}

.w-radio__dot {
  position: relative;
  width: 12px;
  height: 12px;
  flex: 0 0 12px;
  border-radius: 50%;
  background: var(--w-shadow);
}

/* White inner disc — the classic Win32 radio ring. */
.w-radio__dot::before {
  content: '';
  position: absolute;
  inset: 1px;
  border-radius: 50%;
  background: var(--w-window);
}

.w-radio__input:focus-visible + .w-radio__dot {
  outline: 1px dotted var(--w-text);
  outline-offset: 1px;
}

.w-radio__fill {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 4px;
  height: 4px;
  margin: -2px 0 0 -2px;
  border-radius: 50%;
  background: var(--w-text);
  z-index: 1;
}

.w-radio__label {
  white-space: nowrap;
}

.w-mnemonic {
  text-decoration: underline;
}
</style>
