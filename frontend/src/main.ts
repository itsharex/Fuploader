import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

// Import Element Plus overrides BEFORE the main CSS
import './styles/element-override.css'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores'

// 导入样式
import './styles/variables.css'
import './styles/global.css'
import './styles/theme.css'

const app = createApp(App)

// 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

const pinia = createPinia()
app.use(pinia)
app.use(router)
app.use(ElementPlus)

// 初始化主题
const themeStore = useThemeStore()
themeStore.initTheme()

app.mount('#app')
