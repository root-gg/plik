import { createI18n } from 'vue-i18n'
import en from './locales/en.json'
import fr from './locales/fr.json'
import de from './locales/de.json'
import es from './locales/es.json'
import it from './locales/it.json'
import pt from './locales/pt.json'
import nl from './locales/nl.json'
import pl from './locales/pl.json'
import zh from './locales/zh.json'
import ru from './locales/ru.json'

const i18n = createI18n({
    legacy: false,          // use Composition API
    globalInjection: true,  // ensure $t is available in all templates
    locale: 'en',           // default; overridden by loadSettings() before mount
    fallbackLocale: 'en',
    messages: { en, fr, de, es, it, pt, nl, pl, zh, ru },
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
