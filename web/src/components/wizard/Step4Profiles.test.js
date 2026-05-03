import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import Step4Profiles from './Step4Profiles.vue'
import en from '../../i18n/en.js'
import fr from '../../i18n/fr.js'

vi.mock('../../api/client.js', () => ({
  api: {
    listProfiles: vi.fn(),
  },
}))

import { api } from '../../api/client.js'

function makeApp() {
  const app = createApp({ render() { return h(Step4Profiles) } })
  const pinia = createPinia()
  app.use(pinia)
  setActivePinia(pinia)
  const i18n = createI18n({ legacy: false, locale: 'en', fallbackLocale: 'en', messages: { en, fr } })
  app.use(i18n)
  return app
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('Step4Profiles', () => {
  it('renders all five filter pills', async () => {
    api.listProfiles.mockResolvedValue([])
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/All|Tous/)
    expect(html).toMatch(/Random/)
    expect(html).toMatch(/Sequential|Sequentiel/)
    expect(html).toMatch(/Mixed|Mixte/)
    expect(html).toMatch(/Custom|Personnalises/)
  })

  it('renders the search input', async () => {
    api.listProfiles.mockResolvedValue([])
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/search-input/)
  })

  it('renders the title and subtitle', async () => {
    api.listProfiles.mockResolvedValue([])
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/profiles|Profiles|Profils/)
  })
})

describe('ProfileCard inline rendering', () => {
  it('does not render any cards initially while loading', async () => {
    api.listProfiles.mockResolvedValue([])
    const app = makeApp()
    const html = await renderToString(app)
    // SSR runs before onMounted fetches, so the loading copy appears.
    // The test confirms the empty/loading branch is taken (no profile cards
    // before data arrives), which is the relevant guarantee here.
    expect(html).toMatch(/Loading|Chargement/i)
    expect(html).not.toMatch(/profile-card/)
  })
})
