<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { WTreeNode } from '../types'

const props = withDefaults(
  defineProps<{
    nodes?: WTreeNode[]
    modelValue?: string | null
    showLines?: boolean
    expandOnClick?: boolean
    disabled?: boolean
    height?: string | number
  }>(),
  {
    nodes: () => [],
    modelValue: null,
    showLines: true,
    expandOnClick: false,
    disabled: false,
    height: '100%',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string | null): void
  (e: 'nodeClick', node: WTreeNode): void
  (e: 'toggle', node: WTreeNode, expanded: boolean): void
}>()

const expanded = ref<Set<string>>(new Set())

function collectExpandable(list: WTreeNode[], acc: Set<string>) {
  for (const n of list) {
    if (n.children && n.children.length) {
      acc.add(n.id)
      collectExpandable(n.children, acc)
    }
  }
}

watch(
  () => props.nodes,
  (v) => {
    const acc = new Set<string>()
    collectExpandable(v, acc)
    expanded.value = acc
  },
  { immediate: true, deep: true },
)

interface FlatNode {
  node: WTreeNode
  depth: number
  hasChildren: boolean
  isExpanded: boolean
  isLast: boolean
}

const flat = computed<FlatNode[]>(() => {
  const out: FlatNode[] = []
  const walk = (list: WTreeNode[], depth: number) => {
    list.forEach((node, idx) => {
      const hasChildren = !!(node.children && node.children.length)
      const isExpanded = expanded.value.has(node.id)
      out.push({ node, depth, hasChildren, isExpanded, isLast: idx === list.length - 1 })
      if (hasChildren && isExpanded) walk(node.children!, depth + 1)
    })
  }
  walk(props.nodes, 0)
  return out
})

function toggle(node: WTreeNode) {
  if (!node.children || !node.children.length) return
  const next = new Set(expanded.value)
  const isExpanded = next.has(node.id)
  if (isExpanded) next.delete(node.id)
  else next.add(node.id)
  expanded.value = next
  emit('toggle', node, !isExpanded)
}

function onClick(node: WTreeNode) {
  emit('update:modelValue', node.id)
  emit('nodeClick', node)
  if (props.expandOnClick) toggle(node)
}
</script>

<template>
  <div
    class="w-treeview"
    :style="{ height: typeof height === 'number' ? height + 'px' : height }"
    role="tree"
  >
    <div
      v-for="f in flat"
      :key="f.node.id"
      class="w-treeview__row"
      :class="{ 'is-selected': modelValue === f.node.id }"
      :style="{ paddingLeft: `${f.depth * 16 + 3}px` }"
      role="treeitem"
      :aria-expanded="f.hasChildren ? f.isExpanded : undefined"
      @click="onClick(f.node)"
    >
      <span v-if="showLines" class="w-treeview__lines" aria-hidden="true">
        <span v-for="d in f.depth" :key="d" class="w-treeview__line" />
      </span>
      <span
        class="w-treeview__toggle"
        :class="{ 'is-empty': !f.hasChildren, 'is-open': f.isExpanded }"
        @click.stop="toggle(f.node)"
      >
        <span v-if="f.hasChildren" class="w-treeview__toggle-sign" />
      </span>
      <span v-if="f.node.icon" class="w-treeview__icon">{{ f.node.icon }}</span>
      <span class="w-treeview__label">{{ f.node.label }}</span>
    </div>
    <div v-if="flat.length === 0" class="w-treeview__empty">(empty)</div>
  </div>
</template>

<style scoped>
.w-treeview {
  overflow: auto;
  background: var(--w-window);
  padding: 2px 0;
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-treeview__row {
  display: flex;
  align-items: center;
  height: var(--w-row-h);
  line-height: var(--w-row-h);
  white-space: nowrap;
  cursor: default;
}

.w-treeview__row.is-selected {
  background: var(--w-select-bg);
  color: var(--w-select-text);
}

.w-treeview__lines {
  display: flex;
  flex: 0 0 auto;
}

.w-treeview__line {
  width: 16px;
  height: 16px;
  border-left: 1px solid var(--w-shadow);
  align-self: flex-start;
}

.w-treeview__toggle {
  flex: 0 0 9px;
  width: 9px;
  height: 9px;
  margin: 0 3px 0 0;
  background: var(--w-window);
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
  position: relative;
}

.w-treeview__toggle.is-empty {
  background: transparent;
  box-shadow: none;
}

.w-treeview__toggle-sign::before,
.w-treeview__toggle-sign::after {
  content: '';
  position: absolute;
  background: var(--w-text);
}

.w-treeview__toggle-sign::before {
  left: 1px;
  right: 1px;
  top: 3px;
  height: 1px;
}

.w-treeview__toggle-sign::after {
  top: 1px;
  bottom: 1px;
  left: 3px;
  width: 1px;
}

.w-treeview__toggle.is-open .w-treeview__toggle-sign::after {
  display: none;
}

.w-treeview__icon {
  margin-right: 3px;
}

.w-treeview__label {
  overflow: hidden;
  text-overflow: ellipsis;
}

.w-treeview__empty {
  padding: 3px;
  color: var(--w-disabled);
}
</style>
