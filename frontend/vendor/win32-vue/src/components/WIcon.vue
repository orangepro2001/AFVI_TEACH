<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    src?: string
    glyph?: string
    size?: 16 | 24 | 32 | number
    shape?: 'none' | 'square' | 'circle'
    title?: string
    disabled?: boolean
  }>(),
  {
    src: '',
    glyph: '',
    size: 16,
    shape: 'none',
    title: '',
    disabled: false,
  },
)

const px = computed(() => (typeof props.size === 'number' ? props.size : props.size))
</script>

<template>
  <span
    class="w-icon"
    :class="[`w-icon--${shape}`, { 'is-disabled': disabled }]"
    :style="{ width: `${px}px`, height: `${px}px` }"
    :title="title || undefined"
    aria-hidden="true"
  >
    <img v-if="src" class="w-icon__img" :src="src" :alt="''" :style="{ width: `${px}px`, height: `${px}px` }" />
    <span v-else class="w-icon__glyph" :style="{ fontSize: `${Math.round(px * 0.85)}px` }">
      {{ glyph }}
    </span>
  </span>
</template>

<style scoped>
.w-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  line-height: 1;
}

.w-icon--square {
  background: var(--w-face);
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-icon--circle {
  border-radius: 50%;
  background: var(--w-face);
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight);
}

.w-icon__img {
  image-rendering: pixelated;
  display: block;
}

.w-icon.is-disabled {
  opacity: 0.45;
}
</style>
