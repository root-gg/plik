import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SegmentedControl from '../components/SegmentedControl.vue'

const options = [
    { value: '1d', label: 'Today' },
    { value: '7d', label: '7 days' },
    { value: '30d', label: '30 days' },
    { value: 'all', label: 'Lifetime' },
]

function mountControl(props = {}) {
    return mount(SegmentedControl, { props: { options, modelValue: '7d', ...props } })
}

describe('SegmentedControl', () => {
    it('renders one button per option with its label', () => {
        const w = mountControl()
        const labels = w.findAll('button').map((b) => b.text())
        expect(labels).toEqual(['Today', '7 days', '30 days', 'Lifetime'])
    })

    it('marks only the active option as pressed/highlighted', () => {
        const w = mountControl({ modelValue: '30d' })
        const buttons = w.findAll('button')
        const active = buttons.find((b) => b.text() === '30 days')
        const inactive = buttons.find((b) => b.text() === 'Today')
        expect(active.attributes('aria-pressed')).toBe('true')
        expect(active.classes()).toContain('text-accent-300')
        expect(inactive.attributes('aria-pressed')).toBe('false')
        expect(inactive.classes()).not.toContain('text-accent-300')
    })

    it('emits update:modelValue with the clicked option\'s value (not the label)', async () => {
        const w = mountControl({ modelValue: '7d' })
        const button = w.findAll('button').find((b) => b.text() === 'Lifetime')
        await button.trigger('click')
        expect(w.emitted('update:modelValue')).toEqual([['all']])
    })

    it('clicking the already-active option still re-emits (callers own the same-value guard, if any)', async () => {
        const w = mountControl({ modelValue: '7d' })
        const button = w.findAll('button').find((b) => b.text() === '7 days')
        await button.trigger('click')
        expect(w.emitted('update:modelValue')).toEqual([['7d']])
    })

    it('exposes role="group" and the given aria-label for screen readers', () => {
        const w = mountControl({ ariaLabel: 'Trending window' })
        expect(w.attributes('role')).toBe('group')
        expect(w.attributes('aria-label')).toBe('Trending window')
    })

    it('every option is a native <button type="button"> (keyboard-focusable and Enter/Space-activatable for free)', () => {
        const w = mountControl()
        for (const b of w.findAll('button')) {
            expect(b.attributes('type')).toBe('button')
        }
    })
})
