import { describe, it, expect, vi } from 'vitest'
import { nextTick } from 'vue'
import { useTrending } from '../composables/useTrending.js'

function makeRouter() {
    return { push: vi.fn().mockResolvedValue(undefined) }
}

describe('useTrending', () => {
    it('starts with the shared default window/sort state and empty lists', () => {
        const t = useTrending(makeRouter(), { fetchUploads: vi.fn().mockResolvedValue([]) })
        expect(t.trendingWindow.value).toBe('7d')
        expect(t.trendingSort.value).toBe('downloads')
        expect(t.trendingUploads.value).toEqual([])
        expect(t.trendingFiles.value).toEqual([])
        expect(t.trendingLoading.value).toBe(false)
    })

    it('admin scope (fetchFiles present) loads uploads AND files in parallel with the window/sort params', async () => {
        const fetchUploads = vi.fn().mockResolvedValue([{ id: 'u1' }])
        const fetchFiles = vi.fn().mockResolvedValue([{ id: 'f1' }])
        const t = useTrending(makeRouter(), { fetchUploads, fetchFiles })

        await t.loadTrending()

        expect(fetchUploads).toHaveBeenCalledWith({ window: '7d', sort: 'downloads', limit: 8 })
        expect(fetchFiles).toHaveBeenCalledWith({ window: '7d', limit: 8 })
        expect(t.trendingUploads.value).toEqual([{ id: 'u1' }])
        expect(t.trendingFiles.value).toEqual([{ id: 'f1' }])
        expect(t.trendingLoading.value).toBe(false)
    })

    it('self scope (no fetchFiles) loads only uploads and never touches trendingFiles', async () => {
        const fetchUploads = vi.fn().mockResolvedValue([{ id: 'u1' }])
        const t = useTrending(makeRouter(), { fetchUploads })

        await t.loadTrending()

        expect(fetchUploads).toHaveBeenCalledWith({ window: '7d', sort: 'downloads', limit: 8 })
        expect(t.trendingUploads.value).toEqual([{ id: 'u1' }])
        expect(t.trendingFiles.value).toEqual([])
    })

    it('changeTrendingWindow updates state and reloads (uploads + files) with the new window', async () => {
        const fetchUploads = vi.fn().mockResolvedValue([])
        const fetchFiles = vi.fn().mockResolvedValue([])
        const t = useTrending(makeRouter(), { fetchUploads, fetchFiles })

        t.changeTrendingWindow('30d')
        expect(t.trendingWindow.value).toBe('30d')
        await nextTick()
        await Promise.resolve()

        expect(fetchUploads).toHaveBeenCalledWith({ window: '30d', sort: 'downloads', limit: 8 })
        expect(fetchFiles).toHaveBeenCalledWith({ window: '30d', limit: 8 })
    })

    it('changeTrendingWindow is a no-op when the value is unchanged (same-value guard)', async () => {
        const fetchUploads = vi.fn().mockResolvedValue([])
        const t = useTrending(makeRouter(), { fetchUploads })

        t.changeTrendingWindow('7d') // already the default
        await Promise.resolve()

        expect(fetchUploads).not.toHaveBeenCalled()
    })

    it('changeTrendingSort re-fetches ONLY uploads (not files) and never blanks the loading flag', async () => {
        const fetchUploads = vi.fn().mockResolvedValue([{ id: 'u-sorted' }])
        const fetchFiles = vi.fn().mockResolvedValue([])
        const t = useTrending(makeRouter(), { fetchUploads, fetchFiles })

        await t.changeTrendingSort('downloadedBytes')

        expect(t.trendingSort.value).toBe('downloadedBytes')
        expect(fetchUploads).toHaveBeenCalledWith({ window: '7d', sort: 'downloadedBytes', limit: 8 })
        // Files are NOT re-fetched on a pure metric re-sort.
        expect(fetchFiles).not.toHaveBeenCalled()
        expect(t.trendingUploads.value).toEqual([{ id: 'u-sorted' }])
        // trendingLoading is deliberately left untouched by a re-sort.
        expect(t.trendingLoading.value).toBe(false)
    })

    it('changeTrendingSort is a no-op when the value is unchanged (same-value guard)', async () => {
        const fetchUploads = vi.fn().mockResolvedValue([])
        const t = useTrending(makeRouter(), { fetchUploads })

        await t.changeTrendingSort('downloads') // already the default
        expect(fetchUploads).not.toHaveBeenCalled()
    })

    it('invokes onError (and leaves loading false) when a fetch rejects', async () => {
        const onError = vi.fn()
        const boom = new Error('boom')
        const t = useTrending(makeRouter(), { fetchUploads: vi.fn().mockRejectedValue(boom), onError })

        await t.loadTrending()

        expect(onError).toHaveBeenCalledWith(boom)
        expect(t.trendingLoading.value).toBe(false)
    })

    it('openUpload pushes the upload route (id + optional file), and is a no-op without an id', () => {
        const router = makeRouter()
        const t = useTrending(router, { fetchUploads: vi.fn() })

        t.openUpload('upload-1')
        expect(router.push).toHaveBeenCalledWith({ path: '/', query: { id: 'upload-1' } })

        t.openUpload('upload-1', 'file-9')
        expect(router.push).toHaveBeenCalledWith({ path: '/', query: { id: 'upload-1', file: 'file-9' } })

        router.push.mockClear()
        t.openUpload('')
        expect(router.push).not.toHaveBeenCalled()
    })
})
