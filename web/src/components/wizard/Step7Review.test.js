import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import Step7Review from './Step7Review.vue'
import en from '../../i18n/en.js'
import fr from '../../i18n/fr.js'
import { useWizardStore } from '../../stores/wizard.js'

function makeApp() {
  const app = createApp({ render() { return h(Step7Review) } })
  const pinia = createPinia()
  app.use(pinia)
  setActivePinia(pinia)
  const i18n = createI18n({ legacy: false, locale: 'en', fallbackLocale: 'en', messages: { en, fr } })
  app.use(i18n)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div/>' } }, { path: '/dashboard/:id', component: { template: '<div/>' } }],
  })
  app.use(router)
  return app
}

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('Step7Review', () => {
  it('renders the launch button', async () => {
    const app = makeApp()
    const html = await renderToString(app)
    expect(html).toMatch(/Launch job|Lancer le job/i)
  })

  it('shows wallclock and bytes from store estimate', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.type = 'storage'
    store.estimate = { wallclockSec: 720, bytesWritten: 64 * 1024 * 1024 * 1024 }
    const html = await renderToString(app)
    expect(html).toContain('~12m')
    expect(html).toContain('~64')
  })

  it('hides workload section in cpu mode', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.type = 'cpu'
    const html = await renderToString(app)
    expect(html).not.toMatch(/Workload/)
  })

  it('hides cpu section in pure storage mode', async () => {
    const app = makeApp()
    const store = useWizardStore()
    store.type = 'storage'
    const html = await renderToString(app)
    expect(html).not.toMatch(/CPU configuration|Configuration CPU/)
  })
})
