import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import GravityHero from './components/GravityHero.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('GravityHero', GravityHero)
  },
} satisfies Theme
