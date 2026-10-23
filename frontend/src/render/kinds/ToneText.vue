<script setup lang="ts">
import { computed } from 'vue'
import type { UiNode } from '../../types'

const props = defineProps<{ node: UiNode }>()
const tone = computed(() => (props.node.p?.tone ? `tone-${props.node.p.tone}` : ''))
</script>

<template>
  <div class="n-text n-wrap" :class="tone">{{ node.l }}</div>
</template>
