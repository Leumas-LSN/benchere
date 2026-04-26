import { defineStore } from 'pinia'
import { ref, watchEffect } from 'vue'

const STORAGE_KEY = 'benchere-theme'

function detectInitial() {
  if (typeof window === 'undefined') return 'light'
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'dark' || stored === 'light') return stored
  } catch (_) {}
  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark'
  }
  return 'light'
}

export const useThemeStore = defineStore('theme', () => {
  const theme = ref(detectInitial())

  function apply(value) {
    if (typeof document === 'undefined') return
    const root = document.documentElement
    if (value === 'dark') root.classList.add('dark')
    else root.classList.remove('dark')
  }

  function set(value) {
    theme.value = value === 'dark' ? 'dark' : 'light'
  }

  function toggle() {
    set(theme.value === 'dark' ? 'light' : 'dark')
  }

  watchEffect(() => {
    apply(theme.value)
    try { localStorage.setItem(STORAGE_KEY, theme.value) } catch (_) {}
  })

  return { theme, set, toggle }
})
