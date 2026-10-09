import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '../locales/en.json'

// DownloadView pulls in vue-router and the api.js surface it needs to fetch
// upload metadata and file content. Only getUpload is under test (it's the
// call whose result drives the sidebar's download counters); every other
// api.js export DownloadView imports is preserved via importActual so pure
// helpers like getFileURL keep working unmocked.
vi.mock('vue-router', () => ({
    useRouter: () => ({
        push: vi.fn().mockResolvedValue(undefined),
        replace: vi.fn().mockResolvedValue(undefined),
        resolve: vi.fn().mockReturnValue({ href: '' }),
        currentRoute: { value: { query: {} } },
    }),
    useRoute: () => ({ query: {} }),
}))

vi.mock('../api.js', async (importOriginal) => {
    const actual = await importOriginal()
    return { ...actual, getUpload: vi.fn() }
})

import DownloadView from '../views/DownloadView.vue'
import { getUpload } from '../api.js'

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

// Single viewable text file — triggers DownloadView's auto-preview watcher
// (exactly one file, uploaded, viewable, not oneShot/stream).
const textFile = {
    id: 'file-1',
    fileName: 'note.txt',
    fileType: 'text/plain',
    isText: true,
    fileSize: 20,
    status: 'uploaded',
    downloadCount: 0,
    lastDownloadedAt: null,
}

const uploadBeforePreview = {
    id: 'upload-1',
    admin: true,
    oneShot: false,
    stream: false,
    e2ee: false,
    files: [textFile],
    downloadCount: 0,
    lastDownloadedAt: null,
}

const uploadAfterPreview = {
    ...uploadBeforePreview,
    downloadCount: 1,
    lastDownloadedAt: '2026-07-06T12:00:00Z',
    files: [{ ...textFile, downloadCount: 1, lastDownloadedAt: '2026-07-06T12:00:00Z' }],
}

// Single image file — also auto-previewed, but rendered via <img :src> so the
// browser issues the counted GET itself (no fetch promise for the component
// to await). The stats refresh hooks the element's load event instead. Video
// and audio go through the same onMediaPreviewLoaded handler (wired to their
// existing loadedmetadata handler), so the image case pins the whole path.
const imageFile = {
    id: 'file-img',
    fileName: 'photo.png',
    fileType: 'image/png',
    fileSize: 1000,
    status: 'uploaded',
    downloadCount: 0,
    lastDownloadedAt: null,
}

const imageUploadBeforePreview = { ...uploadBeforePreview, files: [imageFile] }

const imageUploadAfterPreview = {
    ...imageUploadBeforePreview,
    downloadCount: 1,
    lastDownloadedAt: '2026-07-06T12:00:00Z',
    files: [{ ...imageFile, downloadCount: 1, lastDownloadedAt: '2026-07-06T12:00:00Z' }],
}

function mockFetchResponse(body) {
    return {
        ok: true,
        status: 200,
        headers: { get: () => 'text/plain; charset=utf-8' },
        text: async () => body,
        arrayBuffer: async () => new TextEncoder().encode(body).buffer,
    }
}

function mountDownload() {
    return mount(DownloadView, {
        props: { id: 'upload-1' },
        global: {
            plugins: [i18n],
            stubs: { CodeEditor: true },
        },
    })
}

describe('DownloadView stale sidebar count fix', () => {
    beforeEach(() => {
        getUpload.mockReset()
        global.fetch = vi.fn().mockResolvedValue(mockFetchResponse('hello world'))
    })

    it('refreshes the sidebar download count after the auto-preview fetch resolves', async () => {
        getUpload
            .mockResolvedValueOnce(uploadBeforePreview) // initial fetchUpload() on mount
            .mockResolvedValueOnce(uploadAfterPreview)  // targeted refresh after the preview fetch

        const w = mountDownload()

        // Mount: fetchUpload() resolves with downloadCount 0.
        await flushPromises()
        // Auto-preview watcher fires -> viewFile() -> fetch() -> refreshDownloadStats().
        await flushPromises()
        await flushPromises()

        expect(getUpload).toHaveBeenCalledTimes(2)
        expect(w.get('p.tabular-nums').text()).toBe('1')
    })

    it('does not stay stale at 0 once the preview has been recorded server-side', async () => {
        getUpload
            .mockResolvedValueOnce(uploadBeforePreview)
            .mockResolvedValueOnce(uploadAfterPreview)

        const w = mountDownload()
        await flushPromises()
        await flushPromises()
        await flushPromises()

        expect(w.get('p.tabular-nums').text()).not.toBe('0')
    })

    it('refreshes the sidebar download count after an image preview loads', async () => {
        getUpload
            .mockResolvedValueOnce(imageUploadBeforePreview) // initial fetchUpload() on mount
            .mockResolvedValueOnce(imageUploadAfterPreview)  // refresh on the img load event

        const w = mountDownload()

        // Mount: fetchUpload() -> auto-preview watcher -> viewFile() renders <img>.
        await flushPromises()
        await flushPromises()

        // No content fetch happens for media — the browser GETs the src itself.
        expect(global.fetch).not.toHaveBeenCalled()
        expect(getUpload).toHaveBeenCalledTimes(1)
        expect(w.get('p.tabular-nums').text()).toBe('0')

        // jsdom never actually loads the image; fire the load event manually.
        await w.get('img[alt="photo.png"]').trigger('load')
        await flushPromises()

        expect(getUpload).toHaveBeenCalledTimes(2)
        expect(w.get('p.tabular-nums').text()).toBe('1')
    })

    it('refreshes at most once per view even if the media element fires load again', async () => {
        getUpload
            .mockResolvedValueOnce(imageUploadBeforePreview)
            .mockResolvedValue(imageUploadAfterPreview)

        const w = mountDownload()
        await flushPromises()
        await flushPromises()

        const img = w.get('img[alt="photo.png"]')
        await img.trigger('load')
        await flushPromises()
        await img.trigger('load') // e.g. re-fire — must not refetch again
        await flushPromises()

        expect(getUpload).toHaveBeenCalledTimes(2) // mount + one refresh only
    })
})
