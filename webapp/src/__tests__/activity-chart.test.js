import { describe, it, expect, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '../locales/en.json'
import ActivityChart from '../components/ActivityChart.vue'

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

const fullSeries = [
    { day: '2026-05-01', downloads: 5, downloadedBytes: 1024, uploads: 2, uploadedBytes: 4096 },
    { day: '2026-05-02', downloads: 8, downloadedBytes: 2048, uploads: 0, uploadedBytes: 0 },
    { day: '2026-05-03', downloads: 12, downloadedBytes: 1048576, uploads: 7, uploadedBytes: 2097152 },
]

function mountChart(metric = 'downloads', series = fullSeries) {
    return mount(ActivityChart, { global: { plugins: [i18n] }, props: { series, metric } })
}

describe('ActivityChart (controlled by metric prop)', () => {
    afterEach(() => {
        delete window.matchMedia
    })

    it('renders one bar per non-zero day of the selected metric', () => {
        const w = mountChart('downloads')
        expect(w.findAll('.chart-bar')).toHaveLength(3)
        expect(w.findAll('.chart-hit')).toHaveLength(3)
    })

    it('renders bars only for non-zero days of the selected metric (uploads)', () => {
        // uploads: day 2 is zero → 2 bars, still 3 hit targets.
        const w = mountChart('uploads')
        expect(w.findAll('.chart-bar')).toHaveLength(2)
        expect(w.findAll('.chart-hit')).toHaveLength(3)
    })

    it('exposes a per-metric aria-label', () => {
        expect(mountChart('downloads').find('[data-testid="activity-chart"]').attributes('aria-label'))
            .toBe('Downloads per day, last 3 days')
        expect(mountChart('uploads').find('[data-testid="activity-chart"]').attributes('aria-label'))
            .toBe('Uploads per day, last 3 days')
        expect(mountChart('downloadedBytes').find('[data-testid="activity-chart"]').attributes('aria-label'))
            .toBe('Downloaded data per day, last 3 days')
        expect(mountChart('uploadedBytes').find('[data-testid="activity-chart"]').attributes('aria-label'))
            .toBe('Uploaded data per day, last 3 days')
    })

    it('shows the honest empty-state line for an all-zero selected metric', () => {
        const w = mountChart('uploadedBytes', [
            { day: '2026-05-01', downloads: 5, downloadedBytes: 10, uploads: 1, uploadedBytes: 0 },
            { day: '2026-05-02', downloads: 8, downloadedBytes: 20, uploads: 2, uploadedBytes: 0 },
        ])
        expect(w.findAll('.chart-bar')).toHaveLength(0)
        expect(w.text()).toContain('No uploaded data yet')
        expect(w.find('.chart-baseline').exists()).toBe(true)
    })

    it('humanizes count metrics as grouped integers in the tooltip', async () => {
        const w = mountChart('downloads')
        await w.findAll('.chart-hit')[2].trigger('mouseenter')
        expect(w.find('[role="tooltip"]').text()).toContain('12')
    })

    it('humanizes byte metrics as data sizes in the tooltip', async () => {
        const w = mountChart('downloadedBytes')
        expect(w.find('[data-testid="activity-chart"]').attributes('data-bytes-mode')).toBe('true')
        await w.findAll('.chart-hit')[2].trigger('mouseenter')
        expect(w.find('[role="tooltip"]').text()).toContain('MB')
    })

    it('applies the grow-in animation class when motion is allowed', () => {
        const w = mountChart('downloads')
        expect(w.find('.chart-bar').classes()).toContain('chart-bar--animate')
    })

    it('skips the animation class when prefers-reduced-motion is set', () => {
        window.matchMedia = (q) => ({
            matches: q.includes('prefers-reduced-motion'),
            media: q,
            addEventListener() {},
            removeEventListener() {},
            addListener() {},
            removeListener() {},
        })
        const w = mountChart('downloads')
        expect(w.find('.chart-bar').classes()).not.toContain('chart-bar--animate')
    })
})
