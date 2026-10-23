# @win32-vue/win32

Retro Win32 / industrial-control UI components for **Vue 3** — built for
**Wails** desktop apps (Go + Wails + Vue 3).

Windows 2000 classic palette, real 3D bevels, tight control metrics,
zero rounded corners, zero soft shadows.

```bash
npm install @win32-vue/win32
```

```ts
import { createApp } from 'vue'
import Win32 from '@win32-vue/win32'
import '@win32-vue/win32/styles/win32.css'

createApp(App).use(Win32).mount('#app')
```

Individual components can also be imported directly:

```ts
import { WButton, WListView, WTitleBar } from '@win32-vue/win32'
```

## Styles

| Entry point | Contents |
|---|---|
| `@win32-vue/win32/styles/win32.css` | everything (tokens + base + bevel) |
| `@win32-vue/win32/styles/tokens.css` | colour palette & metrics only |
| `@win32-vue/win32/styles/base.css` | reset, typography, scrollbars |
| `@win32-vue/win32/styles/bevel.css` | raised / sunken / etched primitives |

## Theming

Every colour and metric is a CSS custom property on `:root`:

```css
:root {
  --w-face: #d4d0c8;      /* COLOR_3DFACE   */
  --w-select-bg: #0a246a; /* COLOR_HIGHLIGHT */
  --w-font-ui: Tahoma, "MS Sans Serif", sans-serif;
  --w-btn-h: 23px;
}
```

## Wails integration

For a frameless Wails window, the in-app `WTitleBar` becomes the real caption bar:

```go
wails.Run(&options.App{Frameless: true, DisableResize: false})
```

```vue
<WTitleBar title="HMI" draggable @close="runtime.Quit()" />
```

`draggable` sets `--wails-draggable: drag`, which the Wails runtime uses to
drag the window. Caption buttons are automatically excluded.

## MIT
