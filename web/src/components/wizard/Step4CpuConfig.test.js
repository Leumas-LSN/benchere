import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import Step4CpuConfig from './Step4CpuConfig.vue'
import en from '../../i18n/en.js'
import fr from '../../i18n/fr.js'
import { useWizardStore } from '../../stores/wizard.js'

function makeApp() {
  const app = createApp({ render() { return h(Step4CpuConfig) } })
  const pinia = createPinia()
  app.use(pinia)
  setActivePinia(pinia)
  const i18n = createI18n({ legacy: false, locale: 'en', fallbackLocale: 'en', messages: { en, fr } })
  app.use(i18n)
  return app
}

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('Step4CpuConfig', () => {
  it('renders all 7 hardcoded stressor pills', async () => {
    const app = makeApp()
    const html = await renderToString(app)
    // The whitelist is fixed in the component, so the user cannot inject
    // a stressor name through the wizard surface. Check each appears.
    const stressors = ['cpu', 'vm', 'io', 'hdd', 'matrix', 'cache', 'pipe']
    for (const s of stressors) {
      expect(html).toContain('>' + s + '<')
    }
  })

  it('marks default cpu stressor as active when set', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.cpuConfig.stressors = ['cpu']
    const html = await renderToString(app)
    // The active cpu pill carries is-active.
    const cpuMatch = html.match(/<button[^>]*class="stressor-pill is-active"[^>]*>[^<]*<svg[^>]*>[^<]*<polyline[^/]*\/><\/svg><span class="num">cpu/) || html.match(/stressor-pill is-active[^"]*"[^>]*>(?:[^<]|<(?!button))*?cpu/)
    expect(cpuMatch).toBeTruthy()
  })

  it('renders timeout and stressor workers inputs', async () => {
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/Duration|Duree/i)
    expect(html).toMatch(/stress-ng workers|Workers stress-ng/i)
  })
})
