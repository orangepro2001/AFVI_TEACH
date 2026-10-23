<script setup lang="ts">
withDefaults(defineProps<{ gripper?: boolean; height?: number }>(), {
  gripper: true,
  height: 20,
})
</script>

<template>
  <div class="w-toolbar w-selectable" :style="{ height: `${height}px` }">
    <span v-if="gripper" class="w-toolbar__gripper" aria-hidden="true">
      <i /><i /><i />
    </span>
    <slot />
  </div>
</template>

<style scoped>
.w-toolbar {
  display: flex;
  align-items: center;
  background: var(--w-face);
  border-bottom: 1px solid var(--w-shadow);
  padding: 0 2px;
  flex: 0 0 auto;
}

.w-toolbar__gripper {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 2px 3px 2px 1px;
  margin-right: 2px;
  border-right: 1px solid var(--w-shadow);
}

.w-toolbar__gripper i {
  display: block;
  width: 8px;
  height: 2px;
  background: var(--w-hilight);
  box-shadow: 0 1px 0 var(--w-shadow);
}
</style>
