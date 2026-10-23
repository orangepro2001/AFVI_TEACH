<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { WTitleBar, WMenuBar, WStatusBar, WStatusPane, WDialog } from '@win32-vue/win32'
import type { WMenuItemModel } from '@win32-vue/win32'
import { frame, send, connected } from './bridge'
import type { UiNode } from './types'
import NodeView from './render/NodeView.vue'
import Viewer from './render/Viewer.vue'
import Palette from './render/Palette.vue'
import { minimise, toggleMaximise, quit, inWails } from './window'

const slots = computed(() => frame.value?.slots ?? {})
const meta = computed(() => frame.value?.meta ?? {})
const title = computed(() => frame.value?.title ?? 'AFVI')
const dialogs = computed<UiNode[]>(() => slots.value.dialogs?.c ?? [])

function menuModel(root: UiNode | null | undefined): WMenuItemModel[] {
  return (root?.c ?? []).map((m) => ({
    id: m.id,
    label: m.l,
    children: (m.c ?? []).map((i) =>
      i.k === 'msep'
        ? { separator: true }
        : {
            id: i.id,
            label: i.l,
            accelerator: i.p?.shortcut,
            checked: i.v === true,
            disabled: i.p?.enabled === false,
          },
    ),
  }))
}
const menu = computed(() => menuModel(slots.value.menu))
function onMenu(item: WMenuItemModel) {
  if (item.id) send({ id: item.id, op: 'click' })
}

// theme / language / font come from the host so a flag or the menu changes them for all clients
watch(
  meta,
  (m) => {
    const d = document.documentElement
    d.dataset.theme = m.theme === 'dark' ? 'dark' : 'light'
    const px = Number(m.fontPx) || 16
    d.style.zoom = String(px / 12)
    d.lang = m.lang ?? 'zh'
  },
  { immediate: true },
)
watch(title, (t) => (document.title = t), { immediate: true })

const KEYS = new Set(['F1', 'F5', 'F11', 'Escape', 'Backspace', 'Delete', 'Enter'])
function onKey(e: KeyboardEvent) {
  const typing = (e.target as HTMLElement)?.matches?.('input, textarea')
  let name = e.key
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'z') name = 'Ctrl+Z'
  else if (!KEYS.has(e.key)) return
  if (typing && !['F1', 'F5', 'F11', 'Escape'].includes(name)) return
  if (name.startsWith('F')) e.preventDefault()
  send({ id: 'key:' + name, op: 'key' })
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

const statusNode = computed(() => slots.value.status?.c?.[0])
const focusView = computed(() => !!meta.value.focusView)
</script>

<template>
  <div class="shell">
    <WTitleBar :title="title" icon="▣" :draggable="inWails()" :show-max="inWails()" :show-min="inWails()"
      @minimize="minimise" @maximize="toggleMaximise" @close="quit" @dblclick="toggleMaximise" />
    <WMenuBar v-if="menu.length" :model="menu" @select="onMenu" />
    <div v-if="slots.toolbar" class="toolbar-row"><NodeView :node="slots.toolbar" /></div>
    <div v-if="slots.tabs" class="tabs-row"><NodeView :node="slots.tabs" /></div>

    <div class="shell-body">
      <div v-if="slots.sub" class="sub-strip"><NodeView :node="slots.sub" /></div>
      <div class="shell-cols">
        <div v-if="slots.left && !focusView" class="col col-left w-bevel-sunken-thin" data-slot="left"><NodeView :node="slots.left" /></div>
        <div class="col-center">
          <div class="viewer-host w-bevel-sunken" data-slot="viewer">
            <Viewer :viewer="frame?.viewer" :background="meta.theme === 'dark' ? '#2b2b2b' : '#808080'" />
            <Palette v-if="slots.palette?.c?.length && !(meta.role === 0)" :node="slots.palette.c[0]" />
            <div v-if="slots.message?.c?.length" class="center-msg"><NodeView :node="slots.message" /></div>
          </div>
        </div>
        <div v-if="slots.right && !focusView" class="col col-right w-bevel-sunken-thin" data-slot="right"><NodeView :node="slots.right" /></div>
      </div>
    </div>

    <WStatusBar>
      <WStatusPane width="auto" class="status-main">{{ statusNode?.l ?? (connected ? '' : '…') }}</WStatusPane>
      <WStatusPane v-if="frame?.viewer?.hover" width="260px">{{ frame.viewer.hover }}</WStatusPane>
      <WStatusPane v-for="(s, i) in statusNode?.c ?? []" :key="i" :width="(s.p?.w ?? 100) + 'px'">{{ s.l }}</WStatusPane>
    </WStatusBar>

    <div v-for="d in dialogs" :key="d.id" class="dlg-layer">
      <WDialog :title="d.l" :width="d.p?.w ?? 420" modal :closable="d.p?.closeable !== false"
        @close="send({ id: 'close:' + d.id, op: 'click' })">
        <div class="dlg-body"><NodeView v-for="(c, i) in d.c ?? []" :key="c.id ?? i" :node="c" /></div>
      </WDialog>
    </div>
  </div>
</template>
