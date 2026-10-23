<script setup lang="ts">
import { computed, ref } from 'vue'
import type { WListViewColumn, WListViewRow } from '../types'

const props = withDefaults(
  defineProps<{
    columns?: WListViewColumn[]
    rows?: WListViewRow[]
    modelValue?: string | string[] | null
    selectable?: boolean
    multiSelect?: boolean
    gridLines?: boolean
    rowHeight?: number
    showHeader?: boolean
    height?: string | number
  }>(),
  {
    columns: () => [],
    rows: () => [],
    modelValue: null,
    selectable: true,
    multiSelect: false,
    gridLines: false,
    rowHeight: 18,
    showHeader: true,
    height: '100%',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string | string[] | null): void
  (e: 'rowClick', row: WListViewRow, index: number): void
  (e: 'sort', key: string, dir: 'asc' | 'desc'): void
}>()

const sortKey = ref('')
const sortDir = ref<'asc' | 'desc'>('asc')

const selected = computed<string[]>(() => {
  if (Array.isArray(props.modelValue)) return props.modelValue
  return props.modelValue === null || props.modelValue === undefined ? [] : [props.modelValue]
})

const sortedRows = computed(() => {
  if (!sortKey.value) return props.rows
  const col = props.columns.find((c) => c.key === sortKey.value)
  const dir = sortDir.value === 'asc' ? 1 : -1
  return [...props.rows].sort((a, b) => {
    const av = a[sortKey.value]
    const bv = b[sortKey.value]
    if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * dir
    return String(av ?? '').localeCompare(String(bv ?? '')) * dir
  })
})

function rowKey(row: WListViewRow, index: number) {
  const c = props.columns[0]
  if (c && row[c.key] !== undefined) return String(row[c.key])
  return `row-${index}`
}

function isSelected(row: WListViewRow, index: number) {
  return selected.value.includes(rowKey(row, index))
}

function select(row: WListViewRow, index: number, ev: MouseEvent) {
  if (!props.selectable) {
    emit('rowClick', row, index)
    return
  }
  const key = rowKey(row, index)
  let next: string | string[] | null
  if (props.multiSelect) {
    const list = ev.ctrlKey || ev.metaKey ? [...selected.value] : []
    const idx = list.indexOf(key)
    if (idx >= 0) list.splice(idx, 1)
    else list.push(key)
    next = list
  } else {
    next = key
  }
  emit('update:modelValue', next)
  emit('rowClick', row, index)
}

function toggleSort(col: WListViewColumn) {
  if (!col.sortable) return
  if (sortKey.value === col.key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = col.key
    sortDir.value = 'asc'
  }
  emit('sort', col.key, sortDir.value)
}

function colWidth(col: WListViewColumn) {
  if (col.width === undefined) return 'auto'
  return typeof col.width === 'number' ? `${col.width}px` : col.width
}
</script>

<template>
  <div
    class="w-listview"
    :style="{ height: typeof height === 'number' ? height + 'px' : height }"
  >
    <div v-if="showHeader" class="w-listview__head">
      <div
        v-for="col in columns"
        :key="col.key"
        class="w-listview__col"
        :class="[`is-${col.align ?? 'left'}`, { 'is-sortable': col.sortable }]"
        :style="{ width: colWidth(col) }"
        @click="toggleSort(col)"
      >
        <span class="w-listview__col-title">{{ col.title }}</span>
        <span v-if="sortKey === col.key" class="w-listview__sortmark">
          {{ sortDir === 'asc' ? '▲' : '▼' }}
        </span>
      </div>
    </div>
    <div class="w-listview__body" :class="{ 'has-grid': gridLines }">
      <div
        v-for="(row, i) in sortedRows"
        :key="rowKey(row, i)"
        class="w-listview__row"
        :class="{ 'is-selected': isSelected(row, i) }"
        :style="{ height: `${rowHeight}px`, lineHeight: `${rowHeight}px` }"
        @click="select(row, i, $event)"
      >
        <div
          v-for="col in columns"
          :key="col.key"
          class="w-listview__cell"
          :class="`is-${col.align ?? 'left'}`"
        >
          {{ row[col.key] ?? '' }}
        </div>
      </div>
      <div v-if="sortedRows.length === 0" class="w-listview__empty">(no records)</div>
    </div>
  </div>
</template>

<style scoped>
.w-listview {
  display: flex;
  flex-direction: column;
  background: var(--w-window);
  overflow: hidden;
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-listview__head {
  display: flex;
  background: var(--w-face);
  flex: 0 0 auto;
  padding: 1px;
}

.w-listview__col {
  display: flex;
  align-items: center;
  gap: 3px;
  height: 18px;
  padding: 0 4px;
  overflow: hidden;
  white-space: nowrap;
  border-right: 1px solid var(--w-shadow);
  box-shadow:
    inset -1px -1px 0 0 var(--w-shadow),
    inset 1px 1px 0 0 var(--w-hilight);
}

.w-listview__col.is-sortable {
  cursor: default;
}

.w-listview__col-title {
  overflow: hidden;
  text-overflow: ellipsis;
}

.w-listview__sortmark {
  font-size: 8px;
  line-height: 1;
}

.w-listview__body {
  flex: 1 1 auto;
  overflow: auto;
}

.w-listview__row {
  display: flex;
  white-space: nowrap;
  cursor: default;
}

.w-listview__row.is-selected {
  background: var(--w-select-bg);
  color: var(--w-select-text);
}

.w-listview__cell {
  flex: 0 0 auto;
  padding: 0 4px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.w-listview__cell.is-left {
  text-align: left;
}
.w-listview__cell.is-center {
  text-align: center;
}
.w-listview__cell.is-right {
  text-align: right;
}

.w-listview__body.has-grid .w-listview__cell {
  border-right: 1px solid var(--w-grid);
  border-bottom: 1px solid var(--w-grid);
}

.w-listview__empty {
  padding: 4px;
  color: var(--w-disabled);
}
</style>
