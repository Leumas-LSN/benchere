<template>
  <section
    class="rounded-lg border overflow-hidden"
    :style="{ borderColor: 'var(--border-subtle)', background: 'var(--surface-base)' }"
  >
    <button
      type="button"
      class="w-full flex items-center justify-between gap-3 px-3 py-1.5 text-left transition-colors hover:bg-ink-50 dark:hover:bg-ink-900"
      @click="toggle"
      :aria-expanded="expanded"
      aria-controls="live-logs-body"
    >
      <span class="flex items-center gap-2">
        <Icon
          :name="expanded ? 'chevron_down' : 'chevron_right'"
          :size="14"
          class="fg-muted"
        />
        <span class="text-xs font-semibold tracking-wide uppercase fg-secondary">
          Logs
        </span>
        <span class="pill num text-xs">{{ count }}</span>
      </span>
      <span class="flex items-center gap-3 text-xs fg-muted">
        <label
          v-if="expanded"
          class="flex items-center gap-1.5 cursor-pointer select-none"
          @click.stop
        >
          <input type="checkbox" v-model="autoScroll" class="form-check accent-brand-500" />
          <span>Auto-scroll</span>
        </label>
        <span v-if="!expanded && lastLine" class="font-mono truncate max-w-[28rem] hidden md:inline">
          {{ lastLine }}
        </span>
      </span>
    </button>

    <div
      v-show="expanded"
      id="live-logs-body"
      ref="bodyEl"
      class="border-t overflow-y-auto font-mono text-xs leading-5 px-3 py-2"
      :style="{ borderColor: 'var(--border-subtle)', height: '200px' }"
    >
      <div v-if="entries.length === 0" class="fg-muted italic py-4 text-center">
        En attente d'evenements...
      </div>
      <div
        v-for="(e, i) in entries"
        :key="i"
        class="whitespace-pre-wrap break-words"
        :class="toneClass(e.type)"
      >
        <span class="fg-muted">[{{ formatTime(e.t) }}]</span>
        <span class="ml-1.5">{{ e.line }}</span>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, ref, watch, onMounted, nextTick } from 'vue'
import { useWsStore } from '../stores/ws.js'
import Icon from './Icon.vue'

const ws = useWsStore()

const STORAGE_KEY = 'benchere.liveLogs.expanded'
const expanded = ref(false)
const autoScroll = ref(true)
const bodyEl = ref(null)

onMounted(() => {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === '1') expanded.value = true
  } catch (_) { /* localStorage may be blocked */ }
})

function toggle() {
  expanded.value = !expanded.value
  try { localStorage.setItem(STORAGE_KEY, expanded.value ? '1' : '0') } catch (_) {}
}

const entries = computed(() => ws.eventsNewestFirst)
const count   = computed(() => ws.eventCount)
const lastLine = computed(() => entries.value[0]?.line ?? '')

function formatTime(t) {
  if (!t) return ''
  const d = t instanceof Date ? t : new Date(t)
  return d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function toneClass(type) {
  switch (type) {
    case 'job_status':        return 'fg-primary'
    case 'provisioning_step': return 'text-amber-700 dark:text-amber-300'
    case 'elbencho_metric':   return 'fg-secondary'
    case 'proxmox_node':      return 'fg-muted'
    case 'proxmox_vm':        return 'fg-muted'
    default:                  return 'fg-secondary'
  }
}

// Newest-first list means scroll-to-top on new event when auto-scroll is on.
watch(count, async () => {
  if (!expanded.value || !autoScroll.value) return
  await nextTick()
  if (bodyEl.value) bodyEl.value.scrollTop = 0
})
</script>
