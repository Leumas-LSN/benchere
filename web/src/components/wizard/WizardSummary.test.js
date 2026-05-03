import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import WizardSummary from './WizardSummary.vue'
import en from '../../i18n/en.js'
import fr from '../../i18n/fr.js'
import { useWizardStore } from '../../stores/wizard.js'

function makeApp() {
  const app = createApp({
    render() { return h(WizardSummary) },
  })
  const pinia = createPinia()
  app.use(pinia)
  setActivePinia(pinia)
  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    fallbackLocale: 'en',
    messages: { en, fr },
  })
  app.use(i18n)
  return app
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.unstubAllGlobals()
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({ wallclock_sec: 0, bytes_written: 0 }),
  }))
})

describe('WizardSummary', () => {
  it('renders SUMMARY and ESTIMATED COST sections', async () => {
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/SUMMARY|RESUME/)
    expect(html).toMatch(/ESTIMATED COST|COUT ESTIME/)
  })

  it('formats wallclock as ~Nm when below 60 minutes', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.estimate = { wallclockSec: 720, bytesWritten: 0 } // 12 min
    const html = await renderToString(app)
    expect(html).toContain('~12m')
  })

  it('formats wallclock as ~Nh Mm when above 60 minutes', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.estimate = { wallclockSec: 3900, bytesWritten: 0 } // 65 min = 1h 5m
    const html = await renderToString(app)
    expect(html).toContain('~1h')
    expect(html).toContain('5m')
  })

  it('formats bytes_written as GB', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.estimate = { wallclockSec: 0, bytesWritten: 64 * 1024 * 1024 * 1024 }
    const html = await renderToString(app)
    expect(html).toContain('64')
  })

  it('shows default verdict label when thresholds not customized', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.range.thresholdsCustom = false
    const html = await renderToString(app)
    expect(html).toMatch(/auto . default thresholds|auto . seuils par defaut/i)
  })

  it('shows pool and cluster from store', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.cluster = 'aqua-prod'
    store.pool = 'ceph-rbd'
    const html = await renderToString(app)
    expect(html).toContain('aqua-prod')
    expect(html).toContain('ceph-rbd')
  })
})
