import { createI18n } from 'vue-i18n'
import fr from './fr.js'
import en from './en.js'

// Locale lookup order:
//   1. localStorage (set during onboarding or via settings)
//   2. browser language
//   3. 'fr' default
function detectLocale() {
  const stored = localStorage.getItem('benchere.locale')
  if (stored === 'fr' || stored === 'en') return stored
  const nav = (navigator.language || 'fr').slice(0, 2).toLowerCase()
  return nav === 'en' ? 'en' : 'fr'
}

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: detectLocale(),
  fallbackLocale: 'en',
  messages: { fr, en },
})

export function setLocale(locale) {
  if (locale !== 'fr' && locale !== 'en') return
  i18n.global.locale.value = locale
  localStorage.setItem('benchere.locale', locale)
}
