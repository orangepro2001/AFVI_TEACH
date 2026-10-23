<script setup lang="ts">
withDefaults(
  defineProps<{
    open?: boolean
    label?: string
    size?: number
  }>(),
  {
    open: false,
    label: '',
    size: 48,
  },
)
</script>

<template>
  <span class="w-valve" :class="{ 'is-open': open }">
    <svg
      class="w-valve__svg"
      :width="size"
      :height="size * 0.8"
      viewBox="0 0 60 48"
      shape-rendering="crispEdges"
    >
      <!-- flanges -->
      <rect x="0" y="16" width="5" height="14" fill="#d4d0c8" stroke="#000000" />
      <rect x="55" y="16" width="5" height="14" fill="#d4d0c8" stroke="#000000" />
      <!-- pipe body -->
      <rect x="4" y="20" width="52" height="6" fill="#d4d0c8" stroke="#000000" />
      <!-- bonnet -->
      <rect x="24" y="14" width="12" height="6" fill="#d4d0c8" stroke="#000000" />
      <!-- gate: up = open, down = shut -->
      <rect
        x="26"
        :y="open ? 8 : 20"
        width="8"
        :height="open ? 6 : 12"
        fill="#d4d0c8"
        stroke="#000000"
      />
      <!-- stem -->
      <rect x="28" :y="open ? 2 : 20" width="4" :height="open ? 6 : 0" fill="#808080" />
      <!-- handwheel -->
      <rect x="21" y="0" width="18" height="4" fill="#d4d0c8" stroke="#000000" />
      <rect x="28" y="4" width="4" height="4" fill="#d4d0c8" stroke="#000000" />
    </svg>
    <span v-if="label" class="w-valve__label">{{ label }}</span>
    <span class="w-valve__state">{{ open ? 'OPEN' : 'SHUT' }}</span>
  </span>
</template>

<style scoped>
.w-valve {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  line-height: 1.2;
}

.w-valve__svg {
  display: block;
}

.w-valve__label {
  font-size: 10px;
}

.w-valve__state {
  font-size: 10px;
  font-weight: bold;
  color: var(--w-err);
}

.w-valve.is-open .w-valve__state {
  color: var(--w-ok);
}
</style>
