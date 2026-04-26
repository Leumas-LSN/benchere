<template>
  <div class="flex h-full bg-canvas">
    <!-- Sidebar -->
    <aside
      :class="[
        'flex flex-col shrink-0 border-r transition-[width] duration-200 ease-out h-full',
        sidebarOpen ? 'w-60' : 'w-[68px]',
      ]"
      style="background: var(--bg-surface); border-color: var(--border-default);"
    >
      <!-- Brand row -->
      <div
        class="h-16 flex items-center px-4 border-b shrink-0"
        style="border-color: var(--border-subtle);"
      >
        <RouterLink
          to="/"
          class="flex items-center min-w-0 outline-none focus-visible:ring-2 focus-visible:ring-brand-500/60 rounded-md"
          aria-label="Benchere, retour à l'accueil"
        >
          <BenchereWordmark v-if="sidebarOpen" size="md" />
          <BenchereMark v-else />
        </RouterLink>
      </div>

      <!-- Nav -->
      <nav class="flex-1 px-2 py-3 space-y-0.5 overflow-y-auto">
        <RouterLink
          v-for="link in navLinks"
          :key="link.to"
          :to="link.to"
          :class="['nav-link', { 'nav-link-active': isActive(link) }]"
          :title="!sidebarOpen ? link.label : undefined"
        >
          <Icon
            :name="link.icon"
            :size="18"
            :class="isActive(link) ? 'text-brand-600 dark:text-brand-400' : ''"
          />
          <span v-if="sidebarOpen" class="truncate">{{ link.label }}</span>
        </RouterLink>
      </nav>

      <!-- Footer: version + API status + sidebar toggle -->
      <div class="px-2 py-3 border-t" style="border-color: var(--border-subtle);">
        <div
          v-if="sidebarOpen"
          class="flex items-center gap-2 px-3 py-1 text-xs fg-muted num mb-1"
        >
          <span class="fg-muted">Benchere</span>
          <span class="ml-auto">{{ appVersion || '—' }}</span>
        </div>
        <div
          v-if="sidebarOpen"
          class="flex items-center gap-2 px-3 py-2 rounded-lg surface-muted text-xs mb-2"
        >
          <span
            class="w-1.5 h-1.5 rounded-full shrink-0"
            :class="apiOnline ? 'bg-emerald-500 animate-pulse-dot' : 'bg-red-500'"
          ></span>
          <span class="fg-secondary">API</span>
          <span class="ml-auto fg-muted num">{{ apiOnline ? t('app.apiOnline') : t('app.apiOffline') }}</span>
        </div>
        <div
          v-else
          class="flex items-center justify-center h-9 mb-1"
          :title="apiOnline ? 'API en ligne' : 'API hors ligne'"
        >
          <span
            class="w-2 h-2 rounded-full"
            :class="apiOnline ? 'bg-emerald-500 animate-pulse-dot' : 'bg-red-500'"
          ></span>
        </div>

        <button
          type="button"
          class="w-full h-9 flex items-center gap-2 px-3 rounded-lg fg-muted hover:fg-primary hover:bg-soft transition-colors text-xs"
          :aria-label="sidebarOpen ? t('app.sidebar.collapseAria') : t('app.sidebar.expandAria')"
          @click="sidebarOpen = !sidebarOpen"
        >
          <Icon :name="sidebarOpen ? 'chevron_left' : 'chevron_right'" :size="14" />
          <span v-if="sidebarOpen">{{ t('app.collapse') }}</span>
        </button>
      </div>
    </aside>

    <!-- Main column -->
    <div class="flex-1 flex flex-col min-w-0 h-full">
      <!-- Top bar -->
      <header
        class="h-16 shrink-0 flex items-center gap-3 px-6 border-b"
        style="background: var(--bg-surface); border-color: var(--border-default);"
      >
        <div class="flex items-center gap-2 min-w-0 flex-1">
          <Icon :name="breadcrumb.icon" :size="16" class="fg-muted" />
          <span class="text-sm font-medium fg-primary truncate">{{ breadcrumb.label }}</span>
          <template v-if="breadcrumb.sub">
            <Icon name="chevron_right" :size="14" class="fg-faint" />
            <span class="text-sm fg-muted truncate">{{ breadcrumb.sub }}</span>
          </template>
        </div>

        <RouterLink
          to="/jobs/new"
          class="btn-primary btn-sm hidden sm:inline-flex"
        >
          <Icon name="plus" :size="14" />
          Nouveau benchmark
        </RouterLink>

        <ThemeToggle />
      </header>

      <main class="flex-1 overflow-y-auto bg-canvas">
        <RouterView v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </RouterView>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, h, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import Icon              from './components/Icon.vue'
import ThemeToggle       from './components/ThemeToggle.vue'
import BenchereWordmark  from './components/BenchereWordmark.vue'
import { useThemeStore } from './stores/theme.js'

const { t } = useI18n()
useThemeStore()

const route = useRoute()
const sidebarOpen = ref(true)
const apiOnline   = ref(true)
const appVersion  = ref('')

const navLinks = computed(() => [
  { to: '/',         icon: 'home',    label: t('nav.home')     },
  { to: '/jobs/new', icon: 'play',    label: t('nav.newJob')   },
  { to: '/profiles', icon: 'layers',  label: t('nav.profiles') },
  { to: '/history',  icon: 'history', label: t('nav.history')  },
  { to: '/settings', icon: 'cog',     label: t('nav.settings') },
])

function isActive(link) {
  if (link.to === '/') return route.path === '/'
  return route.path === link.to || route.path.startsWith(link.to + '/')
}

const breadcrumb = computed(() => {
  const p = route.path
  if (p === '/')                  return { icon: 'home',     label: t('breadcrumb.home') }
  if (p === '/jobs/new')          return { icon: 'play',     label: t('breadcrumb.newJob') }
  if (p === '/profiles')          return { icon: 'layers',   label: t('breadcrumb.profiles') }
  if (p === '/history')           return { icon: 'history',  label: t('breadcrumb.history') }
  if (p === '/settings')          return { icon: 'cog',      label: t('breadcrumb.settings') }
  if (p.startsWith('/dashboard/'))return { icon: 'activity', label: t('breadcrumb.dashboard'), sub: route.params.id }
  if (p.startsWith('/jobs/'))     return { icon: 'file_text',label: t('breadcrumb.job'), sub: route.params.id }
  return { icon: 'home', label: 'Benchere' }
})

// Compact mark for collapsed sidebar : un "b" lourd + barre orange skewée
const BenchereMark = {
  render: () => h('span', {
    class: 'inline-flex items-baseline',
    style: 'font-family:Geist,ui-sans-serif,system-ui,sans-serif;font-weight:800;font-size:22px;letter-spacing:-0.04em;line-height:1;color:currentColor',
    'aria-hidden': true,
  }, [
    h('span', 'b'),
    h('span', { style: 'display:inline-block;width:3px;height:18px;background:#f97316;transform:skew(-18deg);margin-left:3px;align-self:center;border-radius:1px' }),
  ]),
}

let pingTimer = null
async function pingApi() {
  try {
    const res = await fetch('/api/jobs', { method: 'GET', cache: 'no-store' })
    apiOnline.value = res.ok
  } catch (_) {
    apiOnline.value = false
  }
}

onMounted(async () => {
  pingApi()
  pingTimer = setInterval(pingApi, 15000)
  try {
    const v = await fetch('/api/version').then(r => r.json())
    appVersion.value = v.version || ''
  } catch (_) { /* version endpoint not available, leave empty */ }
  if (window.matchMedia('(max-width: 768px)').matches) sidebarOpen.value = false
})

onUnmounted(() => clearInterval(pingTimer))
</script>

<style>
.fade-enter-active, .fade-leave-active { transition: opacity 160ms ease, transform 160ms ease; }
.fade-enter-from { opacity: 0; transform: translateY(4px); }
.fade-leave-to   { opacity: 0; transform: translateY(-2px); }
</style>
