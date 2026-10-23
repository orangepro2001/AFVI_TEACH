<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { splitMnemonic } from '../utils/mnemonic'
import type { WMenuItemModel } from '../types'

const props = withDefaults(
  defineProps<{
    model?: WMenuItemModel[]
    height?: number
  }>(),
  {
    model: () => [],
    height: 20,
  },
)

const emit = defineEmits<{ (e: 'select', item: WMenuItemModel): void }>()

const openIndex = ref<number | null>(null)
const bar = ref<HTMLElement | null>(null)

const roots = computed(() => props.model)

function toggle(i: number) {
  openIndex.value = openIndex.value === i ? null : i
}

function run(item: WMenuItemModel) {
  item.action?.()
  emit('select', item)
  openIndex.value = null
}

function onDocClick(ev: MouseEvent) {
  if (openIndex.value === null) return
  if (bar.value && !bar.value.contains(ev.target as Node)) openIndex.value = null
}

onMounted(() => document.addEventListener('mousedown', onDocClick))
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocClick))
</script>

<template>
  <div ref="bar" class="w-menubar w-selectable" role="menubar" :style="{ height: `${height}px` }">
    <div
      v-for="(item, i) in roots"
      :key="i"
      class="w-menubar__root"
      :class="{ 'is-open': openIndex === i, 'is-disabled': item.disabled }"
      role="menuitem"
      :aria-disabled="item.disabled || undefined"
      @click="!item.disabled && toggle(i)"
    >
      <span class="w-menubar__root-label">
        <template v-for="(c, ci) in splitMnemonic(item.label ?? '')" :key="ci">
          <u v-if="c.mnemonic" class="w-mnemonic">{{ c.ch }}</u>
          <template v-else>{{ c.ch }}</template>
        </template>
      </span>

      <div v-if="openIndex === i" class="w-menubar__popup w-bevel-raised" role="menu">
        <template v-for="(sub, si) in item.children ?? []" :key="si">
          <div v-if="sub.separator" class="w-menubar__sep" role="separator" />
          <div
            v-else
            class="w-menubar__entry"
            :class="{ 'is-disabled': sub.disabled, 'is-checked': sub.checked }"
            role="menuitem"
            :aria-disabled="sub.disabled || undefined"
            @click.stop="!sub.disabled && run(sub)"
          >
            <span class="w-menubar__check" aria-hidden="true">
              <span v-if="sub.checked" class="w-menubar__tick" />
            </span>
            <span v-if="sub.icon" class="w-menubar__icon">{{ sub.icon }}</span>
            <span class="w-menubar__entry-label">{{ sub.label }}</span>
            <span v-if="sub.accelerator" class="w-menubar__accel">{{ sub.accelerator }}</span>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.w-menubar {
  display: flex;
  align-items: stretch;
  background: var(--w-face);
  border-bottom: 1px solid var(--w-shadow);
  position: relative;
  flex: 0 0 auto;
}

.w-menubar__root {
  display: flex;
  align-items: center;
  padding: 0 7px;
  cursor: default;
  position: relative;
}

.w-menubar__root.is-open {
  background: var(--w-select-bg);
  color: var(--w-select-text);
}

.w-menubar__root.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-menubar__popup {
  position: absolute;
  z-index: var(--w-z-dropdown);
  top: 100%;
  left: 0;
  min-width: 170px;
  padding: 2px 0;
  background: var(--w-face);
  color: var(--w-text);
}

.w-menubar__entry {
  display: flex;
  align-items: center;
  gap: 4px;
  height: var(--w-row-h);
  padding: 0 8px 0 2px;
  white-space: nowrap;
  cursor: default;
}

.w-menubar__entry:hover:not(.is-disabled) {
  background: var(--w-select-bg);
  color: var(--w-select-text);
}

.w-menubar__entry.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-menubar__check {
  flex: 0 0 16px;
  width: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.w-menubar__tick {
  width: 7px;
  height: 4px;
  border-left: 2px solid currentColor;
  border-bottom: 2px solid currentColor;
  transform: rotate(-45deg) translate(1px, -2px);
}

.w-menubar__icon {
  flex: 0 0 16px;
  text-align: center;
}

.w-menubar__entry-label {
  flex: 1 1 auto;
}

.w-menubar__accel {
  flex: 0 0 auto;
  padding-left: 20px;
  color: var(--w-disabled);
}

.w-menubar__entry:hover:not(.is-disabled) .w-menubar__accel {
  color: var(--w-select-text);
}

.w-menubar__sep {
  height: 2px;
  margin: 3px 2px;
  border-top: 1px solid var(--w-shadow);
  border-bottom: 1px solid var(--w-hilight);
}

.w-mnemonic {
  text-decoration: underline;
}
</style>
