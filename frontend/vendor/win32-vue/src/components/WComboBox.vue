<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { WComboOption } from '../types'

const props = withDefaults(
  defineProps<{
    modelValue?: string | number | null
    options?: WComboOption[]
    editable?: boolean
    disabled?: boolean
    width?: string | number
    placeholder?: string
  }>(),
  {
    modelValue: null,
    options: () => [],
    editable: false,
    disabled: false,
    width: '160px',
    placeholder: '',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string | number | null): void
  (e: 'change', v: string | number | null): void
  (e: 'open'): void
  (e: 'close'): void
}>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
const draft = ref('')

const selected = computed(() =>
  props.options.find((o) => o.value === props.modelValue) ?? null,
)

const displayText = computed(() => {
  if (props.editable) return draft.value
  return selected.value ? selected.value.label : ''
})

watch(
  () => props.modelValue,
  () => {
    if (!props.editable && selected.value) draft.value = selected.value.label
  },
  { immediate: true },
)

watch(
  () => props.editable,
  (v) => {
    if (v && selected.value) draft.value = selected.value.label
  },
)

function toggle() {
  if (props.disabled) return
  open.value = !open.value
  if (open.value) emit('open')
  else emit('close')
}

function pick(o: WComboOption) {
  if (o.disabled) return
  if (!props.editable) {
    emit('update:modelValue', o.value)
    emit('change', o.value)
  }
  open.value = false
  emit('close')
}

function onInput(ev: Event) {
  draft.value = (ev.target as HTMLInputElement).value
}

function onDocClick(ev: MouseEvent) {
  if (!open.value) return
  if (root.value && !root.value.contains(ev.target as Node)) {
    open.value = false
    emit('close')
  }
}

onMounted(() => document.addEventListener('mousedown', onDocClick))
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocClick))
</script>

<template>
  <div ref="root" class="w-combo" :style="{ width: typeof width === 'number' ? width + 'px' : width }">
    <div class="w-combo__control">
      <input
        v-if="editable"
        class="w-combo__input"
        :value="draft"
        :disabled="disabled"
        :placeholder="placeholder || undefined"
        @input="onInput"
      />
      <span v-else class="w-combo__text" :class="{ 'is-empty': !displayText }">
        {{ displayText || placeholder }}
      </span>
      <button
        type="button"
        class="w-combo__arrow"
        :class="{ 'is-open': open, 'is-disabled': disabled }"
        :disabled="disabled"
        tabindex="-1"
        aria-label="Open dropdown"
        @mousedown.prevent
        @click="toggle"
      >
        <span class="w-combo__arrow-glyph" />
      </button>
    </div>

    <ul v-if="open" class="w-combo__list w-bevel-raised" role="listbox">
      <li
        v-for="(o, i) in options"
        :key="i"
        class="w-combo__item"
        :class="{
          'is-selected': o.value === modelValue,
          'is-disabled': o.disabled,
        }"
        role="option"
        :aria-selected="o.value === modelValue"
        @click="pick(o)"
      >
        {{ o.label }}
      </li>
      <li v-if="options.length === 0" class="w-combo__item is-empty">(no items)</li>
    </ul>
  </div>
</template>

<style scoped>
.w-combo {
  display: inline-block;
  position: relative;
  vertical-align: middle;
}

.w-combo__control {
  display: flex;
  height: var(--w-field-h);
  background: var(--w-window);
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-combo__text,
.w-combo__input {
  flex: 1 1 auto;
  min-width: 0;
  padding: 0 3px;
  background: transparent;
  border: none;
  outline: none;
  line-height: calc(var(--w-field-h) - 2px);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.w-combo__text.is-empty {
  color: var(--w-disabled);
}

.w-combo__arrow {
  flex: 0 0 17px;
  width: 17px;
  background: var(--w-face);
  border: none;
  cursor: default;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-combo__arrow.is-open {
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-combo__arrow-glyph {
  width: 0;
  height: 0;
  border-left: 3px solid transparent;
  border-right: 3px solid transparent;
  border-top: 4px solid var(--w-text);
}

.w-combo__list {
  position: absolute;
  z-index: var(--w-z-dropdown);
  top: 100%;
  left: 0;
  right: 0;
  margin: 0;
  padding: 2px 0;
  list-style: none;
  background: var(--w-face);
  max-height: 220px;
  overflow: auto;
}

.w-combo__item {
  padding: 1px 6px;
  height: var(--w-row-h);
  line-height: calc(var(--w-row-h) - 2px);
  white-space: nowrap;
  cursor: default;
}

.w-combo__item.is-selected {
  background: var(--w-select-bg);
  color: var(--w-select-text);
}

.w-combo__item.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-combo__item.is-empty {
  color: var(--w-disabled);
}
</style>
