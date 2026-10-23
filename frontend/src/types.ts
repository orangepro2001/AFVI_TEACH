export interface UiNode {
  k: string
  id?: string
  l?: string
  v?: any
  p?: Record<string, any>
  c?: UiNode[]
}

export interface Shape {
  t: string
  x: number
  y: number
  w?: number
  h?: number
  x2?: number
  y2?: number
  r?: number
  pts?: [number, number][]
  col?: string
  fill?: string
  th?: number
  dash?: boolean
  text?: string
}

export interface ViewerOut {
  rev: number
  w: number
  h: number
  gray?: boolean
  shapes: Shape[]
  fitSeq: number
  zoomOn?: [number, number, number]
  zoomOnSeq: number
  noDblFit?: boolean
  claim?: boolean
  cursor?: string
  hover?: string
}

export interface Frame {
  seq: number
  title?: string
  slots: Record<string, UiNode | null>
  viewer?: ViewerOut
  meta?: Record<string, any>
}

export interface Cmd {
  id: string
  op: string
  v?: any
  x?: number
  y?: number
  k?: string
  ctrl?: boolean
  shift?: boolean
}
