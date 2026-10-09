import { ref } from 'vue'

/**
 * Shared trending-panel state/behavior for AdminView (server-wide, cross-user)
 * and HomeView (self-scoped, this user's own uploads only). Both callers wire
 * the same window/sort chips into a TrendingPanel and navigate the same way
 * when a row is opened — only which API fetcher(s) back the data differ.
 *
 * `fetchUploads({window, sort, limit})` is required; `fetchFiles({window,
 * limit})` is optional — pass it only for a scope that also has a Trending
 * Files list (server-wide today; the self-scoped endpoint has no files
 * variant). When absent, `trendingFiles` stays an empty ref the caller can
 * simply not use. `onError(err)` is called on any fetch failure so each
 * caller can surface its own scope-appropriate message.
 */
export function useTrending(router, { fetchUploads, fetchFiles = null, onError = null } = {}) {
    const trendingWindow = ref('7d')
    const trendingSort = ref('downloads')
    const trendingUploads = ref([])
    const trendingFiles = ref([])
    const trendingLoading = ref(false)

    async function loadTrending() {
        trendingLoading.value = true
        try {
            if (fetchFiles) {
                const [uploadsData, filesData] = await Promise.all([
                    fetchUploads({ window: trendingWindow.value, sort: trendingSort.value, limit: 8 }),
                    fetchFiles({ window: trendingWindow.value, limit: 8 }),
                ])
                trendingUploads.value = uploadsData || []
                trendingFiles.value = filesData || []
            } else {
                trendingUploads.value = await fetchUploads({ window: trendingWindow.value, sort: trendingSort.value, limit: 8 }) || []
            }
        } catch (err) {
            onError?.(err)
        } finally {
            trendingLoading.value = false
        }
    }

    function changeTrendingWindow(value) {
        if (value === trendingWindow.value) return
        trendingWindow.value = value
        loadTrending()
    }

    // Only the uploads side is re-fetched: the metric toggle is uploads-only
    // (Trending Files has no sort dimension) and the window is unchanged.
    // Deliberately does NOT toggle trendingLoading — that would also blank an
    // untouched Trending Files column for a purely uploads-side re-sort.
    async function changeTrendingSort(value) {
        if (value === trendingSort.value) return
        trendingSort.value = value
        try {
            trendingUploads.value = await fetchUploads({ window: trendingWindow.value, sort: trendingSort.value, limit: 8 }) || []
        } catch (err) {
            onError?.(err)
        }
    }

    // Navigate to the upload's detail/download view — identical for every
    // scope that uses this composable.
    function openUpload(uploadId, fileId = '') {
        if (!uploadId) return
        const query = { id: uploadId }
        if (fileId) query.file = fileId
        router.push({ path: '/', query })
    }

    return {
        trendingWindow, trendingSort, trendingUploads, trendingFiles, trendingLoading,
        loadTrending, changeTrendingWindow, changeTrendingSort, openUpload,
    }
}
