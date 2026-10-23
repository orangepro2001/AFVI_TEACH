<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    state?: 'on' | 'off' | 'fault' | 'blink'
    color?: 'green' | 'red' | 'yellow' | 'blue' | 'gray'
    label?: string
    size?: number
  }>(),
  {
    state: 'off',
    color: 'green',
    label: '',
    size: 16,
  },
)

const lit = computed(() => props.state === 'on' || props.state === 'blink')
const cls = computed(() => {
  if (props.state === 'fault') return 'is-fault'
  if (props.state === 'blink') return 'is-blink'
  return lit.value ? 'is-on' : 'is-off'
})
</script>

<template>
  <span class="w-lamp" :class="cls">
    <span
      class="w-lamp__bulb"
      :class="`is-${color}`"
      :style="{ width: `${size}px`, height: `${size}px` }"
    />
    <span v-if="label" class="w-lamp__label">{{ label }}</span>
  </span>
</template>

<style scoped>
.w-lamp {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.w-lamp__bulb {
  display: inline-block;
  border-radius: 50%;
  background: var(--w-neutral);
  box-shadow:
    inset -1px -1px 0 0 rgba(0, 0, 0, 0.45),
    inset 1px 1px 0 0 rgba(255, 255, 255, 0.35),
    0 0 0 1px var(--w-dkshadow);
}

.w-lamp__bulb.is-green.is-on {
  background: #00c000;
}
.w-lamp__bulb.is-red.is-on {
  background: #e00000;
}
.w-lamp__bulb.is-yellow.is-on {
  background: #e0c000;
}
.w-lamp__bulb.is-blue.is-on {
  background: #0060e0;
}
.w-lamp__bulb.is-gray.is-on {
  background: #a0a0a0;
}

.w-lamp.is-fault .w-lamp__bulb {
  background: repeating-linear-gradient(
    45deg,
    #e00000 0,
    #e00000 3px,
    #600000 3px,
    #600000 6px
  );
}

.w-lamp.is-blink .w-lamp__bulb {
  animation: w-lamp-blink 1s steps(1) infinite;
}

@keyframes w-lamp-blink {
  0%,
  50% {
    background: #00c000;
  }
  50.01%,
  100% {
    background: #006000;
  }
}

.w-lamp__label {
  font-size: 11px;
}
</style>
