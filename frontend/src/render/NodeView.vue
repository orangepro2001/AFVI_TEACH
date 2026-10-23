<script setup lang="ts">
import { computed } from 'vue'
import {
  WButton, WCheckBox, WRadioButton, WTrackbar, WEdit, WComboBox, WGroupBox, WUpDown,
} from '@win32-vue/win32'
import type { UiNode } from '../types'
import { send, setLocal, shown } from '../bridge'
import { fmtNum } from './fmt'
import { customKinds } from './kinds'

const props = defineProps<{ node: UiNode }>()
const n = computed(() => props.node)
const p = computed(() => props.node.p ?? {})
const kids = computed(() => props.node.c ?? [])

function set(id: string | undefined, v: any) {
  if (!id) return
  setLocal(id, v)
  send({ id, op: 'set', v })
}
function click(id?: string, extra: object = {}) {
  if (id) send({ id, op: 'click', ...extra })
}

const val = computed(() => shown(n.value.id, n.value.v))
const comboOptions = computed(() => (p.value.items ?? []).map((l: string, i: number) => ({ label: l, value: i })))
const tabModel = computed(() => (p.value.items ?? []).map((l: string, i: number) => ({ id: String(i), label: l })))
const textStyle = computed(() => ({ whiteSpace: p.value.nowrap ? 'nowrap' : 'normal' }))
const tone = computed(() => (p.value.tone ? `tone-${p.value.tone}` : ''))
const sliderStep = computed(() => (p.value.int ? 1 : Math.max((p.value.max - p.value.min) / 1000, 1e-6)))

function toggle() {
  if (n.value.id) send({ id: n.value.id, op: 'toggle' })
}
function cellStyle(c: any) {
  return c.tone ? `tone-${c.tone}` : ''
}
</script>

<template>
  <!-- layout containers -->
  <template v-if="n.k === 'root'">
    <NodeView v-for="(c, i) in kids" :key="c.id ?? i" :node="c" />
  </template>

  <div v-else-if="n.k === 'row'" class="n-row">
    <NodeView v-for="(c, i) in kids" :key="c.id ?? i" :node="c" />
  </div>

  <WGroupBox v-else-if="n.k === 'group'" :label="n.l" class="n-group">
    <NodeView v-for="(c, i) in kids" :key="c.id ?? i" :node="c" />
  </WGroupBox>

  <div v-else-if="n.k === 'section'" class="n-section">
    <button class="n-sec-head w-bevel-raised" :data-id="n.id" @click="toggle">
      <span class="n-sec-arrow">{{ n.v ? '▾' : '▸' }}</span>{{ n.l }}
    </button>
    <div v-if="n.v" class="n-sec-body">
      <NodeView v-for="(c, i) in kids" :key="c.id ?? i" :node="c" />
    </div>
  </div>

  <!-- text -->
  <div v-else-if="n.k === 'text'" class="n-text" :class="tone" :style="textStyle">{{ n.l }}</div>
  <div v-else-if="n.k === 'note'" class="n-text n-wrap tone-dim">{{ n.l }}</div>
  <div v-else-if="n.k === 'warn'" class="n-text n-wrap tone-warn">{{ n.l }}</div>
  <hr v-else-if="n.k === 'sep'" class="n-sep" />
  <div v-else-if="n.k === 'spacer'" :style="{ height: (p.h ?? 6) + 'px' }" />

  <!-- buttons -->
  <WButton
    v-else-if="n.k === 'button'"
    :label="n.l"
    :variant="p.def ? 'default' : 'push'"
    :disabled="p.enabled === false"
    :data-id="n.id"
    class="n-btn"
    @click="click(n.id)"
  />
  <WButton
    v-else-if="n.k === 'tool'"
    :label="n.l"
    :active="!!n.v"
    :data-id="n.id"
    class="n-btn"
    @click="click(n.id)"
  />

  <WCheckBox
    v-else-if="n.k === 'check'"
    :model-value="!!val"
    :label="n.l"
    :data-id="n.id"
    class="n-check"
    @update:model-value="(v: any) => set(n.id, !!v)"
  />
  <WRadioButton
    v-else-if="n.k === 'radio'"
    :model-value="n.v ? 1 : 0"
    :value="1"
    :label="n.l"
    :data-id="n.id"
    class="n-check"
    @change="click(n.id)"
  />

  <!-- label-left controls -->
  <div v-else-if="n.k === 'slider'" class="n-field" :data-id="n.id">
    <span class="n-label">{{ n.l }}</span>
    <WTrackbar
      class="n-grow"
      :model-value="val"
      :min="p.min"
      :max="p.max"
      :step="sliderStep"
      :ticks="false"
      @update:model-value="(v: number) => set(n.id, p.int ? Math.round(v) : v)"
    />
    <span v-if="!p.field" class="n-value">{{ fmtNum(val, p.fmt) }}</span>
    <WEdit
      v-else
      class="n-num"
      type="number"
      :model-value="fmtNum(val, p.fmt)"
      width="64px"
      @update:model-value="(v: string) => { const f = parseFloat(v); if (!isNaN(f)) set(n.id, f) }"
    />
  </div>

  <div v-else-if="n.k === 'input'" class="n-field" :data-id="n.id">
    <span class="n-label">{{ n.l }}</span>
    <WUpDown
      class="n-grow"
      :model-value="val"
      :min="-1e9"
      :max="1e9"
      :step="p.step || 1"
      @update:model-value="(v: number) => set(n.id, v)"
    />
  </div>

  <div v-else-if="n.k === 'edit'" class="n-field" :data-id="n.id">
    <span class="n-label">{{ n.l }}</span>
    <WEdit
      class="n-grow"
      :type="p.password ? 'password' : 'text'"
      :model-value="val"
      :placeholder="p.hint"
      @update:model-value="(v: string) => set(n.id, v)"
    />
  </div>

  <div v-else-if="n.k === 'combo'" class="n-field" :data-id="n.id">
    <span class="n-label">{{ n.l }}</span>
    <WComboBox
      class="n-grow"
      width="100%"
      :model-value="val"
      :options="comboOptions"
      @update:model-value="(v: any) => set(n.id, Number(v))"
    />
  </div>

  <!-- property sheet tabs -->
  <div v-else-if="n.k === 'tabs'" class="n-tabs" :data-id="n.id">
    <button
      v-for="(t, i) in p.items"
      :key="i"
      class="n-tab"
      :class="{ active: i === val }"
      @click="set(n.id, i)"
    >{{ t }}</button>
  </div>

  <div v-else-if="n.k === 'subtabs'" class="n-subtabs" :data-id="n.id">
    <div class="n-seg">
      <button
        v-for="(t, i) in p.items"
        :key="i"
        class="n-segbtn"
        :class="{ active: i === val }"
        @click="set(n.id, i)"
      >
        <span v-if="t.color" class="n-chip" :style="{ background: t.color }" />{{ t.l }}<span v-if="t.count" class="n-count"> {{ t.count }}</span>
      </button>
    </div>
    <span v-if="n.l" class="n-text tone-dim n-subnote">{{ n.l }}</span>
  </div>

  <!-- data -->
  <div v-else-if="n.k === 'table'" class="n-table-wrap w-bevel-sunken" :style="{ maxHeight: ((p.height ?? 8) * 19 + 28) + 'px' }" :data-id="n.id">
    <table class="n-table">
      <thead>
        <tr><th v-for="(h, i) in p.headers" :key="i">{{ h }}</th></tr>
      </thead>
      <tbody>
        <tr
          v-for="(r, ri) in p.rows"
          :key="ri"
          :class="{ sel: ri === val }"
          @click="set(n.id, ri)"
        >
          <td v-for="(c, ci) in r" :key="ci" :class="cellStyle(c)">
            <span v-if="c.color" class="n-chip" :style="{ background: c.color }" />{{ c.t }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>

  <div v-else-if="n.k === 'kv'" class="n-kv">
    <template v-for="(r, i) in p.rows" :key="i">
      <span class="n-kv-k">{{ r[0] }}</span><span class="n-kv-v">{{ r[1] }}</span>
    </template>
  </div>

  <span v-else-if="n.k === 'badge'" class="n-badge" :class="['lv-' + p.level, { big: p.big }]">{{ n.l }}</span>
  <span v-else-if="n.k === 'swatch'" class="n-swatch w-bevel-sunken-thin" :style="{ background: p.color }" />

  <div v-else-if="n.k === 'hist'" class="n-hist w-bevel-sunken" :style="{ height: (p.height ?? 80) + 'px' }">
    <svg viewBox="0 0 256 100" preserveAspectRatio="none" width="100%" height="100%">
      <rect
        v-for="(b, i) in p.bins"
        :key="i"
        :x="i" :y="100 - Math.min(100, (b === 0 ? 0 : Math.log10(b + 1) / Math.log10(Math.max(...p.bins) + 1) * 100))"
        width="1" :height="Math.min(100, (b === 0 ? 0 : Math.log10(b + 1) / Math.log10(Math.max(...p.bins) + 1) * 100))"
        fill="#000080"
      />
      <rect v-if="p.mark >= 0" :x="p.mark" y="0" width="1" height="100" fill="#ff0000" />
    </svg>
  </div>

  <!-- kinds added as render/kinds/<Kind>.vue -->
  <component :is="customKinds[n.k]" v-else-if="customKinds[n.k]" :node="n" />

  <!-- unknown kinds stay visible instead of vanishing -->
  <div v-else class="n-text tone-warn">⟨{{ n.k }}⟩</div>
</template>

<script lang="ts">
export default { name: 'NodeView' }
</script>
