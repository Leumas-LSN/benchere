import { describe, it, expect } from 'vitest'
import { createApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import NodeCardHorizontal from './NodeCardHorizontal.vue'

function render(props) {
  const app = createApp({ render() { return h(NodeCardHorizontal, props) } })
  return renderToString(app)
}

describe('NodeCardHorizontal', () => {
  it('renders the node name and the NODE tag', async () => {
    const html = await render({ name: 'aqua-pve-01' })
    expect(html).toContain('aqua-pve-01')
    expect(html).toContain('NODE')
  })

  it('renders cpu, ram and load values', async () => {
    const html = await render({
      name: 'pve-1',
      cpu: 42.5,
      ram: 78,
      load: 1.23,
    })
    expect(html).toContain('42.5')
    expect(html).toContain('78')
    expect(html).toContain('1.23')
    expect(html).toContain('cpu')
    expect(html).toContain('ram')
    expect(html).toContain('load')
  })

  it('uses red stroke when cpu exceeds 80', async () => {
    const html = await render({
      name: 'pve-hot',
      cpu: 90,
      ram: 60,
      load: 4,
      history: [80, 85, 90, 92, 95],
    })
    expect(html).toContain('#ef4444')
  })

  it('uses amber stroke when cpu is between 60 and 80', async () => {
    const html = await render({
      name: 'pve-warm',
      cpu: 70,
      ram: 60,
      load: 2,
      history: [60, 65, 68, 70],
    })
    expect(html).toContain('#f59e0b')
  })

  it('uses green stroke when cpu is below 60', async () => {
    const html = await render({
      name: 'pve-cool',
      cpu: 30,
      ram: 40,
      load: 0.5,
      history: [25, 28, 30, 32],
    })
    expect(html).toContain('#10b981')
  })

  it('renders no polyline when history has fewer than 2 samples', async () => {
    const html = await render({
      name: 'pve-empty',
      cpu: 10,
      history: [],
    })
    const polylineCount = (html.match(/<polyline/g) || []).length
    expect(polylineCount).toBe(0)
  })

  it('renders a status pill when status is provided', async () => {
    const html = await render({ name: 'pve-1', status: 'online' })
    expect(html).toContain('online')
    expect(html).toContain('nch-status')
  })
})
