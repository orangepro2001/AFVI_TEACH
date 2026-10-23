<script setup lang="ts">
withDefaults(defineProps<{ label?: string }>(), { label: '' })
</script>

<template>
  <div class="w-menu w-bevel-raised" role="menu">
    <slot />
  </div>
</template>

<style scoped>
.w-menu {
  min-width: 120px;
  padding: 2px 0;
  background: var(--w-face);
  color: var(--w-text);
}
</style>
