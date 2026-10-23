<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    orientation?: 'horizontal' | 'vertical'
    value?: number
    min?: number
    max?: number
    pageSize?: number
    disabled?: boolean
  }>(),
  {
    orientation: 'vertical',
    value: 0,
    min: 0,
    max: 100,
    pageSize: 10,
    disabled: false,
  },
)

const emit = defineEmits<{ (e: 'update:value', v: number): void }>()

const ratio = computed(() => {
  const span = props.max - props.min
  if (span <= 0) return 0
  return Math.min(1, Math.max(0, (props.value - props.min) / span))
})

const thumbSize = computed(() => {
  const span = props.max - props.min
  if (span <= 0) return 1
  const p = props.pageSize / span
  return Math.min(1, Math.max(0.08, p))
})

function step(dir: number) {
  if (props.disabled) return
  const span = props.max - props.min
  const amount = Math.max(1, Math.round(span / 20))
  const next = Math.min(props.max, Math.max(props.min, props.value + dir * amount))
  if (next !== props.value) emit('update:value', next)
}
</script>

<template>
  <div class="w-scrollbar" :class="[`w-scrollbar--${orientation}`, { 'is-disabled': disabled }]">
    <button
      v-if="orientation === 'vertical'"
      type="button"
      class="w-scrollbar__btn w-scrollbar__btn--up"
      tabindex="-1"
      aria-label="Scroll up"
      @click="step(-1)"
    >
      <span class="w-scrollbar__glyph w-scrollbar__glyph--up" />
    </button>
    <div class="w-scrollbar__track">
      <div
        class="w-scrollbar__thumb"
        :style="
          orientation === 'vertical'
            ? { height: `${thumbSize * 100}%`, top: `${ratio * (100 - thumbSize * 100)}%` }
            : { width: `${thumbSize * 100}%`, left: `${ratio * (100 - thumbSize * 100)}%` }
        "
      />
    </div>
    <button
      v-if="orientation === 'vertical'"
      type="button"
      class="w-scrollbar__btn w-scrollbar__btn--down"
      tabindex="-1"
      aria-label="Scroll down"
      @click="step(1)"
    >
      <span class="w-scrollbar__glyph w-scrollbar__glyph--down" />
    </button>
  </div>
</template>

<style scoped>
.w-scrollbar {
  display: flex;
  background: var(--w-scroll-track);
  background-image: linear-gradient(
    45deg,
    var(--w-face) 25%,
    transparent 25%,
    transparent 75%,
    var(--w-face) 75%
  );
  background-size: 2px 2px;
}

.w-scrollbar--vertical {
  flex-direction: column;
  width: 16px;
}

.w-scrollbar--horizontal {
  flex-direction: row;
  height: 16px;
}

.w-scrollbar__btn {
  flex: 0 0 16px;
  height: 16px;
  background: var(--w-face);
  border: none;
  padding: 0;
  cursor: default;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-scrollbar__btn:active {
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-scrollbar__track {
  flex: 1 1 auto;
  position: relative;
}

.w-scrollbar__thumb {
  position: absolute;
  background: var(--w-face);
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-scrollbar--vertical .w-scrollbar__thumb {
  left: 0;
  right: 0;
}

.w-scrollbar--horizontal .w-scrollbar__thumb {
  top: 0;
  bottom: 0;
}

.w-scrollbar__glyph {
  width: 0;
  height: 0;
  border-left: 3px solid transparent;
  border-right: 3px solid transparent;
}

.w-scrollbar__glyph--up {
  border-bottom: 4px solid var(--w-text);
}

.w-scrollbar__glyph--down {
  border-top: 4px solid var(--w-text);
}
</style>
