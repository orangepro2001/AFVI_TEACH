<script setup lang="ts">
withDefaults(defineProps<{ label?: string }>(), { label: '' })
</script>

<template>
  <fieldset class="w-groupbox">
    <legend class="w-groupbox__legend">{{ label }}</legend>
    <div class="w-groupbox__content">
      <slot />
    </div>
  </fieldset>
</template>

<style scoped>
.w-groupbox {
  border: none;
  margin: 0;
  padding: 8px 6px 6px;
  min-width: 0;
  position: relative;
}

/* Etched border: dark line offset by a white line. */
.w-groupbox::before {
  content: '';
  position: absolute;
  left: 5px;
  right: 5px;
  top: 8px;
  bottom: 4px;
  border: 1px solid var(--w-shadow);
  border-right-color: var(--w-hilight);
  border-bottom-color: var(--w-hilight);
  pointer-events: none;
}

.w-groupbox__legend {
  position: relative;
  z-index: 1;
  padding: 0 3px;
  margin-left: 2px;
  background: var(--w-face);
  float: left;
  top: -3px;
}

.w-groupbox__content {
  position: relative;
  z-index: 1;
  padding-top: 2px;
}
</style>
