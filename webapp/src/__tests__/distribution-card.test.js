import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DistributionCard from '../components/DistributionCard.vue'

const items = [
    { key: 'a', label: 'Alpha', value: 3 },
    { key: 'b', label: 'Beta', value: 1 },
    { key: 'c', label: 'Gamma', value: 0 },
]

describe('DistributionCard', () => {
    it('renders the title, one row per item, and row labels', () => {
        const w = mount(DistributionCard, { props: { title: 'Dist', items, total: 4 } })
        expect(w.text()).toContain('Dist')
        expect(w.text()).toContain('Alpha')
        expect(w.text()).toContain('Beta')
        expect(w.text()).toContain('Gamma')
        expect(w.findAll('.h-full')).toHaveLength(3)
    })

    it('sizes each bar proportionally to value/total', () => {
        const w = mount(DistributionCard, { props: { title: 'D', items, total: 4 } })
        const bars = w.findAll('.h-full')
        expect(bars[0].attributes('style')).toContain('width: 75%')
        expect(bars[1].attributes('style')).toContain('width: 25%')
        expect(bars[2].attributes('style')).toContain('width: 0%')
    })

    it('handles total=0 without NaN (all bars 0%)', () => {
        const w = mount(DistributionCard, { props: { title: 'D', items, total: 0 } })
        for (const bar of w.findAll('.h-full')) {
            expect(bar.attributes('style')).toContain('width: 0%')
        }
    })

    it('humanizes row values with thousands grouping', () => {
        const w = mount(DistributionCard, {
            props: { title: 'D', items: [{ key: 'x', label: 'X', value: 1234 }], total: 2000 },
        })
        expect(w.text()).toContain('1,234')
    })

    it('applies the single barClass token to every bar', () => {
        const w = mount(DistributionCard, {
            props: { title: 'D', items, total: 4, barClass: 'bg-emerald-400' },
        })
        expect(w.find('.h-full').classes()).toContain('bg-emerald-400')
    })

    it('defaults the bar colour to the accent token', () => {
        const w = mount(DistributionCard, { props: { title: 'D', items, total: 4 } })
        expect(w.find('.h-full').classes()).toContain('bg-accent-400')
    })

    // ── Optional caption rendered as a native title-attribute tooltip ──
    it('renders the caption as a title attribute on the heading when provided', () => {
        const w = mount(DistributionCard, { props: { title: 'Dist', items, total: 4, caption: 'Explains the heading' } })
        expect(w.get('h3').attributes('title')).toBe('Explains the heading')
    })

    it('omits the title attribute when no caption is given', () => {
        const w = mount(DistributionCard, { props: { title: 'Dist', items, total: 4 } })
        expect(w.get('h3').attributes('title')).toBeUndefined()
    })
})
