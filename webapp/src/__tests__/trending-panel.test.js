import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import en from '../locales/en.json'
import TrendingPanel from '../components/TrendingPanel.vue'

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', fallbackLocale: 'en', messages: { en } })

const uploads = [
    {
        id: 'upload-1', comments: 'quarterly report', user: 'local:alice',
        files: 2, downloadCount: 129, downloadedBytes: 6_820_000_000, lastDownloadedAt: '2026-05-04T10:00:00Z',
    },
]

const files = [
    {
        id: 'file-1', uploadID: 'upload-1', name: 'report.pdf', user: 'local:alice',
        size: 1024, downloadCount: 42, lastDownloadedAt: '2026-05-04T10:00:00Z',
    },
]

function mountPanel(props = {}) {
    return mount(TrendingPanel, {
        global: { plugins: [i18n] },
        props: { window: '7d', uploads, files, ...props },
    })
}

describe('TrendingPanel', () => {
    it('renders admin-mode content matching AdminView\'s original markup (window chips, both cards, human-first headline)', () => {
        const w = mountPanel()
        const text = w.text()
        expect(text).toContain('Trending')
        expect(text).toContain('Trending Uploads')
        expect(text).toContain('Trending Files')
        expect(text).toContain('quarterly report')
        expect(text).toContain('report.pdf')

        const chipLabels = w.findAll('button').map((b) => b.text())
        expect(chipLabels).toContain('Today')
        expect(chipLabels).toContain('7 days')
        expect(chipLabels).toContain('30 days')
        expect(chipLabels).toContain('Lifetime')
    })

    it('shows the metric toggle with Downloads / Downloaded data options, reusing the Activity panel\'s labels', () => {
        const w = mountPanel()
        const toggleLabels = w.findAll('button').map((b) => b.text())
        expect(toggleLabels).toContain('Downloads')
        expect(toggleLabels).toContain('Downloaded data')
    })

    it('emphasizes downloads first by default (sort=downloads) and shows both values humanized', () => {
        const w = mountPanel({ sort: 'downloads' })
        const text = w.text()
        expect(text).toContain('129 downloads')
        expect(text).toContain('6.82 GB')
        // Order: the emphasized (sorted) metric text must appear before the secondary one.
        expect(text.indexOf('129 downloads')).toBeLessThan(text.indexOf('6.82 GB'))
    })

    it('reorders and emphasizes downloaded data first when sort=downloadedBytes', () => {
        const w = mountPanel({ sort: 'downloadedBytes' })
        const text = w.text()
        expect(text.indexOf('6.82 GB')).toBeLessThan(text.indexOf('129 downloads'))
    })

    it('clicking the metric toggle emits update:sort with the new value', async () => {
        const w = mountPanel({ sort: 'downloads' })
        const bytesButton = w.findAll('button').find((b) => b.text() === 'Downloaded data')
        expect(bytesButton).toBeTruthy()
        await bytesButton.trigger('click')
        expect(w.emitted('update:sort')).toEqual([['downloadedBytes']])
    })

    it('clicking a window chip emits update:window with the new value', async () => {
        const w = mountPanel({ window: '7d' })
        const todayButton = w.findAll('button').find((b) => b.text() === 'Today')
        await todayButton.trigger('click')
        expect(w.emitted('update:window')).toEqual([['1d']])
    })

    it('clicking the upload row/ID emits open-upload with the upload id', async () => {
        const w = mountPanel()
        const idButton = w.findAll('button').find((b) => b.text() === 'upload-1')
        await idButton.trigger('click')
        expect(w.emitted('open-upload')).toEqual([['upload-1']])
    })

    it('clicking a file row emits open-upload with both the upload id and file id', async () => {
        const w = mountPanel()
        const fileButton = w.findAll('button').find((b) => b.text() === 'report.pdf')
        await fileButton.trigger('click')
        expect(w.emitted('open-upload')).toEqual([['upload-1', 'file-1']])
    })

    it('admin mode shows the owner chip and emits view-user on click', async () => {
        const w = mountPanel({ mode: 'admin' })
        const ownerButton = w.findAll('button').find((b) => b.text() === 'local:alice')
        expect(ownerButton).toBeTruthy()
        await ownerButton.trigger('click')
        expect(w.emitted('view-user')).toEqual([['local:alice']])
    })

    it('self mode hides the owner chip/anonymous label and never emits view-user', () => {
        // Self-scoped trending never has a Trending Files card (file-grain
        // byte trending is out of scope), so showFiles=false here matches how
        // HomeView actually uses mode="self".
        const w = mountPanel({ mode: 'self', showFiles: false, files: [] })
        expect(w.findAll('button').some((b) => b.text() === 'local:alice')).toBe(false)
        expect(w.text()).not.toContain('Anonymous')
    })

    it('self mode with showFiles=false hides the Trending Files card entirely and uses the self-scoped title', () => {
        const w = mountPanel({ mode: 'self', showFiles: false, files: [] })
        expect(w.text()).not.toContain('Trending Files')
        expect(w.text()).toContain('Your Trending Uploads')
    })

    it('admin mode keeps the "Trending Uploads" title (not the self-scoped one)', () => {
        const w = mountPanel({ mode: 'admin' })
        expect(w.text()).toContain('Trending Uploads')
        expect(w.text()).not.toContain('Your Trending Uploads')
    })

    it('shows the loading state and hides both cards while loading', () => {
        const w = mountPanel({ loading: true })
        expect(w.text()).toContain('Loading...')
        expect(w.text()).not.toContain('quarterly report')
    })

    it('shows the empty state when there are no trending uploads', () => {
        const w = mountPanel({ uploads: [] })
        expect(w.text()).toContain('No trending uploads yet')
    })

    it('falls back to "Upload by <owner>" / "Anonymous upload" when there is no comment', () => {
        const w = mountPanel({
            uploads: [{ id: 'u2', comments: '', user: 'local:bob', files: 0, downloadCount: 1, downloadedBytes: 0 }],
        })
        expect(w.text()).toContain('Upload by local:bob')

        const anon = mountPanel({
            uploads: [{ id: 'u3', comments: '', user: '', files: 0, downloadCount: 1, downloadedBytes: 0 }],
        })
        expect(anon.text()).toContain('Anonymous upload')
    })
})
