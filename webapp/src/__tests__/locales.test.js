import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'fs'
import { join, basename } from 'path'

// ── Locale key sync ──
// Ensures all locale files have the exact same set of keys as en.json (the reference locale).
// This prevents missing translations from reaching production.

const LOCALES_DIR = join(__dirname, '..', 'locales')

function flattenKeys(obj, prefix = '') {
    const keys = []
    for (const [key, value] of Object.entries(obj)) {
        const fullKey = prefix ? `${prefix}.${key}` : key
        if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
            keys.push(...flattenKeys(value, fullKey))
        } else {
            keys.push(fullKey)
        }
    }
    return keys.sort()
}

// Like flattenKeys, but returns a flat { "a.b.c": value } map instead of just
// the key list — used by the suspicious-identical-to-English check below,
// which needs to compare values, not just presence.
function flattenEntries(obj, prefix = '') {
    const out = {}
    for (const [key, value] of Object.entries(obj)) {
        const fullKey = prefix ? `${prefix}.${key}` : key
        if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
            Object.assign(out, flattenEntries(value, fullKey))
        } else {
            out[fullKey] = value
        }
    }
    return out
}

function loadLocale(filename) {
    const content = readFileSync(join(LOCALES_DIR, filename), 'utf-8')
    return JSON.parse(content)
}

const localeFiles = readdirSync(LOCALES_DIR)
    .filter(f => f.endsWith('.json') && f !== 'en.json')

const referenceData = loadLocale('en.json')
const referenceKeys = flattenKeys(referenceData)

describe('locale key sync', () => {
    it('has en.json as reference with keys', () => {
        expect(referenceKeys.length).toBeGreaterThan(0)
    })

    for (const file of localeFiles) {
        const lang = basename(file, '.json')

        describe(`${lang}.json`, () => {
            const data = loadLocale(file)
            const keys = flattenKeys(data)

            it('has no missing keys', () => {
                const missing = referenceKeys.filter(k => !keys.includes(k))
                if (missing.length > 0) {
                    throw new Error(
                        `${file} is missing ${missing.length} key(s):\n` +
                        missing.map(k => `  - ${k}`).join('\n')
                    )
                }
            })

            it('has no extra keys', () => {
                const extra = keys.filter(k => !referenceKeys.includes(k))
                if (extra.length > 0) {
                    throw new Error(
                        `${file} has ${extra.length} extra key(s) not in en.json:\n` +
                        extra.map(k => `  - ${k}`).join('\n')
                    )
                }
            })

            it('has no empty translation values', () => {
                const empties = []
                function checkEmpty(obj, prefix = '') {
                    for (const [key, value] of Object.entries(obj)) {
                        const fullKey = prefix ? `${prefix}.${key}` : key
                        if (typeof value === 'string' && value.trim() === '') {
                            empties.push(fullKey)
                        } else if (typeof value === 'object' && value !== null) {
                            checkEmpty(value, fullKey)
                        }
                    }
                }
                checkEmpty(data)
                if (empties.length > 0) {
                    throw new Error(
                        `${file} has ${empties.length} empty value(s):\n` +
                        empties.map(k => `  - ${k}`).join('\n')
                    )
                }
            })

            it('preserves {placeholder} tokens from en.json', () => {
                const mismatches = []
                function getPlaceholders(str) {
                    return (str.match(/\{[^}]+\}/g) || []).sort()
                }
                function check(ref, target, prefix = '') {
                    for (const [key, value] of Object.entries(ref)) {
                        const fullKey = prefix ? `${prefix}.${key}` : key
                        if (typeof value === 'string' && target[key] !== undefined) {
                            // For pipe-separated plurals (e.g. "{count} file | {count} files"),
                            // compare placeholders per-form rather than globally.
                            // Languages with more plural forms (e.g. Polish has 3) are valid
                            // as long as each form has the same placeholders.
                            const refForms = value.split('|').map(s => s.trim())
                            const targetForms = target[key].split('|').map(s => s.trim())

                            // Get the unique set of placeholders used across all reference forms
                            const refPlaceholders = getPlaceholders(refForms[0])

                            // Check each target form has the same placeholders
                            for (let i = 0; i < targetForms.length; i++) {
                                const formPlaceholders = getPlaceholders(targetForms[i])
                                if (JSON.stringify(refPlaceholders) !== JSON.stringify(formPlaceholders)) {
                                    mismatches.push(
                                        `  - ${fullKey} [form ${i}]: expected ${JSON.stringify(refPlaceholders)}, got ${JSON.stringify(formPlaceholders)}`
                                    )
                                }
                            }
                        } else if (typeof value === 'object' && value !== null && target[key]) {
                            check(value, target[key], fullKey)
                        }
                    }
                }
                check(referenceData, data)
                if (mismatches.length > 0) {
                    throw new Error(
                        `${file} has ${mismatches.length} placeholder mismatch(es):\n` +
                        mismatches.join('\n')
                    )
                }
            })
        })
    }
})

describe('languagePicker key ordering', () => {
    const allLocaleFiles = readdirSync(LOCALES_DIR).filter(f => f.endsWith('.json'))

    for (const file of allLocaleFiles) {
        it(`${file} has languagePicker keys in alphabetical order`, () => {
            const data = loadLocale(file)
            const picker = data.languagePicker
            expect(picker).toBeDefined()

            // Get language code keys (everything except switchLanguage)
            const langKeys = Object.keys(picker).filter(k => k !== 'switchLanguage')
            const sorted = [...langKeys].sort()

            expect(langKeys).toEqual(sorted)
        })
    }
})

// ── Suspicious-identical-to-English tripwire ──
// The key-sync checks above only catch MISSING/EXTRA keys and placeholder
// drift — they never flag a key that exists in every locale but was simply
// never translated (byte-identical to en.json). That is exactly how the
// project's earlier "73 untranslated keys" incident stayed invisible for a
// while. This check counts, per non-English locale, every key whose value is
// byte-identical to en.json's, and fails if any of them are NOT on that
// locale's allowlist — so a NEW untranslated string added later (e.g. a
// translator missing a locale when adding a feature) fails CI instead of
// silently shipping in English.
//
// The allowlists below were seeded from the current, already-translated state
// of the 12 locales (after the new statsPanel/adminView keys were translated
// into all of them) — they are legitimate identical strings, not a
// blanket exemption:
//   - SHARED_ALLOWLIST: identical in literally every locale because there is
//     nothing to translate — numeric/unit-only ranges (TTL and file-size
//     bucket labels are digits + symbols), the CLI acronym, and each
//     language's own autonym in languagePicker (every locale file lists ALL
//     languages' native names, so e.g. "Deutsch" is the same value in every
//     file *by design* — it is not a translation of the word "German" into
//     the current locale).
//   - Per-locale entries: deliberate, consistently-used loanwords/borrowed
//     terms, spot-checked against the same locale's OTHER strings to confirm
//     they are a deliberate choice and not laziness — e.g. Spanish translates
//     "password" to "Contraseña" everywhere except the deliberately-English
//     "Passphrase" term (a distinct concept in the E2EE UI); Italian has no
//     native word commonly used for "password" in this app's tech register
//     and keeps it as "password"/"Password" consistently throughout the
//     *entire* file, not just in the flagged keys.
//   - it.json's "uploadView.maxPerFile" ("Max {size} per file") is the one
//     borderline case: it reads as valid terse Italian (Max abbreviation +
//     "per" as a genuine Italian preposition + "file" as a borrowed noun) but
//     could also be a pre-existing untranslated leftover. Left on the
//     allowlist pending human-translator confirmation, rather than silently
//     failing an unrelated, out-of-scope locale over one ambiguous string.
describe('locale suspicious-identical-to-English check', () => {
    const SHARED_ALLOWLIST = [
        'header.cli',
        'languagePicker.de',
        'languagePicker.en',
        'languagePicker.es',
        'languagePicker.fr',
        'languagePicker.hi',
        'languagePicker.it',
        'languagePicker.nl',
        'languagePicker.pl',
        'languagePicker.pt',
        'languagePicker.ru',
        'languagePicker.sv',
        'languagePicker.zh',
        'statsPanel.fileSize100m1g',
        'statsPanel.fileSize10g100g',
        'statsPanel.fileSize10m100m',
        'statsPanel.fileSize1g10g',
        'statsPanel.fileSize1m10m',
        'statsPanel.fileSizeGt100g',
        'statsPanel.fileSizeLt1m',
        'statsPanel.ttl1d7d',
        'statsPanel.ttl1h1d',
        'statsPanel.ttl7d30d',
        'statsPanel.ttlGt30d',
        'statsPanel.ttlLt1h',
    ]

    const PER_LOCALE_ALLOWLIST = {
        de: [
            'adminView.optional', 'adminView.uploads', 'adminView.uploadsLabel',
            'badges.stream', 'common.admin', 'common.code',
            'downloadSidebar.downloads', 'downloadSidebar.passphrase',
            'downloadView.passphrasePlaceholder', 'editUser.name',
            'fileRow.downloads', 'fileRow.md5', 'header.admin',
            'homeView.tokens', 'homeView.uploads', 'statsPanel.metricDownloads',
            'statsPanel.metricUploads', 'uploadControls.downloads',
            'uploadControls.filter', 'uploadSidebar.passphrase', 'uploadSidebar.streaming',
        ],
        es: [
            'badges.stream', 'common.admin', 'downloadSidebar.passphrase',
            'downloadView.passphrasePlaceholder', 'fileRow.md5', 'header.admin',
            'homeView.tokens', 'uploadCard.token', 'uploadControls.stream',
            'uploadSidebar.passphrase', 'uploadSidebar.streaming',
        ],
        fr: [
            'adminView.date', 'adminView.quotas', 'badges.stream', 'common.admin',
            'common.code', 'common.qrCode', 'common.timeMinute',
            'downloadSidebar.actions', 'downloadSidebar.passphrase',
            'downloadView.passphrasePlaceholder', 'editUser.quotas', 'header.admin',
            'header.documentation', 'homeView.date', 'uploadControls.date',
            'uploadControls.stream', 'uploadSidebar.expiration', 'uploadSidebar.minutes',
            'uploadSidebar.passphrase', 'uploadSidebar.streaming',
        ],
        hi: [],
        it: [
            'adminView.provider', 'badges.password', 'badges.stream', 'common.admin',
            'downloadSidebar.passphrase', 'downloadSidebar.password',
            'downloadView.passphrasePlaceholder', 'editUser.email', 'editUser.password',
            'editUser.provider', 'fileRow.md5', 'header.account', 'header.admin',
            'loginView.passwordLabel', 'uploadCard.token', 'uploadControls.password',
            'uploadControls.stream', 'uploadSidebar.maxTTL', 'uploadSidebar.passphrase',
            'uploadSidebar.password', 'uploadSidebar.passwordPlaceholder',
            'uploadSidebar.streaming',
            'uploadView.maxPerFile', // borderline — see comment above
        ],
        nl: [
            'adminView.provider', 'adminView.uploads', 'adminView.uploadsLabel',
            'badges.stream', 'common.admin', 'common.code',
            'downloadSidebar.downloads', 'downloadSidebar.passphrase',
            'downloadView.passphrasePlaceholder', 'editUser.provider',
            'fileRow.downloads', 'fileRow.md5', 'fileRow.type', 'header.account',
            'header.admin', 'homeView.tokens', 'homeView.uploads',
            'statsPanel.metricDownloads', 'statsPanel.metricUploads',
            'uploadCard.downloads', 'uploadCard.token', 'uploadControls.downloads',
            'uploadControls.filter', 'uploadControls.stream', 'uploadSidebar.maxTTL',
            'uploadSidebar.passphrase', 'uploadSidebar.streaming',
        ],
        pl: [
            'badges.stream', 'common.admin', 'downloadSidebar.login',
            'downloadSidebar.passphrase', 'downloadView.passphrasePlaceholder',
            'editUser.login', 'fileRow.md5', 'header.admin', 'loginView.loginLabel',
            'uploadCard.token', 'uploadControls.stream', 'uploadSidebar.loginPlaceholder',
            'uploadSidebar.passphrase', 'uploadSidebar.streaming',
        ],
        pt: [
            'badges.stream', 'common.admin', 'downloadSidebar.downloads',
            'downloadSidebar.passphrase', 'downloadView.passphrasePlaceholder',
            'fileRow.downloads', 'fileRow.md5', 'header.admin', 'homeView.tokens',
            'statsPanel.metricDownloads', 'uploadCard.downloads', 'uploadCard.token',
            'uploadControls.downloads', 'uploadControls.stream',
            'uploadSidebar.passphrase', 'uploadSidebar.streaming',
        ],
        ru: ['fileRow.md5'],
        sv: [
            'adminView.maxTTL', 'adminView.maxTTLLabel', 'common.admin',
            'editUser.maxTTL', 'fileRow.md5', 'header.admin', 'homeView.maxTTL',
            'homeView.tokens', 'uploadCard.token', 'uploadControls.filter',
            'uploadSidebar.maxTTL',
        ],
        zh: [],
    }

    const referenceEntries = flattenEntries(referenceData)

    for (const file of localeFiles) {
        const lang = basename(file, '.json')
        const allowed = new Set([...SHARED_ALLOWLIST, ...(PER_LOCALE_ALLOWLIST[lang] || [])])

        it(`${file} has no untranslated (English-identical) strings beyond the allowlist`, () => {
            const entries = flattenEntries(loadLocale(file))
            const suspicious = []
            for (const [key, value] of Object.entries(entries)) {
                if (typeof value !== 'string') continue
                if (referenceEntries[key] === value && !allowed.has(key)) {
                    suspicious.push(key)
                }
            }
            if (suspicious.length > 0) {
                throw new Error(
                    `${file} has ${suspicious.length} string(s) byte-identical to en.json that are ` +
                    `not on the allowlist (likely untranslated):\n` +
                    suspicious.map(k => `  - ${k}: ${JSON.stringify(entries[k])}`).join('\n')
                )
            }
        })

        it(`${file}'s allowlist has no stale entries`, () => {
            // Keeps the allowlist honest (the suite must pass because strings
            // are translated, not because the allowlist is bloated) — an
            // allowlisted key whose value no longer matches en.json (translated
            // since, or the English source changed) should be removed from the
            // allowlist above, not left there unused.
            const entries = flattenEntries(loadLocale(file))
            const localeSpecific = PER_LOCALE_ALLOWLIST[lang] || []
            const stale = [...SHARED_ALLOWLIST, ...localeSpecific].filter(
                (key) => entries[key] !== referenceEntries[key]
            )
            if (stale.length > 0) {
                throw new Error(
                    `${file}'s allowlist has ${stale.length} stale entrie(s) (no longer identical ` +
                    `to en.json — remove from the allowlist):\n` +
                    stale.map(k => `  - ${k}`).join('\n')
                )
            }
        })
    }
})
