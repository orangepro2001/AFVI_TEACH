<script setup lang="ts">
import { computed, ref, useAttrs } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: string | number
    type?: 'text' | 'password' | 'number'
    readonly?: boolean
    disabled?: boolean
    placeholder?: string
    maxlength?: number
    align?: 'left' | 'center' | 'right'
    error?: boolean
    invalid?: boolean
    width?: string | number
  }>(),
  {
    modelValue: '',
    type: 'text',
    readonly: false,
    disabled: false,
    placeholder: '',
    maxlength: undefined,
    align: 'left',
    error: false,
    invalid: false,
    width: '100%',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
  (e: 'enter'): void
  (e: 'blur', ev: FocusEvent): void
  (e: 'focus', ev: FocusEvent): void
}>()

const attrs = useAttrs()
const inner = ref<HTMLInputElement | null>(null)

const isError = computed(() => props.error || props.invalid)

function onInput(ev: Event) {
  const el = ev.target as HTMLInputElement
  emit('update:modelValue', el.value)
}

function onKeydown(ev: KeyboardEvent) {
  if (ev.key === 'Enter') {
    emit('enter')
  }
}

function focus() {
  inner.value?.focus()
}

defineExpose({ focus, el: inner })
</script>

<template>
  <input
    ref="inner"
    v-bind="attrs"
    class="w-edit"
    :class="{ 'is-disabled': disabled, 'is-readonly': readonly, 'is-error': isError }"
    :type="type === 'number' ? 'text' : type"
    :value="modelValue"
    :placeholder="placeholder || undefined"
    :maxlength="maxlength"
    :readonly="readonly"
    :disabled="disabled"
    :style="{ width: typeof width === 'number' ? width + 'px' : width, textAlign: align }"
    @input="onInput"
    @keydown="onKeydown"
    @blur="emit('blur', $event)"
    @focus="emit('focus', $event)"
  />
</template>

<style scoped>
.w-edit {
  height: var(--w-field-h);
  padding: 0 3px;
  background: var(--w-window);
  color: var(--w-window-text);
  border: none;
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-edit:focus {
  outline: none;
}

.w-edit:focus-visible {
  outline: 1px dotted var(--w-text);
  outline-offset: -4px;
}

.w-edit::placeholder {
  color: var(--w-disabled);
}

.w-edit.is-readonly {
  background: var(--w-face);
  color: var(--w-disabled);
}

.w-edit.is-disabled {
  background: var(--w-face);
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-edit.is-error {
  /* Error state: navy selection frame, no red (Win32 has no such colour). */
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow),
    0 0 0 1px var(--w-select-bg);
}
</style>
