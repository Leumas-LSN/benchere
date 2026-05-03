import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import Step1Type from './Step1Type.vue'
import en from '../../i18n/en.js'
import fr from '../../i18n/fr.js'
import { useWizardStore } from '../../stores/wizard.js'

function makeApp() {
  const app = createApp({ render() { return h(Step1Type) } })
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

describe('Step1Type', () => {
  it('renders the three mode cards', async () => {
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/Storage|Stockage/)
    expect(html).toMatch(/CPU/)
    expect(html).toMatch(/Mixed|Mixte/)
  })

  it('marks the selected mode card', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.type = 'storage'
    const html = await renderToString(app)
    expect(html).toContain('is-selected')
  })

  it('renders the step eyebrow with current/total', async () => {
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/STEPS|ETAPES/)
  })
})
