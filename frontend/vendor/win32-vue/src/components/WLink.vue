<script setup lang="ts">
withDefaults(
  defineProps<{
    href?: string
    disabled?: boolean
  }>(),
  {
    href: '#',
    disabled: false,
  },
)

const emit = defineEmits<{ (e: 'click', ev: MouseEvent): void }>()
</script>

<template>
  <a
    class="w-link"
    :class="{ 'is-disabled': disabled }"
    :href="disabled ? undefined : href"
    :aria-disabled="disabled || undefined"
    @click="(ev: MouseEvent) => !disabled && emit('click', ev)"
  >
    <slot />
  </a>
</template>

<style scoped>
.w-link {
  color: var(--w-select-bg);
  text-decoration: underline;
  cursor: pointer;
}

.w-link:hover {
  color: var(--w-err);
}

.w-link.is-disabled {
  color: var(--w-disabled);
  text-shadow: 1px 1px 0 var(--w-disabled-shadow);
  cursor: default;
  text-decoration: none;
}
</style>
