<script setup lang="ts">
withDefaults(defineProps<{ gripper?: boolean; height?: number }>(), {
  gripper: true,
  height: 20,
})
</script>

<template>
  <div class="w-statusbar" :style="{ height: `${height}px` }">
    <slot />
    <span v-if="gripper" class="w-statusbar__gripper" aria-hidden="true">
      <i /><i /><i />
    </span>
  </div>
</template>

<style scoped>
.w-statusbar {
  display: flex;
  align-items: stretch;
  background: var(--w-face);
  border-top: 1px solid var(--w-hilight);
  flex: 0 0 auto;
}

.w-statusbar__gripper {
  flex: 0 0 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
  border-left: 1px solid var(--w-shadow);
}

.w-statusbar__gripper i {
  display: block;
  width: 2px;
  height: 12px;
  background: var(--w-hilight);
  box-shadow: 1px 0 0 var(--w-shadow);
}
</style>
