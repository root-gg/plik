import { createI18n } from 'vue-i18n'
import de from './locales/de.json'
import en from './locales/en.json'
import es from './locales/es.json'
import fr from './locales/fr.json'
import hi from './locales/hi.json'
import it from './locales/it.json'
import ja from './locales/ja.json'
import nl from './locales/nl.json'
import pl from './locales/pl.json'
import pt from './locales/pt.json'
import ru from './locales/ru.json'
import sv from './locales/sv.json'
import zh from './locales/zh.json'

const i18n = createI18n({
    legacy: false,          // use Composition API
    globalInjection: true,  // ensure $t is available in all templates
    locale: 'en',           // default; overridden by loadSettings() before mount
    fallbackLocale: 'en',
    messages: { de, en, es, fr, hi, it, ja, nl, pl, pt, ru, sv, zh },
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
