// Webapp Settings Store
// Fetches /settings.json at startup and provides reactive state.
// The file uses JSONC (JSON with comments) — single-line // comments are stripped before parsing.

import { reactive, ref } from 'vue'

// All built-in themes with display metadata
const BUILTIN_THEMES = [
    { name: 'auto', label: 'Auto' },
    { name: 'dark', label: 'Dark' },
    { name: 'light', label: 'Light' },
    { name: 'catppuccin-mocha', label: 'Catppuccin Mocha' },
    { name: 'catppuccin-latte', label: 'Catppuccin Latte' },
    { name: 'nord', label: 'Nord' },
    { name: 'nord-light', label: 'Nord Light' },
    { name: 'solarized-dark', label: 'Solarized Dark' },
    { name: 'solarized-light', label: 'Solarized Light' },
    { name: 'matrix', label: 'Matrix' },
    { name: 'hexless', label: 'Hexless' },
]

// All built-in languages with display metadata.
// Flags are SVG files in webapp/public/flags/ (same pattern as themes in themes/).
export const BUILTIN_LANGUAGES = [
    { name: 'auto', label: 'Auto' },
    { name: 'en', label: 'English', flag: '/flags/en.svg' },
    { name: 'fr', label: 'Français', flag: '/flags/fr.svg' },
]

const STORAGE_KEY = 'plik-theme'
const LANGUAGE_STORAGE_KEY = 'plik-locale'

// White-label safe defaults: empty name so "Plik" is never leaked
// if settings.json is missing or fails to load.
export const settings = reactive({
    name: '',
    logo: '',
    theme: 'auto',
    themes: ['*'],
    defaultDarkTheme: 'dark',
    defaultLightTheme: 'light',
    language: 'auto',
    languages: ['*'],
    backgroundImage: '',
    backgroundColor: '',
    overlayOpacity: 0,
    customCSS: '',
    customJS: '',
    footer: '',
})

/** Reactive current theme value — used by ThemePicker */
export const currentTheme = ref('auto')

/** Reactive current language value — used by LanguagePicker */
export const currentLanguage = ref('auto')

/**
 * Strip single-line // comments from JSONC text.
 * Ignores // inside quoted strings.
 */
function stripJSONCComments(text) {
    return text.replace(/(\"(?:[^\"\\]|\\.)*\")|\/\/[^\n]*/g, (match, quoted) => {
        return quoted || ''
    })
}

/**
 * Dynamically inject a <link rel="stylesheet"> and wait for it to load.
 */
function injectCSS(href) {
    return new Promise((resolve) => {
        const link = document.createElement('link')
        link.rel = 'stylesheet'
        link.href = href
        link.onload = resolve
        link.onerror = resolve // Don't block on failure
        document.head.appendChild(link)
    })
}

/**
 * Dynamically inject a <script> and wait for it to load.
 */
function injectJS(src) {
    return new Promise((resolve) => {
        const script = document.createElement('script')
        script.src = src
        script.onload = resolve
        script.onerror = resolve // Don't block on failure
        document.head.appendChild(script)
    })
}

/**
 * Resolve and apply the theme.
 * - "dark" / "light" / "my-custom" → set directly
 * - "auto" → follow OS preference via prefers-color-scheme
 *
 * Non-"dark" themes lazy-load their CSS from /themes/{name}.css
 * before setting the data-theme attribute (prevents unstyled flash).
 */
const loadedThemes = new Set()
let autoListener = null

export async function applyTheme(value) {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')

    // Remove previous auto listener if switching away
    if (autoListener) {
        mq.removeEventListener('change', autoListener)
        autoListener = null
    }

    async function resolve() {
        const name = value === 'auto'
            ? (mq.matches ? settings.defaultDarkTheme : settings.defaultLightTheme)
            : value

        // "dark" is the compiled-in default — no external CSS needed.
        // All other themes (including "light") load from /themes/{name}.css.
        if (name !== 'dark') {
            const href = `/themes/${name}.css`
            if (!loadedThemes.has(href)) {
                await injectCSS(href)
                loadedThemes.add(href)
            }
        }

        document.documentElement.dataset.theme = name
        document.documentElement.style.colorScheme = name === 'dark' ? 'dark' : ''
    }

    await resolve()

    // Live-switch when the user toggles OS dark mode (only for "auto")
    if (value === 'auto') {
        autoListener = resolve
        mq.addEventListener('change', resolve)
    }
}

/**
 * Get the list of available themes for the picker.
 * - `["*"]`    → expands to all built-in themes (default)
 * - `[]`       → empty list (no picker, dark only)
 * - Otherwise  → only the listed themes
 */
export function getAvailableThemes() {
    const result = []
    for (const entry of settings.themes) {
        if (entry === '*') {
            // Expand wildcard to all built-ins not already in the result
            for (const bt of BUILTIN_THEMES) {
                if (!result.some(r => r.name === bt.name)) {
                    result.push(bt)
                }
            }
        } else if (typeof entry === 'string') {
            const builtin = BUILTIN_THEMES.find(t => t.name === entry)
            result.push(builtin || { name: entry, label: entry })
        } else {
            result.push({ name: entry.name, label: entry.label || entry.name })
        }
    }
    return result
}

/**
 * Get the user's preferred theme from localStorage, falling back to settings default.
 * Validates the stored theme against the available themes list — if the stored
 * theme is not in the list, it is ignored (prevents stale localStorage from
 * overriding a restricted theme configuration).
 * Also validates settings.theme — if it's not available, falls back to the
 * first available theme (handles single-theme deployments).
 */
export function getUserTheme() {
    const available = getAvailableThemes()

    try {
        const stored = localStorage.getItem(STORAGE_KEY)
        if (stored && available.some(t => t.name === stored)) {
            return stored
        }
        if (stored) {
            // Stored theme is not available — clear stale value
            localStorage.removeItem(STORAGE_KEY)
        }
    } catch {
        // localStorage unavailable
    }

    // Validate the settings default against available themes
    if (available.some(t => t.name === settings.theme)) {
        return settings.theme
    }

    // Neither stored nor default is valid — use first available theme
    return available.length > 0 ? available[0].name : 'dark'
}

/**
 * Set the user's theme preference — writes to localStorage and applies immediately.
 * When logged in, also persists to the server (fire-and-forget).
 * @param {string} name - theme name
 * @param {object} opts
 * @param {boolean} opts.skipSync - if true, skip server persistence (used when applying server value)
 */
export async function setUserTheme(name, { skipSync = false } = {}) {
    try {
        localStorage.setItem(STORAGE_KEY, name)
    } catch {
        // localStorage unavailable
    }
    currentTheme.value = name
    await applyTheme(name)

    // Persist to backend if logged in (fire-and-forget)
    if (!skipSync) {
        // Lazy import to avoid circular dependency (authStore imports settings)
        const { auth } = await import('./authStore.js')
        if (auth.user) {
            const { patchMe } = await import('./api.js')
            patchMe({ theme: name }).catch(() => { })
        }
    }
}

/**
 * Apply the user's server-side theme on login/session restore.
 * No-op if the user hasn't set a theme server-side (theme is empty)
 * or if the stored theme is not in the available themes list (stale DB value
 * from a theme that was later removed from settings.json).
 */
export function syncThemeFromUser(user) {
    if (!user?.theme) return
    const available = getAvailableThemes()
    if (available.some(t => t.name === user.theme)) {
        setUserTheme(user.theme, { skipSync: true })
    }
}

// ── Language infrastructure ────────────────────────────────────────────

/**
 * Get the list of available languages for the picker.
 * - `["*"]`    → expands to all built-in languages (default)
 * - `[]`       → empty list (no picker, English only)
 * - Otherwise  → only the listed languages
 */
export function getAvailableLanguages() {
    const result = []
    for (const entry of settings.languages) {
        if (entry === '*') {
            for (const bl of BUILTIN_LANGUAGES) {
                if (!result.some(r => r.name === bl.name)) {
                    result.push(bl)
                }
            }
        } else if (typeof entry === 'string') {
            const builtin = BUILTIN_LANGUAGES.find(l => l.name === entry)
            result.push(builtin || { name: entry, label: entry })
        } else {
            result.push({ name: entry.name, label: entry.label || entry.name, flag: entry.flag })
        }
    }
    return result
}

/**
 * Resolve "auto" language: detect from browser `navigator.language`.
 * Returns the best matching supported locale, or 'en' as fallback.
 */
export function resolveAutoLanguage() {
    const lang = navigator.language?.split('-')[0]
    const available = getAvailableLanguages()
    // Only match against concrete languages (not 'auto')
    if (lang && available.some(l => l.name === lang)) return lang
    // Fallback: first non-auto language, or 'en'
    const first = available.find(l => l.name !== 'auto')
    return first ? first.name : 'en'
}

/**
 * Get the user's preferred language from localStorage, falling back to settings default.
 * Validates against available languages — stale values are cleared.
 */
export function getUserLanguage() {
    const available = getAvailableLanguages()

    try {
        const stored = localStorage.getItem(LANGUAGE_STORAGE_KEY)
        if (stored && available.some(l => l.name === stored)) {
            return stored
        }
        if (stored) {
            localStorage.removeItem(LANGUAGE_STORAGE_KEY)
        }
    } catch {
        // localStorage unavailable
    }

    // Validate the settings default against available languages
    if (available.some(l => l.name === settings.language)) {
        return settings.language
    }

    // Neither stored nor default is valid — use first available
    return available.length > 0 ? available[0].name : 'auto'
}

/**
 * Set the user's language preference — writes to localStorage and applies immediately.
 * When logged in, also persists to the server (fire-and-forget).
 * @param {string} name - language name ('auto', 'en', 'fr', etc.)
 * @param {object} opts
 * @param {boolean} opts.skipSync - if true, skip server persistence (used when applying server value)
 */
export async function setUserLanguage(name, { skipSync = false } = {}) {
    try {
        localStorage.setItem(LANGUAGE_STORAGE_KEY, name)
    } catch {
        // localStorage unavailable
    }
    currentLanguage.value = name

    // Apply the effective locale (resolve 'auto' to concrete language)
    const effectiveLocale = name === 'auto' ? resolveAutoLanguage() : name
    const { setLocale } = await import('./i18n.js')
    setLocale(effectiveLocale)

    // Persist to backend if logged in (fire-and-forget)
    if (!skipSync) {
        const { auth } = await import('./authStore.js')
        if (auth.user) {
            const { patchMe } = await import('./api.js')
            patchMe({ language: name }).catch(() => { })
        }
    }
}

/**
 * Apply the user's server-side language on login/session restore.
 * No-op if the user hasn't set a language server-side.
 */
export function syncLanguageFromUser(user) {
    if (!user?.language) return
    const available = getAvailableLanguages()
    if (available.some(l => l.name === user.language)) {
        setUserLanguage(user.language, { skipSync: true })
    }
}

// ── Initialization ─────────────────────────────────────────────────────

/**
 * Load webapp settings from /settings.json before Vue mounts.
 * - Fetches and parses the JSONC file
 * - Merges values into the reactive settings object
 * - Applies theme (before Vue mounts → zero flash)
 * - Applies language (before Vue mounts → correct locale on first render)
 * - Sets document.title from settings.name
 * - Conditionally injects custom CSS/JS if paths are configured
 */
export async function loadSettings() {
    try {
        const resp = await fetch('/settings.json')
        if (resp.ok) {
            const text = await resp.text()
            const data = JSON.parse(stripJSONCComments(text))
            Object.assign(settings, data)
        }
    } catch {
        // Silently fall back to defaults
    }

    // Resolve theme: localStorage > settings.json default
    const theme = getUserTheme()
    currentTheme.value = theme

    // Apply theme before anything renders
    await applyTheme(theme)

    // Resolve language: localStorage > settings.json default
    const language = getUserLanguage()
    currentLanguage.value = language

    // Apply language before Vue mounts
    const effectiveLocale = language === 'auto' ? resolveAutoLanguage() : language
    const { setLocale } = await import('./i18n.js')
    setLocale(effectiveLocale)

    // Set page title (stays empty if no custom settings — white-label safe)
    document.title = settings.name

    // Conditionally inject custom assets (before Vue mounts)
    const injections = []
    if (settings.customCSS) {
        injections.push(injectCSS(settings.customCSS))
    }
    if (settings.customJS) {
        injections.push(injectJS(settings.customJS))
    }
    if (injections.length) {
        await Promise.all(injections)
    }
}
