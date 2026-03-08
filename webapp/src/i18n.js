import { createI18n } from 'vue-i18n'
import en from './locales/en.json'
import fr from './locales/fr.json'

const STORAGE_KEY = 'plik-locale'
export const SUPPORTED_LOCALES = ['en', 'fr']

/**
 * Locale display names — used by the language picker.
 * Keys must match SUPPORTED_LOCALES entries.
 */
export const LOCALE_LABELS = {
    en: 'English',
    fr: 'Français',
}

/**
 * Flag SVGs for each locale — inline data URIs that render on all platforms.
 */
export const LOCALE_FLAGS = {
    en: `<svg viewBox="0 0 60 30" xmlns="http://www.w3.org/2000/svg"><clipPath id="a"><path d="M0 0v30h60V0z"/></clipPath><g clip-path="url(#a)"><path fill="#012169" d="M0 0v30h60V0z"/><path d="m0 0 60 30m0-30L0 30" stroke="#fff" stroke-width="6"/><path d="m0 0 60 30m0-30L0 30" stroke="#C8102E" stroke-width="4" clip-path="url(#a)"/><path d="M30 0v30M0 15h60" stroke="#fff" stroke-width="10"/><path d="M30 0v30M0 15h60" stroke="#C8102E" stroke-width="6"/></g></svg>`,
    fr: `<svg viewBox="0 0 3 2" xmlns="http://www.w3.org/2000/svg"><rect fill="#002395" width="1" height="2"/><rect fill="#fff" x="1" width="1" height="2"/><rect fill="#ED2939" x="2" width="1" height="2"/></svg>`,
}

/**
 * Detect locale: localStorage → browser language → 'en'
 */
function detectLocale() {
    // 1. localStorage preference
    try {
        const stored = localStorage.getItem(STORAGE_KEY)
        if (stored && SUPPORTED_LOCALES.includes(stored)) return stored
    } catch { /* private browsing / disabled storage */ }

    // 2. Browser language (prefix match: "fr-FR" → "fr")
    const lang = navigator.language?.split('-')[0]
    if (lang && SUPPORTED_LOCALES.includes(lang)) return lang

    // 3. Fallback
    return 'en'
}

const i18n = createI18n({
    legacy: false,          // use Composition API
    globalInjection: true,  // ensure $t is available in all templates
    locale: detectLocale(),
    fallbackLocale: 'en',
    messages: { en, fr },
})

/**
 * Switch locale.
 */
export function setLocale(lang) {
    if (!SUPPORTED_LOCALES.includes(lang)) return

    i18n.global.locale.value = lang
    try { localStorage.setItem(STORAGE_KEY, lang) } catch { /* ignore */ }
    document.documentElement.lang = lang
}

/**
 * Returns the current locale string (e.g. 'en', 'fr').
 */
export function getLocale() {
    return i18n.global.locale.value
}

export default i18n
