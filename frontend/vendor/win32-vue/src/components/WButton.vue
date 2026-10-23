<script setup lang="ts">
import { computed } from 'vue'
import { splitMnemonic } from '../utils/mnemonic'

const props = withDefaults(
  defineProps<{
    label?: string
    variant?: 'push' | 'default' | 'flat'
    size?: 'sm' | 'md'
    disabled?: boolean
    active?: boolean
    icon?: string
    iconOnly?: boolean
    title?: string
  }>(),
  {
    label: '',
    variant: 'push',
    size: 'md',
    disabled: false,
    active: false,
    icon: '',
    iconOnly: false,
    title: '',
  },
)

const emit = defineEmits<{
  (e: 'click', ev: MouseEvent): void
  (e: 'dblclick', ev: MouseEvent): void
  (e: 'focus', ev: FocusEvent): void
  (e: 'blur', ev: FocusEvent): void
}>()

const chars = computed(() => splitMnemonic(props.label))
const showLabel = computed(() => !props.iconOnly && props.label.length > 0)
</script>

<template>
  <button
    type="button"
    class="w-btn w-selectable"
    :class="[
      `w-btn--${variant}`,
      `w-btn--${size}`,
      { 'is-active': active, 'is-disabled': disabled },
    ]"
    :disabled="disabled"
    :title="title || undefined"
    @click="emit('click', $event)"
    @dblclick="emit('dblclick', $event)"
    @focus="emit('focus', $event)"
    @blur="emit('blur', $event)"
  >
    <span v-if="icon" class="w-btn__icon" aria-hidden="true">{{ icon }}</span>
    <span v-if="showLabel" class="w-btn__label">
      <template v-for="(c, i) in chars" :key="i">
        <u v-if="c.mnemonic" class="w-mnemonic">{{ c.ch }}</u>
        <template v-else>{{ c.ch }}</template>
      </template>
    </span>
  </button>
</template>

<style scoped>
.w-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-width: 75px;
  padding: 0 8px;
  height: var(--w-btn-h);
  background: var(--w-face);
  color: var(--w-text);
  border: none;
  cursor: default;
  position: relative;
  /* Raised 3D edge: outer hilight/dkshadow + inner light/shadow. */
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-btn--sm {
  height: var(--w-btn-h-sm);
  min-width: 60px;
}

.w-btn--flat {
  min-width: 0;
  padding: 0 4px;
  background: transparent;
  box-shadow: none;
}

.w-btn--default {
  /* Win32 default button wears an extra heavy black ring. */
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light),
    0 0 0 1px var(--w-dkshadow);
}

.w-btn:active:not(.is-disabled),
.w-btn.is-active {
  /* Pressed: edges invert and the face content shifts 1px down-right. */
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
  padding: 1px 7px 0 9px;
}

.w-btn--default:active:not(.is-disabled),
.w-btn--default.is-active {
  padding: 1px 7px 0 9px;
}

.w-btn.is-disabled {
  cursor: default;
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-btn:focus-visible {
  outline: 1px dotted var(--w-text);
  outline-offset: -4px;
}

.w-btn__icon {
  font-size: 12px;
  line-height: 1;
}

.w-btn__label {
  white-space: nowrap;
}

.w-mnemonic {
  text-decoration: underline;
}
</style>
