import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import WizardSidebar from './WizardSidebar.vue'
import en from '../../i18n/en.js'
import fr from '../../i18n/fr.js'
import { useWizardStore } from '../../stores/wizard.js'

function makeApp() {
  const app = createApp({
    render() { return h(WizardSidebar) },
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
})

describe('WizardSidebar', () => {
  it('renders the eyebrow with current and total step counts', async () => {
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/STEPS|ETAPES/)
    expect(html).toContain('1 /')
  })

  it('renders 7 step rows when storage type selected', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.type = 'storage'
    const html = await renderToString(app)
    const rows = html.match(/wizard-step-row/g) || []
    expect(rows.length).toBe(7)
  })

  it('shows step labels from i18n', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.type = 'storage'
    const html = await renderToString(app)
    // English locale loaded, step titles must appear
    expect(html).toMatch(/cluster|Cluster/i)
    expect(html).toMatch(/profile|Profile/i)
  })
})
