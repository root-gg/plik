import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatTile from '../components/StatTile.vue'

describe('StatTile', () => {
    it('humanizes counts with locale thousands grouping', () => {
        const w = mount(StatTile, { props: { label: 'Uploads', value: 1234567 } })
        expect(w.text()).toContain('1,234,567')
        expect(w.text()).toContain('Uploads')
    })

    it('renders 0 for a zero count', () => {
        const w = mount(StatTile, { props: { label: 'X', value: 0 } })
        expect(w.get('.text-2xl').text()).toBe('0')
    })

    it('never renders NaN for a missing value', () => {
        const w = mount(StatTile, { props: { label: 'X', value: undefined } })
        expect(w.get('.text-2xl').text()).toBe('0')
    })

    // ── Explicit null means "unavailable", distinct from an honest 0 ──
    it('renders "—" for an explicit null value (count format)', () => {
        const w = mount(StatTile, { props: { label: 'X', value: null } })
        expect(w.get('.text-2xl').text()).toBe('—')
    })

    it('renders "—" for an explicit null value (size format), not a false "0 B"', () => {
        const w = mount(StatTile, { props: { label: 'X', value: null, format: 'size' } })
        expect(w.get('.text-2xl').text()).toBe('—')
    })

    it('formats sizes via humanReadableSize', () => {
        const w = mount(StatTile, { props: { label: 'Total', value: 1500000, format: 'size' } })
        expect(w.text()).toContain('1.50 MB')
    })

    it('shows 0 B for a zero size', () => {
        const w = mount(StatTile, { props: { label: 'Total', value: 0, format: 'size' } })
        expect(w.get('.text-2xl').text()).toBe('0 B')
    })

    it('preserves tabular-nums on the value', () => {
        const w = mount(StatTile, { props: { label: 'X', value: 5 } })
        expect(w.get('.text-2xl').classes()).toContain('tabular-nums')
    })

    it('renders a bordered card and custom value colour when bordered', () => {
        const w = mount(StatTile, {
            props: { label: 'X', value: 5, bordered: true, valueClass: 'text-accent-300' },
        })
        expect(w.classes()).toContain('border')
        expect(w.get('.text-2xl').classes()).toContain('text-accent-300')
    })

    it('renders an optional sub-label', () => {
        const w = mount(StatTile, { props: { label: 'X', value: 5, subLabel: 'extra' } })
        expect(w.text()).toContain('extra')
    })
})
