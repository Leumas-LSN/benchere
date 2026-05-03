import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import ProfileCard from './ProfileCard.vue'
import en from '../../i18n/en.js'
import fr from '../../i18n/fr.js'

function makeApp(props) {
  const app = createApp({ render() { return h(ProfileCard, props) } })
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

describe('ProfileCard', () => {
  it('renders the name', async () => {
    const app = makeApp({ name: 'oltp-4k-70-30', bs: '4k', rw: 'randread', estMinutes: 5, popular: true, builtIn: true, selected: false })
    const html = await renderToString(app)
    expect(html).toContain('oltp-4k-70-30')
  })

  it('renders the meta line with bs, rw and estMinutes', async () => {
    const app = makeApp({ name: 'sql-8k', bs: '8k', rw: 'randrw', estMinutes: 7, popular: false, builtIn: true, selected: false })
    const html = await renderToString(app)
    expect(html).toContain('bs=8k')
    expect(html).toContain('randrw')
    expect(html).toContain('~7m')
  })

  it('shows POPULAR pill only when popular AND builtIn', async () => {
    const popular = await renderToString(makeApp({ name: 'a', popular: true, builtIn: true, selected: false }))
    expect(popular).toMatch(/POPULAR|POPULAIRE/)

    const notPopular = await renderToString(makeApp({ name: 'a', popular: false, builtIn: true, selected: false }))
    expect(notPopular).not.toMatch(/POPULAR|POPULAIRE/)

    const popularButCustom = await renderToString(makeApp({ name: 'a', popular: true, builtIn: false, selected: false }))
    expect(popularButCustom).not.toMatch(/POPULAR|POPULAIRE/)
  })

  it('shows CUSTOM pill only when not builtIn', async () => {
    const custom = await renderToString(makeApp({ name: 'a', builtIn: false, selected: false }))
    expect(custom).toMatch(/CUSTOM/)

    const builtIn = await renderToString(makeApp({ name: 'a', builtIn: true, selected: false }))
    expect(builtIn).not.toMatch(/CUSTOM/)
  })

  it('marks the card with is-selected when selected', async () => {
    const html = await renderToString(makeApp({ name: 'a', selected: true }))
    expect(html).toContain('is-selected')
  })
})
