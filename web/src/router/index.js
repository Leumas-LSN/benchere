import { createRouter, createWebHistory } from 'vue-router'

const HomeView      = () => import('../views/HomeView.vue')
const SettingsView  = () => import('../views/SettingsView.vue')
const NewJobView    = () => import('../views/NewJobView.vue')
const DashboardView = () => import('../views/DashboardView.vue')
const JobDetailView = () => import('../views/JobDetailView.vue')
const HistoryView   = () => import('../views/HistoryView.vue')
const ProfilesView  = () => import('../views/ProfilesView.vue')

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/',              component: HomeView },
    { path: '/settings',      component: SettingsView },
    { path: '/jobs/new',      component: NewJobView },
    { path: '/dashboard/:id', component: DashboardView },
    { path: '/jobs/:id',      component: JobDetailView },
    { path: '/history',       component: HistoryView },
    { path: '/profiles',      component: ProfilesView },
  ],
})
