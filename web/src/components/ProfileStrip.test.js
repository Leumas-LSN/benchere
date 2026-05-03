import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import ProfileStrip from './ProfileStrip.vue'
import en from '../i18n/en.js'
import fr from '../i18n/fr.js'
import { useWsStore } from '../stores/ws.js'

function makeApp(props) {
  const app = createApp({
    render() {
      return h(ProfileStrip, props)
    },
  })
  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    fallbackLocale: 'en',
    messages: { en, fr },
  })
  app.use(createPinia())
  setActivePinia(app._context.config.globalProperties.$pinia || createPinia())
  app.use(i18n)
  return app
}

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('ProfileStrip', () => {
  it('renders done tiles for each summary', async () => {
    const app = makeApp({
      phaseSummaries: [
        { profile_name: 'oltp-4k-70-30' },
        { profile_name: 'sql-8k-70-30' },
      ],
      currentProfile: '',
      totalProfiles: 0,
    })
    const html = await renderToString(app)
    expect(html).toContain('oltp-4k')
    expect(html).toContain('sql-8k-7')
    expect(html).toMatch(/Pass/)
    expect(html).toContain('2 done')
  })

  it('renders a running tile for the current profile with a fill bar', async () => {
    const app = makeApp({
      phaseSummaries: [{ profile_name: 'oltp-4k-70-30' }],
      currentProfile: 'vdi-4k-20-80',
      totalProfiles: 0,
    })
    const html = await renderToString(app)
    expect(html).toContain('vdi-4k-2')
    expect(html).toContain('ps-running')
    expect(html).toContain('ps-fill-track')
    expect(html).toContain('1 done')
    expect(html).toContain('1 running')
  })

  it('renders queued placeholders when totalProfiles is set', async () => {
    const app = makeApp({
      phaseSummaries: [{ profile_name: 'a' }],
      currentProfile: 'b',
      totalProfiles: 4,
    })
    const html = await renderToString(app)
    expect(html).toContain('ps-queued')
    expect(html).toContain('2 queued')
    expect(html).toContain('1/4')
  })

  it('renders only done counter and no running tile when no current profile', async () => {
    const app = makeApp({
      phaseSummaries: [{ profile_name: 'a' }, { profile_name: 'b' }],
      currentProfile: '',
      totalProfiles: 0,
    })
    const html = await renderToString(app)
    expect(html).not.toContain('ps-running')
    expect(html).toContain('2 done')
    expect(html).not.toContain('1 running')
    expect(html).toContain('2/2')
  })

  it('treats prefill phase as not-running', async () => {
    const app = makeApp({
      phaseSummaries: [],
      currentProfile: 'prefill',
      totalProfiles: 0,
    })
    const html = await renderToString(app)
    expect(html).not.toContain('ps-running')
    expect(html).toContain('0 done')
    expect(html).toContain('0/0')
  })

  it('shortens long profile names to 8 characters in the visible label', async () => {
    const app = makeApp({
      phaseSummaries: [{ profile_name: 'ceph-rand-4k-write-100' }],
      currentProfile: '',
      totalProfiles: 0,
    })
    const html = await renderToString(app)
    // The full name lives in the title attribute (tooltip), but the
    // visible name in the tile is truncated to 8 characters.
    expect(html).toContain('title="ceph-rand-4k-write-100"')
    const visibleMatch = html.match(/<span class="ps-tile-name"[^>]*>([^<]*)<\/span>/)
    expect(visibleMatch).not.toBeNull()
    expect(visibleMatch[1]).toBe('rand-4k-')
    expect(visibleMatch[1].length).toBeLessThanOrEqual(8)
  })
})
