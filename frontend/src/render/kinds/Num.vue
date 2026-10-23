<script setup lang="ts">
import { computed } from 'vue'
import { WEdit } from '@win32-vue/win32'
import type { UiNode } from '../../types'
import { send, setLocal, shown } from '../../bridge'
import { fmtNum } from '../fmt'

const props = defineProps<{ node: UiNode }>()
const p = computed(() => props.node.p ?? {})
const val = computed(() => shown(props.node.id, props.node.v))
function onInput(v: string) {
  const f = parseFloat(v)
  if (isNaN(f) || !props.node.id) return
  setLocal(props.node.id, f)
  send({ id: props.node.id, op: 'set', v: f })
}
</script>

<template>
  <WEdit class="n-num" type="number" :data-id="node.id" :model-value="fmtNum(val, p.fmt)" width="72px" align="right"
    @update:model-value="onInput" />
</template>
