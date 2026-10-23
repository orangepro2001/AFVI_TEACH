<script setup lang="ts">
import { computed, ref, watch } from 'vue'

export interface WTabModel {
  id: string
  label: string
  icon?: string
  disabled?: boolean
}

const props = withDefaults(
  defineProps<{
    model?: WTabModel[]
    modelValue?: string
    position?: 'top' | 'bottom'
  }>(),
  {
    model: () => [],
    modelValue: '',
    position: 'top',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
  (e: 'change', v: string): void
}>()

const current = ref(props.modelValue)

watch(
  () => props.modelValue,
  (v) => {
    if (v) current.value = v
  },
  { immediate: true },
)

watch(
  () => props.model,
  (list) => {
    if (!list.length) return
    if (!list.some((t) => t.id === current.value)) current.value = list[0].id
  },
  { immediate: true },
)

const activeId = computed(() => current.value)

function select(tab: WTabModel) {
  if (tab.disabled) return
  current.value = tab.id
  emit('update:modelValue', tab.id)
  emit('change', tab.id)
}
</script>

<template>
  <div class="w-tabbar" :class="`w-tabbar--${position}`" role="tablist">
    <div
      v-for="tab in model"
      :key="tab.id"
      class="w-tab"
      :class="{ 'is-active': activeId === tab.id, 'is-disabled': tab.disabled }"
      role="tab"
      :aria-selected="activeId === tab.id"
      @click="select(tab)"
    >
      <span v-if="tab.icon" class="w-tab__icon" aria-hidden="true">{{ tab.icon }}</span>
      <span class="w-tab__label">{{ tab.label }}</span>
    </div>
  </div>
</template>

<style scoped>
.w-tabbar {
  display: flex;
  flex: 0 0 auto;
  padding: 2px 0 0;
  background: var(--w-face);
  border-bottom: 1px solid var(--w-shadow);
}

.w-tabbar--bottom {
  border-bottom: none;
  border-top: 1px solid var(--w-shadow);
  padding: 0 0 2px;
}

.w-tabbar--bottom .w-tab {
  top: -2px;
}

.w-tab {
  display: flex;
  align-items: center;
  gap: 4px;
  height: var(--w-tab-h);
  padding: 0 9px;
  background: var(--w-face);
  cursor: default;
  white-space: nowrap;
  position: relative;
  border: 1px solid var(--w-shadow);
  border-bottom: none;
  margin-right: -1px;
}

.w-tab.is-active {
  top: 0;
  height: calc(var(--w-tab-h) + 2px);
  padding-top: 2px;
  z-index: 2;
}

.w-tabbar--bottom .w-tab.is-active {
  top: -2px;
  padding-top: 0;
}

.w-tab.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-tab__icon {
  font-size: 12px;
  line-height: 1;
}
</style>
