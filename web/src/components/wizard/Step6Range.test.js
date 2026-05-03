import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import Step6Range from './Step6Range.vue'
import en from '../../i18n/en.js'
import fr from '../../i18n/fr.js'
import { useWizardStore } from '../../stores/wizard.js'

function makeApp() {
  const app = createApp({ render() { return h(Step6Range) } })
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

describe('Step6Range', () => {
  it('renders runtime, ramp and io depth inputs', async () => {
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/Runtime/i)
    expect(html).toMatch(/Ramp/i)
    expect(html).toMatch(/IO depth/i)
  })

  it('does not render custom thresholds when auto mode active', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.range.thresholdsCustom = false
    const html = await renderToString(app)
    expect(html).not.toMatch(/Min read IOPS|IOPS read minimum/i)
  })

  it('renders custom threshold inputs when toggled on', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.range.thresholdsCustom = true
    const html = await renderToString(app)
    expect(html).toMatch(/Min read IOPS|IOPS read minimum/i)
    expect(html).toMatch(/Max p99 latency|Latence p99 max/i)
  })
})
