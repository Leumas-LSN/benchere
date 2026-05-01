<template>
  <section class="rounded-lg border overflow-hidden" :style="{ borderColor: 'var(--border-subtle)', background: 'var(--surface-base)' }">
    <button type="button" class="w-full flex items-center justify-between gap-3 px-3 py-1.5 text-left transition-colors hover:bg-ink-50 dark:hover:bg-ink-900" @click="toggle" :aria-expanded="expanded">
      <span class="flex items-center gap-2">
        <Icon :name="expanded ? 'chevron_down' : 'chevron_right'" :size="14" class="fg-muted" />
        <span class="text-xs font-semibold tracking-wide uppercase fg-secondary">Logs</span>
        <span class="pill num text-xs">{{ filteredEntries.length }}/{{ totalCount }}</span>
      </span>
      <span class="flex items-center gap-3 text-xs fg-muted" @click.stop>
        <select v-show="expanded" v-model="srcFilter" class="bg-transparent border border-default rounded px-1 py-0.5 text-xs">
          <option value="">all sources</option>
          <option value="orch">orch</option>
          <option value="ansible">ansible</option>
          <option value="fio">fio</option>
          <option value="elbencho">elbencho</option>
          <option value="system">system</option>
        </select>
        <select v-show="expanded" v-model="lvlFilter" class="bg-transparent border border-default rounded px-1 py-0.5 text-xs">
          <option value="">all levels</option>
          <option value="info">info</option>
          <option value="warn">warn</option>
          <option value="error">error</option>
        </select>
        <label v-show="expanded" class="flex items-center gap-1.5 cursor-pointer select-none">
          <input type="checkbox" v-model="autoScroll" class="form-check accent-brand-500" />
          <span>Auto-scroll</span>
        </label>
        <span v-if="!expanded && lastLine" class="font-mono truncate max-w-[28rem] hidden md:inline">{{ lastLine }}</span>
      </span>
    </button>
    <div v-show="expanded" ref="bodyEl" class="border-t overflow-y-auto font-mono text-xs leading-5 px-3 py-2" :style="{ borderColor: 'var(--border-subtle)', height: '240px' }">
      <div v-if="filteredEntries.length === 0" class="fg-muted italic py-4 text-center">No matching events.</div>
      <div v-for="(e, i) in filteredEntries" :key="i" class="whitespace-pre-wrap break-words" :class="toneClass(e)">
        <span class="fg-muted">[{{ formatTime(e.t) }}]</span>
        <span class="ml-1.5 fg-faint">{{ e.source }}/{{ e.level }}</span>
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
const srcFilter = ref('')
const lvlFilter = ref('')

onMounted(() => {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === '1') expanded.value = true
  } catch (_) {}
})
function toggle() {
  expanded.value = !expanded.value
  try { localStorage.setItem(STORAGE_KEY, expanded.value ? '1' : '0') } catch (_) {}
}

const allEntries = computed(() => ws.eventsNewestFirst)
const totalCount = computed(() => ws.eventCount)
const filteredEntries = computed(() => {
  let list = allEntries.value
  if (srcFilter.value) list = list.filter(function(e) { return e.source === srcFilter.value })
  if (lvlFilter.value) list = list.filter(function(e) { return e.level === lvlFilter.value })
  return list
})
const lastLine = computed(() => filteredEntries.value[0]?.line ?? '')

function formatTime(t) {
  const d = t instanceof Date ? t : new Date(t)
  return d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
function toneClass(e) {
  if (e.level === 'error') return 'text-red-700 dark:text-red-300'
  if (e.level === 'warn')  return 'text-amber-700 dark:text-amber-300'
  switch (e.type) {
    case 'job_status':       return 'fg-primary'
    case 'phase_summary':    return 'text-emerald-700 dark:text-emerald-300'
    case 'worker_saturation':return 'text-red-700 dark:text-red-300'
    case 'storage_metric':   return 'fg-secondary'
    default:                 return 'fg-secondary'
  }
}

watch(filteredEntries, async function() {
  if (!expanded.value || !autoScroll.value) return
  await nextTick()
  if (bodyEl.value) bodyEl.value.scrollTop = 0
})
</script>
