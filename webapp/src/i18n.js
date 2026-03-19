import { createI18n } from 'vue-i18n'
import en from './locales/en.json'
import fr from './locales/fr.json'

const i18n = createI18n({
    legacy: false,          // use Composition API
    globalInjection: true,  // ensure $t is available in all templates
    locale: 'en',           // default; overridden by loadSettings() before mount
    fallbackLocale: 'en',
    messages: { en, fr },
})

/**
 * Switch locale.
 * Called by setUserLanguage() in settings.js after resolving 'auto'.
 */
export function setLocale(lang) {
    i18n.global.locale.value = lang
    document.documentElement.lang = lang
}

/**
 * Returns the current locale string (e.g. 'en', 'fr').
 */
export function getLocale() {
    return i18n.global.locale.value
}

export default i18n
