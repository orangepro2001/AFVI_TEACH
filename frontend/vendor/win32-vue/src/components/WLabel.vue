<script setup lang="ts">
import { computed } from 'vue'
import { splitMnemonic, stripMnemonic } from '../utils/mnemonic'

const props = withDefaults(
  defineProps<{
    text?: string
    align?: 'left' | 'center' | 'right'
    disabled?: boolean
    for?: string
    bold?: boolean
  }>(),
  {
    text: '',
    align: 'left',
    disabled: false,
    for: '',
    bold: false,
  },
)

const chars = computed(() => splitMnemonic(props.text))
const plain = computed(() => stripMnemonic(props.text))
const htmlFor = computed(() => props.for || undefined)
</script>

<template>
  <label
    class="w-label"
    :class="[`is-${align}`, { 'is-disabled': disabled, 'is-bold': bold }]"
    :for="htmlFor"
    :title="plain"
  >
    <slot>
      <template v-for="(c, i) in chars" :key="i">
        <u v-if="c.mnemonic" class="w-mnemonic">{{ c.ch }}</u>
        <template v-else>{{ c.ch }}</template>
      </template>
    </slot>
  </label>
</template>

<style scoped>
.w-label {
  display: inline-block;
  color: var(--w-text);
  white-space: nowrap;
  cursor: default;
}

.w-label.is-left {
  text-align: left;
}
.w-label.is-center {
  text-align: center;
}
.w-label.is-right {
  text-align: right;
}

.w-label.is-bold {
  font-weight: bold;
}

.w-label.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-mnemonic {
  text-decoration: underline;
}
</style>
