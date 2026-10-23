<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    title?: string
    icon?: string
    active?: boolean
    showMin?: boolean
    showMax?: boolean
    showClose?: boolean
    draggable?: boolean
    height?: number
  }>(),
  {
    title: '',
    icon: '',
    active: true,
    showMin: true,
    showMax: true,
    showClose: true,
    draggable: false,
    height: 18,
  },
)

const emit = defineEmits<{
  (e: 'minimize'): void
  (e: 'maximize'): void
  (e: 'close'): void
  (e: 'dblclick'): void
}>()

const cls = computed(() => (props.active ? 'is-active' : 'is-inactive'))
</script>

<template>
  <div
    class="w-titlebar w-selectable"
    :class="cls"
    :style="{ height: `${height}px`, '--wails-draggable': draggable ? 'drag' : 'no-drag' }"
    @dblclick="emit('dblclick')"
  >
    <span v-if="icon" class="w-titlebar__icon" aria-hidden="true">{{ icon }}</span>
    <span class="w-titlebar__text">{{ title }}</span>
    <span class="w-titlebar__spacer" :data-draggable="draggable ? 'true' : 'false'" />
    <span class="w-titlebar__buttons">
      <button
        v-if="showMin"
        type="button"
        class="w-titlebar__btn"
        tabindex="-1"
        aria-label="Minimize"
        @click.stop="emit('minimize')"
      >
        <span class="w-titlebar__glyph w-titlebar__glyph--min" />
      </button>
      <button
        v-if="showMax"
        type="button"
        class="w-titlebar__btn"
        tabindex="-1"
        aria-label="Maximize"
        @click.stop="emit('maximize')"
      >
        <span class="w-titlebar__glyph w-titlebar__glyph--max" />
      </button>
      <button
        v-if="showClose"
        type="button"
        class="w-titlebar__btn"
        tabindex="-1"
        aria-label="Close"
        @click.stop="emit('close')"
      >
        <span class="w-titlebar__glyph w-titlebar__glyph--close" />
      </button>
    </span>
  </div>
</template>

<style scoped>
.w-titlebar {
  display: flex;
  align-items: center;
  padding: 0 2px 0 3px;
  color: var(--w-title-text);
  font-weight: bold;
  user-select: none;
  overflow: hidden;
}

.w-titlebar.is-active {
  background: linear-gradient(90deg, var(--w-title-a2) 0%, var(--w-title-a1) 100%);
}

.w-titlebar.is-inactive {
  background: linear-gradient(90deg, var(--w-title-i2) 0%, var(--w-title-i1) 100%);
  color: var(--w-title-text-inactive);
}

.w-titlebar__icon {
  margin-right: 4px;
  font-size: 12px;
  line-height: 1;
}

.w-titlebar__text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}

.w-titlebar__spacer {
  flex: 1 1 auto;
  height: 100%;
}

.w-titlebar__buttons {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: 0 0 auto;
}

.w-titlebar__btn {
  /* The caption buttons must never start a window drag. */
  --wails-draggable: no-drag;
  width: 16px;
  height: 14px;
  padding: 0;
  border: none;
  background: var(--w-face);
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

.w-titlebar__btn:active {
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-titlebar__glyph {
  display: block;
  background: var(--w-text);
}

.w-titlebar__glyph--min {
  width: 6px;
  height: 2px;
  margin-top: 5px;
}

.w-titlebar__glyph--max {
  width: 9px;
  height: 8px;
  background: transparent;
  border: 1px solid var(--w-text);
  border-top-width: 2px;
}

.w-titlebar__glyph--close {
  width: 8px;
  height: 8px;
  background: transparent;
  position: relative;
}

.w-titlebar__glyph--close::before,
.w-titlebar__glyph--close::after {
  content: '';
  position: absolute;
  left: -1px;
  top: 3px;
  width: 10px;
  height: 2px;
  background: var(--w-text);
}

.w-titlebar__glyph--close::before {
  transform: rotate(45deg);
}

.w-titlebar__glyph--close::after {
  transform: rotate(-45deg);
}
</style>
