import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '../locales/en.json'
import UploadCard from '../components/UploadCard.vue'

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

const baseUpload = {
    id: 'upload-1',
    createdAt: '2026-05-04T10:00:00Z',
    expireAt: '2099-01-01T00:00:00Z', // avoid "Never" leaking from the expiry row
    files: [],
    downloadCount: 0,
    downloadedBytes: 0,
    lastDownloadedAt: null,
}

function mountCard(upload = {}) {
    return mount(UploadCard, {
        global: { plugins: [i18n] },
        props: { upload: { ...baseUpload, ...upload } },
    })
}

describe('UploadCard zero-download consistency', () => {
    it('shows "Never" for the last-download row instead of hiding it (matches FileRow/DownloadSidebar)', () => {
        const w = mountCard()
        expect(w.text()).toContain(en.uploadCard.lastDownload)
        expect(w.text()).toContain('Never')
    })

    it('shows the formatted date once a download has been recorded', () => {
        const w = mountCard({ downloadCount: 2, lastDownloadedAt: '2026-05-04T10:30:00Z' })
        expect(w.text()).not.toContain('Never')
        expect(w.text()).toContain('2')
    })

    it('hides both download rows when downloadCount is not present on the upload at all', () => {
        const upload = { ...baseUpload }
        delete upload.downloadCount
        delete upload.lastDownloadedAt
        const w = mount(UploadCard, { global: { plugins: [i18n] }, props: { upload } })
        expect(w.text()).not.toContain(en.uploadCard.downloads)
        expect(w.text()).not.toContain(en.uploadCard.lastDownload)
    })
})

describe('UploadCard total size and downloaded data', () => {
    it('shows the humanized total size, summed client-side from the upload files', () => {
        const w = mountCard({
            files: [
                { id: 'f1', fileName: 'a.txt', fileSize: 1000, status: 'uploaded' },
                { id: 'f2', fileName: 'b.txt', fileSize: 500, status: 'uploaded' },
            ],
        })
        expect(w.text()).toContain(en.uploadCard.totalSize)
        expect(w.text()).toContain('1.50 KB')
    })

    it('shows "0 B" total size for an upload with no files', () => {
        const w = mountCard({ files: [] })
        expect(w.text()).toContain(en.uploadCard.totalSize)
        expect(w.text()).toContain('0 B')
    })

    it('shows the humanized downloaded data for an owner/admin fixture (downloadedBytes present)', () => {
        const w = mountCard({ downloadCount: 3, downloadedBytes: 2_500_000, lastDownloadedAt: '2026-05-04T10:30:00Z' })
        expect(w.text()).toContain(en.uploadCard.downloadedData)
        expect(w.text()).toContain('2.50 MB')
    })

    it('hides the downloaded-data row for a non-owner fixture (downloadedBytes absent)', () => {
        const upload = { ...baseUpload }
        delete upload.downloadedBytes
        const w = mount(UploadCard, { global: { plugins: [i18n] }, props: { upload } })
        expect(w.text()).not.toContain(en.uploadCard.downloadedData)
    })
})
