import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '../locales/en.json'
import { auth } from '../authStore.js'

// HomeView pulls in vue-router (useRouter/useRoute) and the whole api.js
// surface it needs for its three tabs. Only the stats tab is under test here,
// but onMounted() also kicks off the tokens fetch (feature_api_tokens
// defaults to 'enabled'), so every function HomeView imports from api.js must
// have a mock or the component throws on mount.
vi.mock('vue-router', () => ({
    useRouter: () => ({ push: vi.fn().mockResolvedValue(undefined), replace: vi.fn().mockResolvedValue(undefined) }),
    useRoute: () => ({ params: { tab: 'stats' }, query: {} }),
}))

vi.mock('../api.js', () => ({
    getUserUploads: vi.fn().mockResolvedValue({ results: [], after: null, total: 0 }),
    deleteUserUploads: vi.fn(),
    removeUpload: vi.fn(),
    getUserTokens: vi.fn().mockResolvedValue({ results: [], after: null }),
    createToken: vi.fn(),
    revokeToken: vi.fn(),
    deleteAccount: vi.fn(),
    updateUser: vi.fn(),
    getUserStatistics: vi.fn(),
    getUserActivityDaily: vi.fn().mockResolvedValue([]),
    getUserTrendingUploads: vi.fn().mockResolvedValue([]),
}))

// Import after the mocks so HomeView (and this file) resolve the mocked module.
import HomeView from '../views/HomeView.vue'
import TrendingPanel from '../components/TrendingPanel.vue'
import { getUserStatistics, getUserActivityDaily, getUserTrendingUploads } from '../api.js'

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

// Canonical `/me/stats` shape copied from docs/reference/api.md, with current
// vs lifetime buckets distinct so both periods are provably rendered.
const meStatsFixture = {
    uploads: 1,
    files: 2,
    totalSize: 42,
    usage: {
        startedAt: '2026-05-04T10:00:00Z',
        lastUploadAt: '2026-05-04T10:30:00Z',
        downloads: { total: 42, bytes: 1048576, today: 3, last7Days: 10, last30Days: 40 },
        uploads: { total: 30, bytes: 2097152, today: 2, last7Days: 6, last30Days: 20 },
        current: {
            uploads: 1, files: 2, totalSize: 42,
            features: { passwordUploads: 0, removableUploads: 0, oneShotUploads: 0, streamUploads: 0, extendTTLUploads: 0, e2eeUploads: 0, commentUploads: 0 },
            ttl: { noneUploads: 0, lessThan1HourUploads: 0, oneHourToOneDayUploads: 0, oneDayToSevenDaysUploads: 0, sevenDaysTo30DaysUploads: 0, greaterThan30DaysUploads: 0 },
            fileSizes: { lessThan1MBFiles: 2, oneMBTo10MBFiles: 0, tenMBTo100MBFiles: 0, hundredMBTo1GBFiles: 0, oneGBTo10GBFiles: 0, tenGBTo100GBFiles: 0, greaterThan100GBFiles: 0 },
        },
        lifetime: {
            uploads: 5, files: 9, totalSize: 100,
            features: { passwordUploads: 1, removableUploads: 0, oneShotUploads: 2, streamUploads: 0, extendTTLUploads: 0, e2eeUploads: 0, commentUploads: 3 },
            ttl: { noneUploads: 1, lessThan1HourUploads: 0, oneHourToOneDayUploads: 2, oneDayToSevenDaysUploads: 0, sevenDaysTo30DaysUploads: 0, greaterThan30DaysUploads: 2 },
            fileSizes: { lessThan1MBFiles: 4, oneMBTo10MBFiles: 3, tenMBTo100MBFiles: 2, hundredMBTo1GBFiles: 0, oneGBTo10GBFiles: 0, tenGBTo100GBFiles: 0, greaterThan100GBFiles: 0 },
        },
    },
}

const emptyMeStatsFixture = {
    uploads: 0,
    files: 0,
    totalSize: 0,
    usage: {
        downloads: { total: 0, bytes: 0, today: 0, last7Days: 0, last30Days: 0 },
        uploads: { total: 0, bytes: 0, today: 0, last7Days: 0, last30Days: 0 },
        current: { uploads: 0, files: 0, totalSize: 0, features: {}, ttl: {}, fileSizes: {} },
        lifetime: { uploads: 0, files: 0, totalSize: 0, features: {}, ttl: {}, fileSizes: {} },
    },
}

function mountHome() {
    return mount(HomeView, {
        global: {
            plugins: [i18n],
            // StatsUsagePanel's empty state renders a real <router-link
            // to="/">; 'vue-router' is fully mocked above (useRouter/useRoute
            // only), so stub it out rather than installing a real router.
            stubs: { RouterLink: { template: '<a><slot /></a>' } },
        },
    })
}

describe('HomeView stats tab', () => {
    beforeEach(() => {
        auth.user = { id: 'user-1', login: 'alice', name: 'Alice', admin: false, provider: 'local' }
    })

    it('renders the shared panel with user tiles + downloads row from the canonical /me/stats fixture', async () => {
        getUserStatistics.mockResolvedValue(meStatsFixture)
        const w = mountHome()
        await flushPromises()

        // Panel title is the user-specific title (not Admin's "Server Statistics")
        expect(w.text()).toContain('User Statistics')
        expect(w.text()).not.toContain('Server Statistics')

        // Current/lifetime periods, since-date printed once
        expect(w.text()).toContain('Current Usage')
        expect(w.text()).toMatch(/Lifetime Usage \(since /)
        expect((w.text().match(/\(since /g) || []).length).toBe(1)

        // Activity card window tiles (default metric = downloads; user scope)
        expect(w.text()).toContain('Activity')
        expect(w.text()).toContain('Today')
        expect(w.text()).toContain('7 days')
        expect(w.text()).toContain('30 days')
        expect(w.text()).toContain('Lifetime')
        expect(w.text()).toContain('42') // downloads.total → Lifetime tile

        // Six distribution cards (feature / TTL / file-size x current / lifetime)
        expect(w.text()).toContain('Current Feature Usage')
        expect(w.text()).toContain('Lifetime Feature Usage')
        expect(w.text()).toContain('Current TTL Distribution')
        expect(w.text()).toContain('Lifetime TTL Distribution')
        expect(w.text()).toContain('Current file size distribution')
        expect(w.text()).toContain('Lifetime file size distribution')

        // Privacy: no admin-only server-scope markup leaks onto Home
        expect(w.text()).not.toContain('storage split')
        expect(w.text()).not.toContain('Anonymous')
    })

    it('shows the friendly empty-state panel (not six zero-filled cards) from an empty-usage fixture (fresh user)', async () => {
        getUserStatistics.mockResolvedValue(emptyMeStatsFixture)
        const w = mountHome()
        await flushPromises()

        expect(w.text()).toContain('User Statistics')
        expect(w.text()).not.toContain('NaN')
        // No startedAt -> falls back to the no-since title, exactly once
        expect(w.text()).toContain('Lifetime Usage')
        expect(w.text()).not.toMatch(/Lifetime Usage \(since /)
        // An all-zero lifetime on Home replaces the six distribution cards
        // with one encouraging "get started" panel.
        expect(w.text()).toContain('Upload your first file to see your usage stats here.')
        expect(w.text()).toContain('Upload a file')
        expect(w.text()).not.toContain('Current Feature Usage')
        expect(w.text()).not.toContain('Lifetime file size distribution')
    })

    it('shows the "no stats available" fallback when the fetch fails', async () => {
        getUserStatistics.mockRejectedValue(new Error('boom'))
        getUserActivityDaily.mockResolvedValue([])
        const w = mountHome()
        await flushPromises()

        expect(w.text()).toContain('No stats available')
        expect(w.text()).not.toContain('User Statistics')
    })

    it('renders the daily downloads chart from the daily-series fetch', async () => {
        getUserStatistics.mockResolvedValue(meStatsFixture)
        getUserActivityDaily.mockResolvedValue([
            { day: '2026-05-01', downloads: 3, downloadedBytes: 1024, uploads: 1, uploadedBytes: 2048 },
            { day: '2026-05-02', downloads: 7, downloadedBytes: 4096, uploads: 2, uploadedBytes: 8192 },
        ])
        const w = mountHome()
        await flushPromises()

        expect(w.find('[data-testid="activity-chart"]').exists()).toBe(true)
        expect(w.text()).toContain('Activity')
        expect(w.text()).toContain('UTC days')
    })

    it('omits the chart when the daily-series fetch fails (dashboard survives)', async () => {
        getUserStatistics.mockResolvedValue(meStatsFixture)
        getUserActivityDaily.mockRejectedValue(new Error('boom'))
        const w = mountHome()
        await flushPromises()

        // Stats still render, chart section is absent
        expect(w.text()).toContain('User Statistics')
        expect(w.find('[data-testid="activity-chart"]').exists()).toBe(false)
    })
})

describe('HomeView self-scoped trending panel', () => {
    beforeEach(() => {
        auth.user = { id: 'user-1', login: 'alice', name: 'Alice', admin: false, provider: 'local' }
    })

    it('renders the self-scoped panel (no Trending Files, "Your Trending Uploads" title) with the user\'s own uploads', async () => {
        getUserStatistics.mockResolvedValue(meStatsFixture)
        getUserTrendingUploads.mockResolvedValue([
            { id: 'upload-1', comments: 'my report', user: 'user-1', files: 1, downloadCount: 12, downloadedBytes: 4096, lastDownloadedAt: '2026-05-04T10:00:00Z' },
        ])
        const w = mountHome()
        await flushPromises()

        expect(w.text()).toContain('Your Trending Uploads')
        expect(w.text()).toContain('my report')
        expect(w.text()).not.toContain('Trending Files')
        expect(getUserTrendingUploads).toHaveBeenCalled()
    })

    it('hides the trending panel entirely for a genuinely fresh account (no second wall-of-zeros next to the empty state)', async () => {
        getUserStatistics.mockResolvedValue(emptyMeStatsFixture)
        const w = mountHome()
        await flushPromises()

        // The empty-state CTA is already asserted elsewhere; here we only
        // check trending does not ALSO render its own (redundant) empty state.
        expect(w.text()).not.toContain('Your Trending Uploads')
        expect(w.text()).not.toContain('No trending uploads yet')
    })

    it('shows the panel\'s own empty state when the user has activity but nothing trending yet', async () => {
        getUserStatistics.mockResolvedValue(meStatsFixture)
        getUserTrendingUploads.mockResolvedValue([])
        const w = mountHome()
        await flushPromises()

        expect(w.text()).toContain('Your Trending Uploads')
        expect(w.text()).toContain('No trending uploads yet')
    })

    it('the metric toggle reorders/emphasizes and re-fetches with the new sort', async () => {
        // StatsUsagePanel's own Activity selector also has a "Downloaded data"
        // button — scope the lookup to the TrendingPanel instance specifically
        // (mirrors how the admin e2e test scopes to its own trending card) so
        // this test can't accidentally click the wrong toggle.
        getUserStatistics.mockResolvedValue(meStatsFixture)
        getUserTrendingUploads.mockResolvedValue([
            { id: 'upload-1', comments: 'my report', user: 'user-1', files: 1, downloadCount: 12, downloadedBytes: 4096, lastDownloadedAt: '2026-05-04T10:00:00Z' },
        ])
        const w = mountHome()
        await flushPromises()

        const panel = w.findComponent(TrendingPanel)
        expect(panel.exists()).toBe(true)
        const bytesButton = panel.findAll('button').find((b) => b.text() === 'Downloaded data')
        expect(bytesButton).toBeTruthy()
        await bytesButton.trigger('click')
        await flushPromises()

        expect(getUserTrendingUploads).toHaveBeenCalledWith(expect.objectContaining({ sort: 'downloadedBytes' }))
        const text = panel.text()
        expect(text.indexOf('4.10 KB')).toBeLessThan(text.indexOf('12 downloads'))
    })
})
