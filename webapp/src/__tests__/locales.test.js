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
