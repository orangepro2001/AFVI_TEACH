import type { App, Plugin } from 'vue'

import WButton from './components/WButton.vue'
import WCheckBox from './components/WCheckBox.vue'
import WRadioButton from './components/WRadioButton.vue'
import WEdit from './components/WEdit.vue'
import WTextArea from './components/WTextArea.vue'
import WComboBox from './components/WComboBox.vue'
import WListBox from './components/WListBox.vue'
import WUpDown from './components/WUpDown.vue'
import WTrackbar from './components/WTrackbar.vue'
import WDateTime from './components/WDateTime.vue'
import WSearchBox from './components/WSearchBox.vue'

import WListView from './components/WListView.vue'
import WTreeView from './components/WTreeView.vue'
import WProgressBar from './components/WProgressBar.vue'
import WLabel from './components/WLabel.vue'
import WLink from './components/WLink.vue'
import WSeparator from './components/WSeparator.vue'
import WIcon from './components/WIcon.vue'
import WField from './components/WField.vue'
import WScrollBar from './components/WScrollBar.vue'

import WTitleBar from './components/WTitleBar.vue'
import WMenuBar from './components/WMenuBar.vue'
import WMenu from './components/WMenu.vue'
import WMenuItem from './components/WMenuItem.vue'
import WToolBar from './components/WToolBar.vue'
import WToolButton from './components/WToolButton.vue'
import WToolSeparator from './components/WToolSeparator.vue'
import WStatusBar from './components/WStatusBar.vue'
import WStatusPane from './components/WStatusPane.vue'
import WTabBar from './components/WTabBar.vue'
import WTab from './components/WTab.vue'
import WDialog from './components/WDialog.vue'
import WMessageBox from './components/WMessageBox.vue'
import WGroupBox from './components/WGroupBox.vue'
import WSplitter from './components/WSplitter.vue'
import WPanel from './components/WPanel.vue'

import WStatusLamp from './components/WStatusLamp.vue'
import WGauge from './components/WGauge.vue'
import WKnob from './components/WKnob.vue'
import WSegmentDisplay from './components/WSegmentDisplay.vue'
import WTrendChart from './components/WTrendChart.vue'
import WValve from './components/WValve.vue'

export {
  WButton,
  WCheckBox,
  WRadioButton,
  WEdit,
  WTextArea,
  WComboBox,
  WListBox,
  WUpDown,
  WTrackbar,
  WDateTime,
  WSearchBox,
  WListView,
  WTreeView,
  WProgressBar,
  WLabel,
  WLink,
  WSeparator,
  WIcon,
  WField,
  WScrollBar,
  WTitleBar,
  WMenuBar,
  WMenu,
  WMenuItem,
  WToolBar,
  WToolButton,
  WToolSeparator,
  WStatusBar,
  WStatusPane,
  WTabBar,
  WTab,
  WDialog,
  WMessageBox,
  WGroupBox,
  WSplitter,
  WPanel,
  WStatusLamp,
  WGauge,
  WKnob,
  WSegmentDisplay,
  WTrendChart,
  WValve,
}

export { splitMnemonic, stripMnemonic } from './utils/mnemonic'
export type { MnemonicChar } from './utils/mnemonic'
export type {
  WComboOption,
  WListViewColumn,
  WListViewRow,
  WTreeNode,
  WGaugeZone,
  WTrendSeries,
  WMenuItemModel,
} from './types'

/**
 * Optional global registration:
 *   import Win32 from '@win32-vue/win32'
 *   app.use(Win32)
 */
const components = {
  WButton,
  WCheckBox,
  WRadioButton,
  WEdit,
  WTextArea,
  WComboBox,
  WListBox,
  WUpDown,
  WTrackbar,
  WDateTime,
  WSearchBox,
  WListView,
  WTreeView,
  WProgressBar,
  WLabel,
  WLink,
  WSeparator,
  WIcon,
  WField,
  WScrollBar,
  WTitleBar,
  WMenuBar,
  WMenu,
  WMenuItem,
  WToolBar,
  WToolButton,
  WToolSeparator,
  WStatusBar,
  WStatusPane,
  WTabBar,
  WTab,
  WDialog,
  WMessageBox,
  WGroupBox,
  WSplitter,
  WPanel,
  WStatusLamp,
  WGauge,
  WKnob,
  WSegmentDisplay,
  WTrendChart,
  WValve,
}

export const Win32: Plugin = {
  install(app: App) {
    for (const [name, comp] of Object.entries(components)) {
      app.component(name, comp)
    }
  },
}

export default Win32
