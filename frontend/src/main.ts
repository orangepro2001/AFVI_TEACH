import { createApp } from 'vue'
import Win32 from '@win32-vue/win32'
import './style.css'
import App from './App.vue'
import { start, injectFrame } from './bridge'

createApp(App).use(Win32).mount('#app')

// Test hook: ?mock lets a Playwright spec push frames without a Go backend.
if (new URLSearchParams(location.search).has('mock')) {
  ;(window as any).__inject = injectFrame
} else {
  start()
}
