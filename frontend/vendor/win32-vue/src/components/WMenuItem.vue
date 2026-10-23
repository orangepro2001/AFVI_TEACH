<script setup lang="ts">
import { computed } from 'vue'
import { splitMnemonic } from '../utils/mnemonic'

const props = withDefaults(
  defineProps<{
    label?: string
    accelerator?: string
    icon?: string
    checked?: boolean
    radio?: boolean
    disabled?: boolean
    separator?: boolean
    submenu?: boolean
  }>(),
  {
    label: '',
    accelerator: '',
    icon: '',
    checked: false,
    radio: false,
    disabled: false,
    separator: false,
    submenu: false,
  },
)

const emit = defineEmits<{ (e: 'click'): void }>()

const chars = computed(() => splitMnemonic(props.label))
</script>

<template>
  <li
    v-if="separator"
    class="w-menu__sep"
    role="separator"
  />
  <li
    v-else
    class="w-menu__item"
    :class="{ 'is-disabled': disabled, 'is-checked': checked }"
    role="menuitem"
    :aria-disabled="disabled || undefined"
    :aria-haspopup="submenu ? 'menu' : undefined"
    @click="!disabled && emit('click')"
  >
    <span class="w-menu__check" aria-hidden="true">
      <span v-if="radio && checked" class="w-menu__radio-dot" />
      <span v-else-if="checked" class="w-menu__tick" />
    </span>
    <span v-if="icon" class="w-menu__icon" aria-hidden="true">{{ icon }}</span>
    <span class="w-menu__label">
      <template v-for="(c, i) in chars" :key="i">
        <u v-if="c.mnemonic" class="w-mnemonic">{{ c.ch }}</u>
        <template v-else>{{ c.ch }}</template>
      </template>
    </span>
    <span v-if="accelerator" class="w-menu__accel">{{ accelerator }}</span>
    <span v-if="submenu" class="w-menu__arrow" aria-hidden="true">&#9654;</span>
  </li>
</template>

<style scoped>
.w-menu__sep {
  height: 2px;
  margin: 3px 2px;
  border-top: 1px solid var(--w-shadow);
  border-bottom: 1px solid var(--w-hilight);
}

.w-menu__item {
  display: flex;
  align-items: center;
  gap: 4px;
  height: var(--w-row-h);
  padding: 0 6px 0 2px;
  white-space: nowrap;
  cursor: default;
}

.w-menu__item:hover:not(.is-disabled) {
  background: var(--w-select-bg);
  color: var(--w-select-text);
}

.w-menu__item.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-menu__check {
  flex: 0 0 16px;
  width: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.w-menu__tick {
  width: 7px;
  height: 4px;
  border-left: 2px solid currentColor;
  border-bottom: 2px solid currentColor;
  transform: rotate(-45deg) translate(1px, -2px);
}

.w-menu__radio-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}

.w-menu__icon {
  flex: 0 0 auto;
  width: 16px;
  text-align: center;
}

.w-menu__label {
  flex: 1 1 auto;
}

.w-menu__accel {
  flex: 0 0 auto;
  padding-left: 18px;
  color: var(--w-disabled);
}

.w-menu__item:hover:not(.is-disabled) .w-menu__accel {
  color: var(--w-select-text);
}

.w-menu__arrow {
  flex: 0 0 auto;
  font-size: 9px;
}

.w-mnemonic {
  text-decoration: underline;
}
</style>
