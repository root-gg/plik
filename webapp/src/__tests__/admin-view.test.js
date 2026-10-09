import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '../locales/en.json'
import { auth } from '../authStore.js'

// AdminView pulls in vue-router (useRouter/useRoute) and the whole api.js
// surface its stats/users/uploads tabs need, so every function it imports
// from api.js must have a mock or the component throws on mount.
//
// route.params.tab is read once at setup (a plain, non-reactive mock object,
// unlike the real reactive route), so it only matters at mount time — tests
// that need to start on a tab other than 'stats' set `routeState.tab` before
// mounting rather than trying to simulate a live in-app tab switch.
const { routerPush, routerReplace, routeState } = vi.hoisted(() => ({
    routerPush: vi.fn().mockResolvedValue(undefined),
    routerReplace: vi.fn().mockResolvedValue(undefined),
    routeState: { tab: 'stats', query: {} },
}))
vi.mock('vue-router', () => ({
    useRouter: () => ({ push: routerPush, replace: routerReplace }),
    useRoute: () => ({ params: { tab: routeState.tab }, query: routeState.query }),
}))

vi.mock('../api.js', () => ({
    getServerStats: vi.fn(),
    getServerActivityDaily: vi.fn().mockResolvedValue([]),
    getAdminUsers: vi.fn().mockResolvedValue({ results: [], after: null, total: 0 }),
    getAdminUploads: vi.fn().mockResolvedValue({ results: [], after: null, total: 0 }),
    searchUsers: vi.fn().mockResolvedValue([]),
    createUser: vi.fn(),
    deleteUser: vi.fn(),
    updateUser: vi.fn(),
    removeUpload: vi.fn(),
    getVersion: vi.fn().mockResolvedValue({ version: '1.0.0', goVersion: 'go1.24', isRelease: true, isMint: true }),
    getTrendingUploads: vi.fn().mockResolvedValue([]),
    getTrendingFiles: vi.fn().mockResolvedValue([]),
}))

// Import after the mocks so AdminView (and this file) resolve the mocked module.
import AdminView from '../views/AdminView.vue'
import {
    getServerStats, getTrendingUploads, getTrendingFiles, getAdminUsers, getAdminUploads,
} from '../api.js'

beforeEach(() => {
    routeState.tab = 'stats'
    routeState.query = {}
})

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

// Canonical GetServerStatistics shape (docs/reference/api.md), current vs
// lifetime buckets distinct so both periods are provably rendered.
const serverStatsFixture = {
    users: 10, uploads: 20, files: 30, totalSize: 1000,
    anonymousTotalSize: 100,
    lifetimeUsers: 12,
    anonymousUsage: { lifetime: { totalSize: 150 } },
    usage: {
        startedAt: '2026-05-04T10:00:00Z',
        downloads: { total: 42, bytes: 1048576, today: 3, last7Days: 10, last30Days: 40 },
        uploads: { total: 30, bytes: 2097152, today: 2, last7Days: 6, last30Days: 20 },
        current: {
            uploads: 1, files: 2, totalSize: 42,
            features: {}, ttl: {}, fileSizes: {},
        },
        lifetime: {
            uploads: 5, files: 9, totalSize: 100,
            features: {}, ttl: {}, fileSizes: {},
        },
    },
}

function mountAdmin() {
    auth.user = { id: 'admin-1', login: 'admin', admin: true, provider: 'local' }
    auth.originalUser = null
    return mount(AdminView, { global: { plugins: [i18n] } })
}

describe('AdminView trending window vocabulary', () => {
    beforeEach(() => {
        getServerStats.mockResolvedValue(serverStatsFixture)
        getTrendingUploads.mockResolvedValue([])
        getTrendingFiles.mockResolvedValue([])
    })

    it('uses the same Today / 7 days / 30 days / Lifetime vocabulary as the Activity tiles, not "All time"', async () => {
        const w = mountAdmin()
        await flushPromises()

        const trendingCard = w.findAll('.glass-card').find((c) => c.text().includes('Trending Uploads'))
        expect(trendingCard).toBeTruthy()
        const chipLabels = trendingCard.findAll('button').map((b) => b.text())
        expect(chipLabels).toContain('Today')
        expect(chipLabels).toContain('7 days')
        expect(chipLabels).toContain('30 days')
        expect(chipLabels).toContain('Lifetime')
        expect(chipLabels).not.toContain('All time')
    })
})

describe('AdminView trending uploads human-first hierarchy', () => {
    beforeEach(() => {
        getServerStats.mockResolvedValue(serverStatsFixture)
        getTrendingFiles.mockResolvedValue([])
    })

    it('leads with the comment when present, demotes the ID to a monospace subtitle', async () => {
        getTrendingUploads.mockResolvedValue([{
            id: 'UiMK1eYvlVH2l1Mb', downloadCount: 12, files: 3,
            comments: 'quarterly report', user: 'local:alice', lastDownloadedAt: '2026-05-04T10:00:00Z',
        }])
        const w = mountAdmin()
        await flushPromises()

        const text = w.text()
        expect(text).toContain('quarterly report')
        // The ID is still present (demoted subtitle), still a clickable button.
        const idButton = w.findAll('button').find((b) => b.text() === 'UiMK1eYvlVH2l1Mb')
        expect(idButton).toBeTruthy()
        // Owner is still shown in the subtitle even though the headline is the comment.
        const ownerButton = w.findAll('button').find((b) => b.text() === 'local:alice')
        expect(ownerButton).toBeTruthy()
    })

    it('falls back to "Upload by <owner>" when there is no comment', async () => {
        getTrendingUploads.mockResolvedValue([{
            id: 'abc123', downloadCount: 4, files: 1,
            comments: '', user: 'local:bob', lastDownloadedAt: '2026-05-04T10:00:00Z',
        }])
        const w = mountAdmin()
        await flushPromises()

        expect(w.text()).toContain('Upload by local:bob')
    })

    it('falls back to "Anonymous upload" when there is no comment and no owner', async () => {
        getTrendingUploads.mockResolvedValue([{
            id: 'xyz789', downloadCount: 1, files: 1,
            comments: '', user: '', lastDownloadedAt: '2026-05-04T10:00:00Z',
        }])
        const w = mountAdmin()
        await flushPromises()

        expect(w.text()).toContain('Anonymous upload')
    })

    it('keeps the ID button\'s click target navigating to the upload (both headline and subtitle)', async () => {
        getTrendingUploads.mockResolvedValue([{
            id: 'nav-target-id', downloadCount: 4, files: 1,
            comments: '', user: '', lastDownloadedAt: '2026-05-04T10:00:00Z',
        }])
        const w = mountAdmin()
        await flushPromises()

        const idButton = w.findAll('button').find((b) => b.text() === 'nav-target-id')
        await idButton.trigger('click')
        expect(routerPush).toHaveBeenCalledWith({ path: '/', query: { id: 'nav-target-id' } })
    })
})

describe('AdminView same-value setter guards', () => {
    beforeEach(() => {
        getServerStats.mockResolvedValue(serverStatsFixture)
        getTrendingUploads.mockResolvedValue([])
        getTrendingFiles.mockResolvedValue([])
        routerPush.mockClear()
        routerReplace.mockClear()
    })

    it('re-clicking the active ("All") provider filter on the Users tab is a no-op (no redundant navigation/reload)', async () => {
        routeState.tab = 'users'
        const w = mountAdmin()
        await flushPromises()

        getAdminUsers.mockClear()
        routerReplace.mockClear()

        // "All" (empty-string provider filter) is already active by default.
        const allProviderButton = w.findAll('button').find((b) => b.text() === 'All')
        expect(allProviderButton).toBeTruthy()
        await allProviderButton.trigger('click')
        await flushPromises()

        expect(routerReplace).not.toHaveBeenCalled()
        expect(getAdminUsers).not.toHaveBeenCalled()
    })

    it('re-clicking the active sort option on the Uploads tab is a no-op (no redundant navigation/reload)', async () => {
        routeState.tab = 'uploads'
        const w = mountAdmin()
        await flushPromises()

        getAdminUploads.mockClear()
        routerReplace.mockClear()

        // "Date" is the default/active uploads sort.
        const dateSortButton = w.findAll('button').find((b) => b.text() === 'Date')
        expect(dateSortButton).toBeTruthy()
        await dateSortButton.trigger('click')
        await flushPromises()

        expect(routerReplace).not.toHaveBeenCalled()
        expect(getAdminUploads).not.toHaveBeenCalled()
    })
})
