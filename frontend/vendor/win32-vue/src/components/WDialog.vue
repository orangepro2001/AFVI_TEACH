<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    title?: string
    icon?: string
    width?: number | string
    modal?: boolean
    closable?: boolean
    active?: boolean
  }>(),
  {
    title: '',
    icon: '',
    width: '360px',
    modal: false,
    closable: true,
    active: true,
  },
)

const emit = defineEmits<{ (e: 'close'): void }>()

const width = computed(() =>
  typeof props.width === 'number' ? `${props.width}px` : props.width,
)
</script>

<template>
  <div class="w-dialog-layer" :class="{ 'is-modal': modal }">
    <div class="w-dialog w-bevel-window" :style="{ width }" role="dialog">
      <div class="w-dialog__title" :class="{ 'is-active': active }">
        <span v-if="icon" class="w-dialog__icon" aria-hidden="true">{{ icon }}</span>
        <span class="w-dialog__title-text">{{ title }}</span>
        <button
          v-if="closable"
          type="button"
          class="w-dialog__close"
          tabindex="-1"
          aria-label="Close"
          @click="emit('close')"
        >
          <span class="w-dialog__close-glyph" />
        </button>
      </div>
      <div class="w-dialog__body">
        <slot />
      </div>
      <div v-if="$slots.footer" class="w-dialog__footer">
        <slot name="footer" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.w-dialog-layer {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.w-dialog-layer.is-modal {
  position: fixed;
  inset: 0;
  z-index: var(--w-z-dialog);
  background: rgba(0, 0, 0, 0.18);
}

.w-dialog {
  background: var(--w-face);
  color: var(--w-text);
  padding: 3px;
  display: flex;
  flex-direction: column;
  max-height: 100%;
}

.w-dialog__title {
  display: flex;
  align-items: center;
  height: var(--w-title-h);
  padding: 0 2px 0 4px;
  font-weight: bold;
  color: var(--w-title-text);
  background: linear-gradient(90deg, var(--w-title-i2) 0%, var(--w-title-i1) 100%);
  color: var(--w-title-text-inactive);
}

.w-dialog__title.is-active {
  background: linear-gradient(90deg, var(--w-title-a2) 0%, var(--w-title-a1) 100%);
  color: var(--w-title-text);
}

.w-dialog__icon {
  margin-right: 4px;
}

.w-dialog__title-text {
  flex: 1 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.w-dialog__close {
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

.w-dialog__close-glyph {
  position: relative;
  width: 8px;
  height: 8px;
}

.w-dialog__close-glyph::before,
.w-dialog__close-glyph::after {
  content: '';
  position: absolute;
  left: -1px;
  top: 3px;
  width: 10px;
  height: 2px;
  background: var(--w-text);
}

.w-dialog__close-glyph::before {
  transform: rotate(45deg);
}

.w-dialog__close-glyph::after {
  transform: rotate(-45deg);
}

.w-dialog__body {
  padding: 6px;
  flex: 1 1 auto;
  overflow: auto;
}

.w-dialog__footer {
  padding: 4px 6px 6px;
  display: flex;
  justify-content: flex-end;
  gap: 5px;
}
</style>
