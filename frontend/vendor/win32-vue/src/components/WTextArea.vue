<script setup lang="ts">
import { computed, useAttrs } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    rows?: number
    cols?: number
    readonly?: boolean
    disabled?: boolean
    wrap?: 'off' | 'soft' | 'hard'
    placeholder?: string
  }>(),
  {
    modelValue: '',
    rows: 4,
    cols: 20,
    readonly: false,
    disabled: false,
    wrap: 'soft',
    placeholder: '',
  },
)

const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const attrs = useAttrs()

const wrapAttr = computed(() => (props.wrap === 'off' ? 'off' : props.wrap))

function onInput(ev: Event) {
  emit('update:modelValue', (ev.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <textarea
    v-bind="attrs"
    class="w-textarea"
    :class="{ 'is-disabled': disabled, 'is-readonly': readonly }"
    :value="modelValue"
    :rows="rows"
    :cols="cols"
    :readonly="readonly"
    :disabled="disabled"
    :placeholder="placeholder || undefined"
    :wrap="wrapAttr"
    @input="onInput"
  />
</template>

<style scoped>
.w-textarea {
  padding: 2px 3px;
  background: var(--w-window);
  color: var(--w-window-text);
  border: none;
  resize: none;
  overflow: auto;
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-textarea:focus {
  outline: none;
}

.w-textarea:focus-visible {
  outline: 1px dotted var(--w-text);
  outline-offset: -4px;
}

.w-textarea.is-readonly {
  background: var(--w-face);
  color: var(--w-disabled);
}

.w-textarea.is-disabled {
  background: var(--w-face);
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}
</style>
