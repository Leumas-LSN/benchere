import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client.js'

export const useJobsStore = defineStore('jobs', () => {
  const jobs = ref([])
  const loading = ref(false)
  const error = ref(null)

  async function fetchJobs() {
    loading.value = true
    error.value = null
    try {
      jobs.value = (await api.listJobs()) ?? []
    } catch (err) {
      jobs.value = []
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createJob(config) {
    const result = await api.createJob(config)
    return result.id
  }

  async function cancelJob(id) {
    await api.cancelJob(id)
  }

  async function getJob(id) {
    return api.getJob(id)
  }

  async function listWorkers(id) {
    return api.listWorkers(id)
  }

  return { jobs, loading, error, fetchJobs, createJob, cancelJob, getJob, listWorkers }
})
