<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    type?: 'info' | 'warning' | 'error' | 'question'
    title?: string
    message?: string
    buttons?: string[]
    width?: number | string
  }>(),
  {
    type: 'info',
    title: 'Message',
    message: '',
    buttons: () => ['OK'],
    width: '320px',
  },
)

const emit = defineEmits<{ (e: 'action', button: string): void }>()

const width = computed(() =>
  typeof props.width === 'number' ? `${props.width}px` : props.width,
)
</script>

<template>
  <div class="w-msgbox w-bevel-window" :style="{ width }" role="alertdialog">
    <div class="w-msgbox__title">
      <span class="w-msgbox__title-text">{{ title }}</span>
    </div>
    <div class="w-msgbox__body">
      <span class="w-msgbox__icon" :class="`is-${type}`" aria-hidden="true">
        <span class="w-msgbox__glyph">{{
          type === 'error' ? '×' : type === 'warning' ? '!' : type === 'question' ? '?' : 'i'
        }}</span>
      </span>
      <p class="w-msgbox__message">{{ message }}</p>
    </div>
    <div class="w-msgbox__footer">
      <button
        v-for="b in buttons"
        :key="b"
        type="button"
        class="w-msgbox__btn"
        :class="{ 'is-default': b === buttons[0] }"
        @click="emit('action', b)"
      >
        {{ b }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.w-msgbox {
  background: var(--w-face);
  padding: 3px;
  max-width: 100%;
}

.w-msgbox__title {
  height: var(--w-title-h);
  display: flex;
  align-items: center;
  padding: 0 4px;
  font-weight: bold;
  color: var(--w-title-text);
  background: linear-gradient(90deg, var(--w-title-a2) 0%, var(--w-title-a1) 100%);
}

.w-msgbox__body {
  display: flex;
  gap: 10px;
  padding: 12px 10px;
  align-items: flex-start;
}

.w-msgbox__icon {
  flex: 0 0 32px;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: #ffffff;
  font-weight: bold;
  font-size: 20px;
}

.w-msgbox__icon.is-info {
  background: var(--w-info);
}
.w-msgbox__icon.is-warning {
  background: var(--w-warn);
  color: #000000;
}
.w-msgbox__icon.is-error {
  background: var(--w-err);
}
.w-msgbox__icon.is-question {
  background: var(--w-info);
}

.w-msgbox__message {
  margin: 0;
  line-height: 1.45;
  white-space: pre-wrap;
}

.w-msgbox__footer {
  display: flex;
  justify-content: center;
  gap: 6px;
  padding: 4px 6px 8px;
}

.w-msgbox__btn {
  min-width: 72px;
  height: var(--w-btn-h);
  background: var(--w-face);
  border: none;
  cursor: default;
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-msgbox__btn:active {
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-msgbox__btn.is-default {
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light),
    0 0 0 1px var(--w-dkshadow);
}
</style>
