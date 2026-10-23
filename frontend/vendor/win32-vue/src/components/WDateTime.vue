<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    format?: string
    showTime?: boolean
    disabled?: boolean
    width?: string | number
  }>(),
  {
    modelValue: '',
    format: 'yyyy-MM-dd',
    showTime: false,
    disabled: false,
    width: '140px',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
  (e: 'change', v: string): void
}>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)

function parse(v: string): Date | null {
  if (!v) return null
  const m = v.match(/^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?$/)
  if (!m) return null
  const d = new Date(
    Number(m[1]),
    Number(m[2]) - 1,
    Number(m[3]),
    Number(m[4] ?? 0),
    Number(m[5] ?? 0),
    Number(m[6] ?? 0),
  )
  return Number.isNaN(d.getTime()) ? null : d
}

const current = computed(() => parse(props.modelValue) ?? new Date())
const viewYear = ref(current.value.getFullYear())
const viewMonth = ref(current.value.getMonth())

const WEEKDAYS = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su']
const MONTHS = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
]

interface Cell {
  day: number
  inMonth: boolean
  iso: string
}

const cells = computed<Cell[]>(() => {
  const first = new Date(viewYear.value, viewMonth.value, 1)
  // Monday-first offset
  const offset = (first.getDay() + 6) % 7
  const out: Cell[] = []
  const start = new Date(viewYear.value, viewMonth.value, 1 - offset)
  for (let i = 0; i < 42; i++) {
    const d = new Date(start.getFullYear(), start.getMonth(), start.getDate() + i)
    out.push({
      day: d.getDate(),
      inMonth: d.getMonth() === viewMonth.value,
      iso: isoOf(d),
    })
  }
  return out
})

function pad(n: number) {
  return String(n).padStart(2, '0')
}

function isoOf(d: Date) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function formatDate(d: Date): string {
  const date = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  if (!props.showTime) return date
  return `${date} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function pick(cell: Cell) {
  const d = parse(cell.iso) ?? new Date()
  if (props.showTime) {
    const src = current.value
    d.setHours(src.getHours(), src.getMinutes(), src.getSeconds())
  }
  const v = formatDate(d)
  emit('update:modelValue', v)
  emit('change', v)
  open.value = false
}

function today() {
  const d = new Date()
  viewYear.value = d.getFullYear()
  viewMonth.value = d.getMonth()
  pick({ day: d.getDate(), inMonth: true, iso: isoOf(d) })
}

function shiftMonth(delta: number) {
  const d = new Date(viewYear.value, viewMonth.value + delta, 1)
  viewYear.value = d.getFullYear()
  viewMonth.value = d.getMonth()
}

function onDocClick(ev: MouseEvent) {
  if (!open.value) return
  if (root.value && !root.value.contains(ev.target as Node)) open.value = false
}

onMounted(() => document.addEventListener('mousedown', onDocClick))
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocClick))
</script>

<template>
  <div
    ref="root"
    class="w-datetime"
    :style="{ width: typeof width === 'number' ? width + 'px' : width }"
  >
    <div class="w-datetime__field">
      <input
        class="w-datetime__input w-mono"
        :value="modelValue"
        :disabled="disabled"
        :placeholder="format"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      />
    </div>
    <button
      type="button"
      class="w-datetime__btn"
      :disabled="disabled"
      tabindex="-1"
      aria-label="Open calendar"
      @mousedown.prevent
      @click="open = !open"
    >
      <span class="w-datetime__glyph" />
    </button>

    <div v-if="open" class="w-datetime__popup w-bevel-raised">
      <div class="w-datetime__head">
        <button type="button" class="w-datetime__nav" @click="shiftMonth(-1)">&#9664;</button>
        <span class="w-datetime__title">
          {{ MONTHS[viewMonth] }} {{ viewYear }}
        </span>
        <button type="button" class="w-datetime__nav" @click="shiftMonth(1)">&#9654;</button>
      </div>
      <table class="w-datetime__grid">
        <thead>
          <tr>
            <th v-for="w in WEEKDAYS" :key="w">{{ w }}</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td
              v-for="c in cells"
              :key="c.iso"
              :class="{ 'is-out': !c.inMonth, 'is-today': c.iso === isoOf(new Date()) }"
              @click="pick(c)"
            >
              {{ c.day }}
            </td>
          </tr>
        </tbody>
      </table>
      <div class="w-datetime__foot">
        <button type="button" class="w-datetime__today" @click="today">Today</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.w-datetime {
  display: inline-flex;
  position: relative;
  vertical-align: middle;
}

.w-datetime__field {
  flex: 1 1 auto;
  min-width: 0;
  height: var(--w-field-h);
  background: var(--w-window);
  box-shadow:
    inset -1px -1px 0 0 var(--w-hilight),
    inset 1px 1px 0 0 var(--w-dkshadow),
    inset -2px -2px 0 0 var(--w-light),
    inset 2px 2px 0 0 var(--w-shadow);
}

.w-datetime__input {
  width: 100%;
  height: 100%;
  border: none;
  outline: none;
  background: transparent;
  padding: 0 3px;
}

.w-datetime__btn {
  flex: 0 0 17px;
  background: var(--w-face);
  border: none;
  cursor: default;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-datetime__glyph {
  width: 11px;
  height: 10px;
  border: 1px solid var(--w-text);
  background: var(--w-window);
  position: relative;
}

.w-datetime__glyph::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  top: 2px;
  height: 1px;
  background: var(--w-err);
}

.w-datetime__popup {
  position: absolute;
  z-index: var(--w-z-dropdown);
  top: 100%;
  left: 0;
  margin-top: 2px;
  padding: 3px;
  background: var(--w-face);
  width: 172px;
}

.w-datetime__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 2px;
}

.w-datetime__title {
  font-weight: bold;
}

.w-datetime__nav,
.w-datetime__today {
  background: var(--w-face);
  border: none;
  cursor: default;
  font-size: 10px;
  height: 17px;
  min-width: 20px;
  padding: 0 3px;
  box-shadow:
    inset -1px -1px 0 0 var(--w-dkshadow),
    inset 1px 1px 0 0 var(--w-hilight),
    inset -2px -2px 0 0 var(--w-shadow),
    inset 2px 2px 0 0 var(--w-light);
}

.w-datetime__grid {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  background: var(--w-window);
}

.w-datetime__grid th {
  font-weight: normal;
  font-size: 10px;
  height: 15px;
  color: var(--w-disabled);
}

.w-datetime__grid td {
  text-align: center;
  height: 17px;
  font-size: 11px;
  cursor: default;
}

.w-datetime__grid td.is-out {
  color: var(--w-disabled);
}

.w-datetime__grid td.is-today {
  background: var(--w-select-bg);
  color: var(--w-select-text);
}

.w-datetime__foot {
  margin-top: 3px;
  display: flex;
  justify-content: flex-end;
}
</style>
