import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createRouter, createMemoryHistory } from 'vue-router'
import en from '../locales/en.json'
import StatsUsagePanel from '../components/StatsUsagePanel.vue'
import StatTile from '../components/StatTile.vue'
import DistributionCard from '../components/DistributionCard.vue'

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

// Real (minimal) router so the empty-state's <router-link to="/"> resolves
// cleanly instead of warning "Failed to resolve component: router-link".
const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div />' } }],
})

// Canonical usage object with the symmetric downloads/uploads windows so we can
// prove the 4-metric selector drives the window tiles.
const usage = {
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
}

const dailySeries = [
    { day: '2026-05-01', downloads: 3, downloadedBytes: 1024, uploads: 1, uploadedBytes: 500000 },
    { day: '2026-05-02', downloads: 7, downloadedBytes: 4096, uploads: 4, uploadedBytes: 1000000 },
]

const currentTiles = [
    { label: 'Users', value: 12, format: 'count' },
    { label: 'Uploads', value: usage.current.uploads, format: 'count' },
    { label: 'Files', value: usage.current.files, format: 'count' },
    { label: 'Total Size', value: usage.current.totalSize, format: 'size' },
]
const lifetimeTiles = [
    { label: 'Lifetime Users', value: 15, format: 'count' },
    { label: 'Lifetime Uploads', value: usage.lifetime.uploads, format: 'count' },
    { label: 'Lifetime Files', value: usage.lifetime.files, format: 'count' },
    { label: 'Lifetime Size', value: usage.lifetime.totalSize, format: 'size' },
]

function mountPanel(props = {}, slots = {}) {
    return mount(StatsUsagePanel, {
        global: { plugins: [i18n, router] },
        props: { usage, title: 'Server Statistics', currentTiles, lifetimeTiles, ...props },
        slots,
    })
}

// All-zero usage object (fresh scope, nothing has ever happened) for the
// empty-state tests.
const zeroUsage = {
    startedAt: '',
    downloads: { total: 0, bytes: 0, today: 0, last7Days: 0, last30Days: 0 },
    uploads: { total: 0, bytes: 0, today: 0, last7Days: 0, last30Days: 0 },
    current: { uploads: 0, files: 0, totalSize: 0, features: {}, ttl: {}, fileSizes: {} },
    lifetime: { uploads: 0, files: 0, totalSize: 0, features: {}, ttl: {}, fileSizes: {} },
}

// Click a metric selector button by its visible label.
async function selectMetric(w, label) {
    const btn = w.findAll('button').find((b) => b.text() === label)
    expect(btn, `metric button "${label}"`).toBeTruthy()
    await btn.trigger('click')
}

describe('StatsUsagePanel', () => {
    it('renders the current and lifetime periods side by side', () => {
        const w = mountPanel()
        expect(w.text()).toContain('Server Statistics')
        expect(w.text()).toContain('Current Usage')
        expect(w.text()).toMatch(/Lifetime Usage \(since /)
        expect((w.text().match(/\(since /g) || []).length).toBe(1)
    })

    it('renders both period tile sets plus the four window tiles', () => {
        const w = mountPanel()
        for (const label of ['Users', 'Uploads', 'Files', 'Total Size',
            'Lifetime Users', 'Lifetime Uploads', 'Lifetime Files', 'Lifetime Size']) {
            expect(w.text()).toContain(label)
        }
        // 4 current + 4 lifetime + 4 window tiles
        expect(w.findAllComponents(StatTile)).toHaveLength(12)
    })

    it('renders the Activity card with a 4-metric selector', () => {
        const w = mountPanel()
        expect(w.text()).toContain('Activity')
        for (const label of ['Downloads', 'Uploads', 'Downloaded data', 'Uploaded data']) {
            expect(w.findAll('button').some((b) => b.text() === label)).toBe(true)
        }
        // Window tile labels use the renamed "Lifetime" (was "All time").
        for (const label of ['Today', '7 days', '30 days', 'Lifetime']) {
            expect(w.text()).toContain(label)
        }
    })

    it('shows the Downloads windows for the default metric', () => {
        const w = mountPanel()
        // today 3 / 7d 10 / 30d 40 / lifetime 42
        expect(w.text()).toContain('42')
        expect(w.text()).toContain('40')
    })

    it('selector drives the window tiles to the Uploads metric (count)', async () => {
        const w = mountPanel()
        await selectMetric(w, 'Uploads')
        // uploads today 2 / 7d 6 / 30d 20 / lifetime 30
        const text = w.text()
        expect(text).toContain('30') // uploads.total → Lifetime
        expect(text).toContain('20') // uploads.last30Days → 30 days
    })

    it('selector drives the window tiles to a byte metric (humanized)', async () => {
        const w = mountPanel({ dailySeries })
        await selectMetric(w, 'Downloaded data')
        // Lifetime byte tile = usage.downloads.bytes = 1048576 → humanized MB.
        expect(w.text()).toContain('MB')
        // Byte windows are summed from the series (humanReadableSize uses "KB").
        expect(w.text()).toContain('KB') // downloadedBytes summed from the series
    })

    it('selector drives the chart metric together with the tiles', async () => {
        const w = mountPanel({ dailySeries })
        const chart = () => w.find('[data-testid="activity-chart"]')
        expect(chart().attributes('aria-label')).toContain('Downloads per day')

        await selectMetric(w, 'Uploaded data')
        expect(chart().attributes('aria-label')).toContain('Uploaded data per day')
        expect(chart().attributes('data-bytes-mode')).toBe('true')

        await selectMetric(w, 'Uploads')
        expect(chart().attributes('aria-label')).toContain('Uploads per day')
        expect(chart().attributes('data-bytes-mode')).toBe('false')
    })

    it('renders the six distribution cards (feature / TTL / file-size × current / lifetime)', () => {
        const w = mountPanel()
        expect(w.findAllComponents(DistributionCard)).toHaveLength(6)
        for (const title of ['Current Feature Usage', 'Lifetime Feature Usage',
            'Current TTL Distribution', 'Lifetime TTL Distribution']) {
            expect(w.text()).toContain(title)
        }
    })

    it('hides the Activity card when showActivity is false', () => {
        const w = mountPanel({ showActivity: false })
        expect(w.text()).not.toContain('Activity')
        // only the 4 + 4 usage tiles remain
        expect(w.findAllComponents(StatTile)).toHaveLength(8)
    })

    it('renders the scope-specific storage slot', () => {
        const w = mountPanel({}, { storage: '<div class="storage-marker">split</div>' })
        expect(w.find('.storage-marker').exists()).toBe(true)
    })

    it('falls back to "Lifetime Usage" with no since date', () => {
        const noSince = { ...usage, startedAt: '' }
        const w = mountPanel({ usage: noSince })
        expect(w.text()).toContain('Lifetime Usage')
        expect(w.text()).not.toMatch(/\(since /)
    })

    it('omits the chart but keeps the window tiles when dailySeries is null', () => {
        const w = mountPanel()
        expect(w.find('[data-testid="activity-chart"]').exists()).toBe(false)
        // window tiles still render (chart absent → tiles still show)
        expect(w.findAllComponents(StatTile)).toHaveLength(12)
        expect(w.text()).toContain('Lifetime')
    })

    it('renders the daily chart when a dailySeries is provided', () => {
        const w = mountPanel({ dailySeries })
        expect(w.find('[data-testid="activity-chart"]').exists()).toBe(true)
        expect(w.text()).toContain('Activity')
    })

    // ── Current/Lifetime captions ──
    it('renders a one-line caption under both the Current and Lifetime headers', () => {
        const w = mountPanel()
        expect(w.text()).toContain('Active uploads stored right now')
        expect(w.text()).toContain('Everything since stats began, including expired and deleted')
    })

    it('passes the same captions to the distribution cards as a title-attribute tooltip', () => {
        const w = mountPanel()
        const cards = w.findAllComponents(DistributionCard)
        expect(cards).toHaveLength(6)
        const captions = cards.map((c) => c.props('caption'))
        expect(captions.filter((c) => c === 'Active uploads stored right now')).toHaveLength(3)
        expect(captions.filter((c) => c === 'Everything since stats began, including expired and deleted')).toHaveLength(3)
    })

    // ── Byte window tiles show "—" (not a false "0 B") when the daily
    // series is unavailable — count-metric windows are unaffected. ──
    describe('byte window tiles when the daily series is unavailable', () => {
        it('shows "—" for Today/7d/30d while Lifetime still shows a real humanized value', async () => {
            const w = mountPanel() // dailySeries defaults to null (unavailable)
            await selectMetric(w, 'Downloaded data')
            const tiles = w.findAllComponents(StatTile).filter((t) => t.props('bordered'))
            expect(tiles).toHaveLength(4)
            const [today, d7, d30, lifetime] = tiles.map((t) => t.text())
            expect(today).toContain('—')
            expect(d7).toContain('—')
            expect(d30).toContain('—')
            // Lifetime = usage.downloads.bytes = 1048576 -> humanized MB, not a dash.
            expect(lifetime).toContain('MB')
            expect(lifetime).not.toContain('—')
        })

        it('does not affect count-metric windows (they read from the stats object, not the series)', async () => {
            const w = mountPanel() // dailySeries null
            // Default metric is 'downloads' (a count metric)
            const tiles = w.findAllComponents(StatTile).filter((t) => t.props('bordered'))
            for (const tile of tiles) {
                expect(tile.text()).not.toContain('—')
            }
        })

        it('shows real numbers (not dashes) once a dailySeries is provided', async () => {
            const w = mountPanel({ dailySeries })
            await selectMetric(w, 'Downloaded data')
            const tiles = w.findAllComponents(StatTile).filter((t) => t.props('bordered'))
            for (const tile of tiles) {
                expect(tile.text()).not.toContain('—')
            }
        })
    })

    // ── Zero-activity dashboard ──
    describe('zero-activity empty state', () => {
        it('is opt-in only: an all-zero scope still shows the six distribution cards by default (Admin behavior)', () => {
            const w = mountPanel({ usage: zeroUsage }) // emptyStateEnabled defaults false
            expect(w.findAllComponents(DistributionCard)).toHaveLength(6)
            expect(w.text()).not.toContain('Upload your first file')
        })

        it('replaces the six distribution cards with one friendly panel when opted in on an all-zero scope', () => {
            const w = mountPanel({ usage: zeroUsage, emptyStateEnabled: true })
            expect(w.findAllComponents(DistributionCard)).toHaveLength(0)
            expect(w.text()).toContain('Upload your first file to see your usage stats here.')
            expect(w.text()).toContain('Upload a file')
            const link = w.find('a')
            expect(link.exists()).toBe(true)
            expect(link.attributes('href')).toBe('/')
        })

        it('does not show the empty state when opted in but activity is non-zero', () => {
            const w = mountPanel({ emptyStateEnabled: true }) // default `usage` has lifetime.uploads = 5
            expect(w.findAllComponents(DistributionCard)).toHaveLength(6)
            expect(w.text()).not.toContain('Upload your first file')
        })

        it('still renders the tiles and the chart in the empty state (only distribution cards are replaced)', () => {
            const zeroSeries = [
                { day: '2026-05-01', downloads: 0, downloadedBytes: 0, uploads: 0, uploadedBytes: 0 },
            ]
            const w = mountPanel({ usage: zeroUsage, emptyStateEnabled: true, dailySeries: zeroSeries })
            // 4 current + 4 lifetime + 4 window tiles still render
            expect(w.findAllComponents(StatTile)).toHaveLength(12)
            expect(w.find('[data-testid="activity-chart"]').exists()).toBe(true)
        })
    })
})
