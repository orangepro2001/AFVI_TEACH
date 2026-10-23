// Thin wrappers over the Wails window runtime; no-ops in a plain browser (dev/tests).
type RT = {
  WindowMinimise?: () => void
  WindowToggleMaximise?: () => void
  Quit?: () => void
}
const rt = (): RT => ((window as any).runtime ?? {}) as RT

export const minimise = () => rt().WindowMinimise?.()
export const toggleMaximise = () => rt().WindowToggleMaximise?.()
export const quit = () => (rt().Quit ? rt().Quit!() : window.close())
export const inWails = () => !!(window as any).runtime
