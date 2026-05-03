import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import Step3Pool from './Step3Pool.vue'
import en from '../../i18n/en.js'
import fr from '../../i18n/fr.js'

function makeApp() {
  const app = createApp({ render() { return h(Step3Pool) } })
  const pinia = createPinia()
  app.use(pinia)
  setActivePinia(pinia)
  const i18n = createI18n({ legacy: false, locale: 'en', fallbackLocale: 'en', messages: { en, fr } })
  app.use(i18n)
  return app
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.unstubAllGlobals()
})

describe('Step3Pool', () => {
  it('renders the title', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    }))
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/pool|Pool/i)
  })
})
