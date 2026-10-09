import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '../locales/en.json'
import DownloadSidebar from '../components/DownloadSidebar.vue'

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

const baseUpload = {
    id: 'upload-1',
    admin: true,
    expireAt: null,
    files: [],
    downloadCount: 0,
    lastDownloadedAt: null,
}

function mountSidebar(upload = {}) {
    return mount(DownloadSidebar, {
        global: { plugins: [i18n] },
        props: { upload: { ...baseUpload, ...upload } },
    })
}

describe('DownloadSidebar download stats (admin view)', () => {
    it('shows 0 downloads and "Never" for the zero-download fixture', () => {
        const w = mountSidebar()
        expect(w.text()).toContain(en.downloadSidebar.downloads)
        expect(w.text()).toContain('Never')
    })

    it('shows the formatted date once a download has been recorded', () => {
        // expireAt set so "Never expires" (unrelated to download stats) doesn't
        // leak a false-positive "Never" match.
        const w = mountSidebar({ expireAt: '2099-01-01T00:00:00Z', downloadCount: 5, lastDownloadedAt: '2026-05-04T10:30:00Z' })
        expect(w.text()).not.toContain('Never')
        expect(w.text()).toContain('5')
    })

    it('renders the shared download-count help tooltip next to the Downloads label', () => {
        const w = mountSidebar()
        expect(w.find('.setting-help').exists()).toBe(true)
        expect(w.find('.setting-tooltip').text()).toBe(en.common.downloadCountHelp)
    })

    it('hides the download-stats block entirely for non-admin viewers', () => {
        const w = mountSidebar({ admin: false })
        expect(w.find('.setting-help').exists()).toBe(false)
    })
})
