import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '../locales/en.json'
import FileRow from '../components/FileRow.vue'

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

const zeroDownloadFile = {
    id: 'file-1',
    fileName: 'report.txt',
    fileType: 'text/plain',
    fileSize: 1234,
    status: 'uploaded',
    downloadCount: 0,
    lastDownloadedAt: null,
}

const downloadedFile = {
    ...zeroDownloadFile,
    id: 'file-2',
    downloadCount: 3,
    lastDownloadedAt: '2026-05-04T10:30:00Z',
}

function mountRow(props = {}) {
    return mount(FileRow, {
        global: { plugins: [i18n] },
        props: {
            file: zeroDownloadFile,
            mode: 'download',
            uploadId: 'upload-1',
            showDownloadStats: true,
            ...props,
        },
    })
}

describe('FileRow download stats', () => {
    it('renders "Last download:" (not the old "Last:") for the label', async () => {
        const w = mountRow()
        await w.find('[title="Toggle details"]').trigger('click')
        expect(w.text()).toContain('Last download:')
        expect(w.text()).not.toContain('Last:')
    })

    it('shows "Never" for the zero-download fixture instead of hiding the row', async () => {
        const w = mountRow()
        await w.find('[title="Toggle details"]').trigger('click')
        expect(w.text()).toContain('Downloads:')
        expect(w.text()).toContain('Never')
    })

    it('renders the formatted date when the file has been downloaded', async () => {
        const w = mountRow({ file: downloadedFile })
        await w.find('[title="Toggle details"]').trigger('click')
        expect(w.text()).not.toContain('Never')
        expect(w.text()).toContain('3') // downloadCount
    })

    it('renders the shared download-count help tooltip next to the downloads value', async () => {
        const w = mountRow()
        await w.find('[title="Toggle details"]').trigger('click')
        expect(w.find('.setting-help').exists()).toBe(true)
        expect(w.find('.setting-tooltip').text()).toBe(en.common.downloadCountHelp)
    })

    it('does not render download stats when showDownloadStats is false', async () => {
        const w = mountRow({ showDownloadStats: false })
        await w.find('[title="Toggle details"]').trigger('click')
        expect(w.text()).not.toContain('Downloads:')
        expect(w.find('.setting-help').exists()).toBe(false)
    })
})
