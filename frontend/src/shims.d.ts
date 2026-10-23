// The library build ships no .d.ts; these loose declarations keep vue-tsc quiet.
declare module '@win32-vue/win32' {
  import type { Component, Plugin } from 'vue'
  const plugin: Plugin
  export default plugin
  export const WButton: Component
  export const WCheckBox: Component
  export const WRadioButton: Component
  export const WTrackbar: Component
  export const WEdit: Component
  export const WComboBox: Component
  export const WGroupBox: Component
  export const WUpDown: Component
  export const WTitleBar: Component
  export const WMenuBar: Component
  export const WStatusBar: Component
  export const WStatusPane: Component
  export const WDialog: Component
  export const WListView: Component
  export interface WMenuItemModel {
    id?: string
    label?: string
    accelerator?: string
    disabled?: boolean
    checked?: boolean
    separator?: boolean
    children?: WMenuItemModel[]
    action?: () => void
  }
}
declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const c: DefineComponent<{}, {}, any>
  export default c
}
