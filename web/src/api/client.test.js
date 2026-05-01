import { describe, it, expect, vi, beforeEach } from 'vitest'
import { api } from './client.js'

function mockFetch(status, body) {
  return vi.fn().mockResolvedValue({
    ok: status < 400,
    status,
    text: () => Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
    json: () => Promise.resolve(body),
  })
}

beforeEach(() => {
  vi.unstubAllGlobals()
})

describe('api.getSettings', () => {
  it('calls GET /api/settings and returns JSON', async () => {
    vi.stubGlobal('fetch', mockFetch(200, { proxmox_url: 'https://pve:8006' }))
    const result = await api.getSettings()
    expect(result.proxmox_url).toBe('https://pve:8006')
    expect(fetch).toHaveBeenCalledWith('/api/settings', expect.objectContaining({ method: 'GET' }))
  })
})

describe('api.createJob', () => {
  it('posts config and returns id', async () => {
    vi.stubGlobal('fetch', mockFetch(201, { id: 'abc-123' }))
    const result = await api.createJob({ name: 'Test', mode: 'storage' })
    expect(result.id).toBe('abc-123')
    expect(fetch).toHaveBeenCalledWith('/api/jobs', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'Test', mode: 'storage' }),
    }))
  })
})

describe('api.cancelJob', () => {
  it('returns null on 204', async () => {
    vi.stubGlobal('fetch', mockFetch(204, null))
    const result = await api.cancelJob('job-1')
    expect(result).toBeNull()
  })
})

describe('api.saveSettings', () => {
  it('posts settings and returns null on 204', async () => {
    vi.stubGlobal('fetch', mockFetch(204, null))
    const result = await api.saveSettings({ proxmox_url: 'https://pve' })
    expect(result).toBeNull()
  })
})

describe('api error handling', () => {
  it('throws with status code on non-ok response', async () => {
    vi.stubGlobal('fetch', mockFetch(400, 'bad request'))
    await expect(api.createJob({})).rejects.toThrow('400')
  })
})

describe('api.reportPdfUrl', () => {
  it('returns correct URL without fetch', () => {
    expect(api.reportPdfUrl('job-42')).toBe('/api/jobs/job-42/report.pdf?lang=fr')
  })
})

describe('api.testProxmox', () => {
  it('sends POST with no body and no Content-Type', async () => {
    vi.stubGlobal('fetch', mockFetch(200, { ok: true, nodes: [] }))
    const result = await api.testProxmox()
    expect(result.ok).toBe(true)
    const callOpts = fetch.mock.calls[0][1]
    expect(callOpts.body).toBeUndefined()
    expect(callOpts.headers['Content-Type']).toBeUndefined()
  })
})

describe('api.deleteProfile', () => {
  it('sends DELETE to correct URL', async () => {
    vi.stubGlobal('fetch', mockFetch(204, null))
    const result = await api.deleteProfile('prof-1')
    expect(result).toBeNull()
    expect(fetch).toHaveBeenCalledWith('/api/profiles/prof-1', expect.objectContaining({ method: 'DELETE' }))
  })
})

describe('api network failure', () => {
  it('propagates fetch rejection', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    await expect(api.listJobs()).rejects.toThrow('Failed to fetch')
  })
})
