import { describe, it, expect } from 'vitest'
import { themeHref } from '../settings.js'

describe('themeHref', () => {
    it('appends the server version so browsers refetch after an upgrade', () => {
        expect(themeHref('light', '1.4.3')).toBe('themes/light.css?v=1.4.3')
    })

    it('encodes the version', () => {
        expect(themeHref('nord', '1.4.3 rc/1')).toBe('themes/nord.css?v=1.4.3%20rc%2F1')
    })

    it('falls back to the plain path when the version is unknown', () => {
        expect(themeHref('light', '')).toBe('themes/light.css')
        expect(themeHref('light', undefined)).toBe('themes/light.css')
    })
})
