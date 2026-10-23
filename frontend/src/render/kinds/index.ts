import type { Component } from 'vue'

// Every render/kinds/<Name>.vue renders the node kind <name> (lower case). A panel that
// needs a control the generic renderer lacks adds a file here and records
// ui.Node{K: "<name>"} on the Go side; nothing else has to change.
const mods = import.meta.glob('./*.vue', { eager: true }) as Record<string, { default: Component }>
export const customKinds: Record<string, Component> = {}
for (const [path, m] of Object.entries(mods)) {
  const name = path.replace('./', '').replace('.vue', '').toLowerCase()
  customKinds[name] = m.default
}
