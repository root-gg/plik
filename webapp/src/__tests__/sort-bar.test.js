import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SortBar from '../components/SortBar.vue'

const options = [
    { value: 'date', label: 'Date' },
    { value: 'size', label: 'Size' },
    { value: 'lifetimeSize', label: 'Lifetime Size' },
]

describe('SortBar', () => {
    it('renders the label and all option labels', () => {
        const w = mount(SortBar, { props: { label: 'Sort:', options, modelValue: 'date' } })
        expect(w.text()).toContain('Sort:')
        expect(w.text()).toContain('Date')
        expect(w.text()).toContain('Size')
        expect(w.text()).toContain('Lifetime Size')
    })

    it('emits update:modelValue with the chosen option value', async () => {
        const w = mount(SortBar, { props: { label: 'Sort:', options, modelValue: 'date' } })
        await w.findAll('button')[1].trigger('click')
        expect(w.emitted('update:modelValue')).toBeTruthy()
        expect(w.emitted('update:modelValue')[0]).toEqual(['size'])
    })

    it('highlights only the active option', () => {
        const w = mount(SortBar, { props: { label: 'Sort:', options, modelValue: 'size' } })
        const btns = w.findAll('button')
        expect(btns[1].classes()).toContain('text-accent-400')
        expect(btns[0].classes()).not.toContain('text-accent-400')
    })

    it('renders pipe separators between options', () => {
        const w = mount(SortBar, { props: { label: 'Sort:', options, modelValue: 'date' } })
        // n options → n-1 separators
        expect(w.findAll('.text-surface-600')).toHaveLength(options.length - 1)
    })
})
