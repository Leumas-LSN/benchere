import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useJobsStore } from './jobs.js'
import * as clientModule from '../api/client.js'

vi.mock('../api/client.js', () => ({
  api: {
    listJobs:   vi.fn(),
    createJob:  vi.fn(),
    cancelJob:  vi.fn(),
    getJob:     vi.fn(),
    listWorkers: vi.fn(),
  }
}))

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('fetchJobs', () => {
  it('populates jobs list', async () => {
    clientModule.api.listJobs.mockResolvedValue([{ id: '1', name: 'Job A', status: 'done' }])
    const store = useJobsStore()
    await store.fetchJobs()
    expect(store.jobs).toHaveLength(1)
    expect(store.jobs[0].name).toBe('Job A')
    expect(store.loading).toBe(false)
  })

  it('sets loading false even if API throws', async () => {
    clientModule.api.listJobs.mockRejectedValue(new Error('Network error'))
    const store = useJobsStore()
    await expect(store.fetchJobs()).rejects.toThrow()
    expect(store.loading).toBe(false)
  })
})

describe('createJob', () => {
  it('returns job id from API', async () => {
    clientModule.api.createJob.mockResolvedValue({ id: 'new-id' })
    const store = useJobsStore()
    const id = await store.createJob({ name: 'Test', mode: 'storage' })
    expect(id).toBe('new-id')
    expect(clientModule.api.createJob).toHaveBeenCalledWith({ name: 'Test', mode: 'storage' })
  })
})

describe('cancelJob', () => {
  it('calls api.cancelJob with correct id', async () => {
    clientModule.api.cancelJob.mockResolvedValue(null)
    const store = useJobsStore()
    await store.cancelJob('job-42')
    expect(clientModule.api.cancelJob).toHaveBeenCalledWith('job-42')
  })
})

describe('fetchJobs null guard', () => {
  it('treats null API response as empty list', async () => {
    clientModule.api.listJobs.mockResolvedValue(null)
    const store = useJobsStore()
    await store.fetchJobs()
    expect(store.jobs).toEqual([])
  })
})

describe('fetchJobs error state', () => {
  it('sets error message and clears jobs on API failure', async () => {
    clientModule.api.listJobs.mockRejectedValue(new Error('Network error'))
    const store = useJobsStore()
    store.jobs = [{ id: '1' }]
    await expect(store.fetchJobs()).rejects.toThrow()
    expect(store.jobs).toEqual([])
    expect(store.error).toBe('Network error')
  })
})
