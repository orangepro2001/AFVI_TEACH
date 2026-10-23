<script setup lang="ts">
withDefaults(
  defineProps<{
    label?: string
    icon?: string
    active?: boolean
    disabled?: boolean
  }>(),
  {
    label: '',
    icon: '',
    active: false,
    disabled: false,
  },
)

const emit = defineEmits<{ (e: 'click'): void }>()
</script>

<template>
  <div
    class="w-tab"
    :class="{ 'is-active': active, 'is-disabled': disabled }"
    role="tab"
    :aria-selected="active"
    @click="!disabled && emit('click')"
  >
    <span v-if="icon" class="w-tab__icon" aria-hidden="true">{{ icon }}</span>
    <span class="w-tab__label">{{ label }}</span>
  </div>
</template>

<style scoped>
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
  top: 2px;
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

.w-tab.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
}

.w-tab__icon {
  font-size: 12px;
  line-height: 1;
}
</style>
