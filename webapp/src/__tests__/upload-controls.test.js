import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '../locales/en.json'
import UploadControls from '../components/UploadControls.vue'

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

function mountControls(sortBy = 'date') {
    return mount(UploadControls, {
        global: { plugins: [i18n] },
        props: {
            sortBy,
            sortOrder: 'desc',
            badgeFilters: { oneShot: false, removable: false, stream: false, extendTTL: false, password: false, e2ee: false },
        },
    })
}

describe('UploadControls downloaded-data sort option', () => {
    it('renders the "Downloaded data" sort button', () => {
        const w = mountControls()
        expect(w.text()).toContain(en.uploadControls.downloadedData)
    })

    it('emits update:sort-by with "downloadedBytes" when clicked', async () => {
        const w = mountControls()
        const button = w.findAll('button').find(b => b.text() === en.uploadControls.downloadedData)
        expect(button).toBeTruthy()
        await button.trigger('click')
        expect(w.emitted('update:sort-by')).toBeTruthy()
        expect(w.emitted('update:sort-by')[0]).toEqual(['downloadedBytes'])
    })

    it('highlights the downloaded-data button when active', () => {
        const w = mountControls('downloadedBytes')
        const button = w.findAll('button').find(b => b.text() === en.uploadControls.downloadedData)
        expect(button.classes()).toContain('text-accent-400')
    })
})
