<script setup lang="ts">
import { computed } from 'vue'
import { splitMnemonic } from '../utils/mnemonic'

const props = withDefaults(
  defineProps<{
    label?: string
    labelWidth?: number | string
    required?: boolean
    inline?: boolean
  }>(),
  {
    label: '',
    labelWidth: 110,
    required: false,
    inline: false,
  },
)

const chars = computed(() => splitMnemonic(props.label))
const width = computed(() =>
  typeof props.labelWidth === 'number' ? `${props.labelWidth}px` : props.labelWidth,
)
</script>

<template>
  <div class="w-field" :class="{ 'is-inline': inline }">
    <div class="w-field__label" :style="{ width }">
      <template v-for="(c, i) in chars" :key="i">
        <u v-if="c.mnemonic" class="w-mnemonic">{{ c.ch }}</u>
        <template v-else>{{ c.ch }}</template>
      </template>
      <span v-if="required" class="w-field__req" aria-hidden="true">*</span>
    </div>
    <div class="w-field__control">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.w-field {
  display: flex;
  align-items: center;
  gap: var(--w-gap);
  margin: 3px 0;
}

.w-field.is-inline {
  display: inline-flex;
}

.w-field__label {
  flex: 0 0 auto;
  text-align: right;
  white-space: nowrap;
  color: var(--w-text);
}

.w-field__req {
  color: var(--w-err);
  margin-left: 2px;
}

.w-field__control {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  align-items: center;
}

.w-mnemonic {
  text-decoration: underline;
}
</style>
