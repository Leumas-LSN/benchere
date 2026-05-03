import { describe, it, expect } from 'vitest'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import LatencyTriPanel from './LatencyTriPanel.vue'
import en from '../i18n/en.js'
import fr from '../i18n/fr.js'

function makeApp(props) {
  const app = createApp({ render() { return h(LatencyTriPanel, props) } })
  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    fallbackLocale: 'en',
    messages: { en, fr },
  })
  app.use(i18n)
  return app
}

describe('LatencyTriPanel', () => {
  it('renders P50, P95 and P99 labels', async () => {
    const app = makeApp({
      p50History: [0.4, 0.5, 0.6],
      p95History: [1.0, 1.2, 1.5],
      p99History: [2.0, 2.5, 3.0],
      p50Current: 0.6,
      p95Current: 1.5,
      p99Current: 3.0,
    })
    const html = await renderToString(app)
    expect(html).toContain('P50')
    expect(html).toContain('P95')
    expect(html).toContain('P99')
  })

  it('renders the current values formatted to 2 decimals when below 100', async () => {
    const app = makeApp({
      p50History: [0.5],
      p95History: [1.5],
      p99History: [2.99],
      p50Current: 0.5,
      p95Current: 1.5,
      p99Current: 2.99,
    })
    const html = await renderToString(app)
    expect(html).toContain('0.50')
    expect(html).toContain('1.50')
    expect(html).toContain('2.99')
  })

  it('uses the canonical color stroke for each percentile', async () => {
    const app = makeApp({
      p50History: [1, 2, 3],
      p95History: [2, 3, 4],
      p99History: [3, 4, 5],
      p50Current: 3,
      p95Current: 4,
      p99Current: 5,
    })
    const html = await renderToString(app)
    expect(html).toContain('#22d3ee') // P50 cyan
    expect(html).toContain('#f59e0b') // P95 amber
    expect(html).toContain('#ef4444') // P99 red
  })

  it('renders three sparkline polylines when history has data', async () => {
    const app = makeApp({
      p50History: [1, 2, 3, 4, 5],
      p95History: [2, 3, 4, 5, 6],
      p99History: [3, 4, 5, 6, 7],
      p50Current: 5,
      p95Current: 6,
      p99Current: 7,
    })
    const html = await renderToString(app)
    const polylineCount = (html.match(/<polyline/g) || []).length
    expect(polylineCount).toBe(3)
  })

  it('renders the unit label in the header', async () => {
    const app = makeApp({
      p50Current: 0.5, p95Current: 1.5, p99Current: 2.5,
      unit: 'ms',
    })
    const html = await renderToString(app)
    expect(html).toContain('ms')
  })
})
