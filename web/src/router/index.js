import { createRouter, createWebHistory } from 'vue-router'

const HomeView       = () => import('../views/HomeView.vue')
const SettingsView   = () => import('../views/SettingsView.vue')
const OnboardingView = () => import('../views/OnboardingView.vue')
const NewJobView     = () => import('../views/NewJobView.vue')
const DashboardView  = () => import('../views/DashboardView.vue')
const JobDetailView  = () => import('../views/JobDetailView.vue')
const HistoryView    = () => import('../views/HistoryView.vue')
const ProfilesView   = () => import('../views/ProfilesView.vue')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/',              component: HomeView },
    { path: '/onboarding',    component: OnboardingView },
    { path: '/settings',      component: SettingsView },
    { path: '/jobs/new',      component: NewJobView },
    { path: '/dashboard/:id', component: DashboardView },
    { path: '/jobs/:id',      component: JobDetailView },
    { path: '/history',       component: HistoryView },
    { path: '/profiles',      component: ProfilesView },
  ],
})

// Redirect to /onboarding on first visit (settings empty), unless the user
// is already on it. Skip for explicit settings access so power users can
// reach the edit form without going through the wizard.
let alreadyChecked = false
router.beforeEach(async (to, from, next) => {
  if (alreadyChecked || to.path === '/onboarding' || to.path === '/settings') return next()
  alreadyChecked = true
  try {
    const r = await fetch('/api/settings').then(r => r.json())
    if (!r || !r.proxmox_url) return next('/onboarding')
  } catch (_) { /* api unreachable, just continue */ }
  next()
})

export default router
