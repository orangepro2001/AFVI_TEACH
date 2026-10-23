<script setup lang="ts">
withDefaults(
  defineProps<{
    icon?: string
    label?: string
    text?: 'both' | 'icon' | 'label'
    disabled?: boolean
    checked?: boolean
    tooltip?: string
  }>(),
  {
    icon: '',
    label: '',
    text: 'icon',
    disabled: false,
    checked: false,
    tooltip: '',
  },
)

const emit = defineEmits<{ (e: 'click', ev: MouseEvent): void }>()
</script>

<template>
  <button
    type="button"
    class="w-toolbtn w-selectable"
    :class="{
      'is-disabled': disabled,
      'is-checked': checked,
      'has-label': text !== 'icon',
      'is-icon-only': text === 'icon',
    }"
    :disabled="disabled"
    :title="tooltip || undefined"
    @click="emit('click', $event)"
  >
    <span v-if="icon" class="w-toolbtn__icon" aria-hidden="true">{{ icon }}</span>
    <span v-if="text !== 'icon' && label" class="w-toolbtn__label">{{ label }}</span>
  </button>
</template>

<style scoped>
.w-toolbtn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  height: 22px;
  min-width: 22px;
  padding: 0 2px;
  background: transparent;
  border: none;
  cursor: default;
  color: var(--w-text);
}

.w-toolbtn.has-label {
  padding: 0 5px;
}

.w-toolbtn:hover:not(.is-disabled) {
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-toolbtn:active:not(.is-disabled),
.w-toolbtn.is-checked {
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
  background: var(--w-face-alt);
}

.w-toolbtn.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-toolbtn:focus-visible {
  outline: 1px dotted var(--w-text);
  outline-offset: -2px;
}

.w-toolbtn__icon {
  font-size: 13px;
  line-height: 1;
}

.w-toolbtn__label {
  font-size: 11px;
  white-space: nowrap;
}
</style>
