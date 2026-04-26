import { defineStore } from 'pinia'
import { api } from '../api/client.js'

export const useSettingsStore = defineStore('settings', () => {
  async function load() {
    return api.getSettings()
  }

  async function save(settings) {
    return api.saveSettings(settings)
  }

  async function scanStorages() {
    return api.scanStorages()
  }

  async function scanBridges() {
    return api.scanBridges()
  }

  async function test() {
    return api.testProxmox()
  }

  return { load, save, test, scanStorages, scanBridges }
})
