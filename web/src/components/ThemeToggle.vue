<template>
  <button
    type="button"
    class="btn-ghost btn-sm h-9 w-9 px-0 rounded-lg"
    :aria-label="theme.theme === 'dark' ? 'Activer le mode clair' : 'Activer le mode sombre'"
    :title="theme.theme === 'dark' ? 'Mode clair' : 'Mode sombre'"
    @click="theme.toggle()"
  >
    <Icon :name="theme.theme === 'dark' ? 'sun' : 'moon'" :size="18" />
  </button>
</template>

<script setup>
import { useThemeStore } from '../stores/theme.js'
import Icon from './Icon.vue'

const theme = useThemeStore()
</script>
