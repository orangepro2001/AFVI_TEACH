/** Shared public types of the win32-vue component library. */

export type WComboOption = {
  label: string
  value: string | number
  disabled?: boolean
  group?: string
}

export type WListViewColumn = {
  key: string
  title: string
  width?: number | string
  align?: 'left' | 'center' | 'right'
  sortable?: boolean
}

export type WListViewRow = Record<string, string | number>

export type WTreeNode = {
  id: string
  label: string
  children?: WTreeNode[]
  icon?: string
  disabled?: boolean
  data?: unknown
}

export type WGaugeZone = {
  from: number
  to: number
  color: string
}

export type WTrendSeries = {
  name: string
  color: string
  points: number[]
}

export type WMenuItemModel = {
  id?: string
  label?: string
  accelerator?: string
  icon?: string
  disabled?: boolean
  checked?: boolean
  radio?: boolean
  separator?: boolean
  children?: WMenuItemModel[]
  action?: () => void
}
