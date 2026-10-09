import { test, expect, uploadTestFile } from './fixtures.js'

async function createToken(page, comment) {
    return page.evaluate(async (comment) => {
        const xsrf = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)?.[1] || ''
        const headers = { 'Content-Type': 'application/json' }
        if (xsrf) headers['X-XSRFToken'] = xsrf
        const r = await fetch('/me/token', {
            method: 'POST',
            credentials: 'same-origin',
            headers,
            body: JSON.stringify({ comment }),
        })
        if (!r.ok) throw new Error(`create token failed: ${r.status} ${await r.text()}`)
        return r.json()
    }, comment)
}

async function createTokenUploadWithFile(page, token, filename, content) {
    return page.evaluate(async ({ token, filename, content }) => {
        const xsrf = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)?.[1] || ''
        const headers = { 'Content-Type': 'application/json', 'X-PlikToken': token }
        if (xsrf) headers['X-XSRFToken'] = xsrf

        const uploadResp = await fetch('/upload', {
            method: 'POST',
            credentials: 'same-origin',
            headers,
            body: JSON.stringify({}),
        })
        if (!uploadResp.ok) throw new Error(`create upload failed: ${uploadResp.status} ${await uploadResp.text()}`)
        const upload = await uploadResp.json()

        const form = new FormData()
        form.append('file', new Blob([content], { type: 'text/plain' }), filename)
        const fileHeaders = { 'X-PlikToken': token }
        if (xsrf) fileHeaders['X-XSRFToken'] = xsrf
        const fileResp = await fetch(`/file/${upload.id}`, {
            method: 'POST',
            credentials: 'same-origin',
            headers: fileHeaders,
            body: form,
        })
        if (!fileResp.ok) throw new Error(`add file failed: ${fileResp.status} ${await fileResp.text()}`)
        return { upload, file: await fileResp.json() }
    }, { token, filename, content })
}

async function createAuthenticatedUploadWithFile(page, filename, content, type = 'application/octet-stream') {
    return page.evaluate(async ({ filename, content, type }) => {
        const xsrf = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)?.[1] || ''
        const headers = { 'Content-Type': 'application/json' }
        if (xsrf) headers['X-XSRFToken'] = xsrf

        const uploadResp = await fetch('/upload', {
            method: 'POST',
            credentials: 'same-origin',
            headers,
            body: JSON.stringify({}),
        })
        if (!uploadResp.ok) throw new Error(`create upload failed: ${uploadResp.status} ${await uploadResp.text()}`)
        const upload = await uploadResp.json()

        const form = new FormData()
        form.append('file', new Blob([content], { type }), filename)
        const fileHeaders = {}
        if (xsrf) fileHeaders['X-XSRFToken'] = xsrf
        const fileResp = await fetch(`/file/${upload.id}`, {
            method: 'POST',
            credentials: 'same-origin',
            headers: fileHeaders,
            body: form,
        })
        if (!fileResp.ok) throw new Error(`add file failed: ${fileResp.status} ${await fileResp.text()}`)
        return { upload, file: await fileResp.json() }
    }, { filename, content, type })
}

test.describe('Stats UI', () => {
    test('upload admins see download stats but regular viewers do not', async ({ authenticatedPage: page, browser }) => {
        // Genuinely binary content (NUL bytes) — the server sniffs actual bytes
        // to set isText, regardless of filename/declared content-type. A real
        // .bin file must not be auto-previewed by DownloadView, or the "Never
        // downloaded yet" baseline below wouldn't hold (previews count as downloads).
        const binaryContent = '\x00'.repeat(32)
        const never = await createAuthenticatedUploadWithFile(page, 'stats-download-never.bin', binaryContent)
        await page.goto(`/#/?id=${never.upload.id}`)
        await page.waitForLoadState('networkidle')
        await page.getByTitle('Toggle details').first().click()
        const main = page.locator('main')
        await expect(main.getByText('Downloads:')).toBeVisible({ timeout: 5_000 })
        await expect(main.getByText('Last download:')).toBeVisible()
        await expect(main.getByText('Never', { exact: true })).toBeVisible()

        await uploadTestFile(page, 'stats-download.txt', 'download stats content')
        const fileLink = page.getByRole('link', { name: 'stats-download.txt' }).first()
        const href = await fileLink.getAttribute('href')
        expect(href).toBeTruthy()

        await page.evaluate(async (href) => {
            const r = await fetch(href)
            if (!r.ok) throw new Error(`download failed: ${r.status}`)
            await r.text()
        }, href)

        await page.reload({ waitUntil: 'networkidle' })
        await page.getByTitle('Toggle details').first().click()
        await expect(main.getByText('Downloads:')).toBeVisible({ timeout: 5_000 })
        await expect(main.getByText('Last download:')).toBeVisible()

        const viewerContext = await browser.newContext()
        const viewer = await viewerContext.newPage()
        await viewer.goto(page.url())
        await viewer.waitForLoadState('networkidle')
        await viewer.getByTitle('Toggle details').first().click()
        await expect(viewer.locator('main').getByText('Downloads:')).toHaveCount(0)
        await viewerContext.close()
    })

    test('admin uploads tab sorts by downloaded data and shows the values', async ({ authenticatedPage: page }) => {
        const downloadURL = await uploadTestFile(page, 'stats-uploads-tab-sort.txt', 'downloaded data sort content')
        const uploadId = downloadURL.match(/[?&]id=([^&]+)/)?.[1]
        expect(uploadId).toBeTruthy()

        const fileLink = page.getByRole('link', { name: 'stats-uploads-tab-sort.txt' }).first()
        const href = await fileLink.getAttribute('href')
        expect(href).toBeTruthy()

        // Trigger a real download so this upload's downloaded_bytes is non-zero.
        await page.evaluate(async (href) => {
            const r = await fetch(href)
            if (!r.ok) throw new Error(`download failed: ${r.status}`)
            await r.text()
        }, href)

        await page.goto('/#/admin/uploads')
        await page.waitForLoadState('networkidle')

        const mainContent = page.locator('main')
        const sortButton = mainContent.getByRole('button', { name: 'Downloaded data', exact: true })
        await expect(sortButton).toBeVisible({ timeout: 5_000 })

        await sortButton.click()
        await page.waitForLoadState('networkidle')
        await expect(sortButton).toHaveClass(/text-accent-400/)

        // The e2e suite runs against one shared server (single worker), so other
        // specs' uploads are also listed here — locate this test's own upload by
        // ID instead of assuming sort position, and check it shows a non-zero
        // total size and a non-zero downloaded-data figure.
        const uploadCard = mainContent.locator('.glass-card.p-4').filter({ hasText: uploadId })
        await expect(uploadCard).toBeVisible({ timeout: 5_000 })
        await expect(uploadCard.getByText(/total size:\s*[1-9][\d.]*\s*(B|KB|MB)/)).toBeVisible()
        await expect(uploadCard.getByText(/downloaded data:\s*[1-9][\d.]*\s*(B|KB|MB)/)).toBeVisible()
    })

    test('sidebar download count reflects the auto-preview immediately after an in-app upload (no stale count)', async ({ authenticatedPage: page }) => {
        // Owner self-downloads and previews count as downloads (they are real
        // served traffic). Uploading a single text file through the UI auto-opens
        // its preview, which fetches the file content and is itself counted as a
        // download.
        // Regression check for the stale-count fix: without any reload, the
        // sidebar must reflect that download, not the pre-preview count.
        await uploadTestFile(page, 'stats-stale-count.txt', 'stale count regression content')

        const sidebar = page.locator('aside').first()
        await expect(sidebar.getByText('Downloads', { exact: false })).toBeVisible({ timeout: 5_000 })
        await expect(sidebar.locator('p.tabular-nums')).toHaveText('1', { timeout: 5_000 })

        // Same regression for the image preview path: an <img> preview loads
        // straight from the server URL (the browser's own GET is the counted
        // download), so the refresh hooks the element's load event rather than
        // a fetch promise. 1x1 transparent PNG, uploaded through the real UI.
        const pngBuffer = Buffer.from(
            'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
            'base64',
        )
        await page.goto('/')
        await page.waitForLoadState('networkidle')
        await page.locator('input[type="file"]').setInputFiles({
            name: 'stats-stale-count.png',
            mimeType: 'image/png',
            buffer: pngBuffer,
        })
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Auto-preview renders the image; its load event triggers the refresh.
        await expect(page.locator('#file-viewer-panel img')).toBeVisible({ timeout: 5_000 })
        const imageSidebar = page.locator('aside').first()
        await expect(imageSidebar.locator('p.tabular-nums')).toHaveText('1', { timeout: 5_000 })
    })

    test('admin stats shows lifetime stats and trending downloads', async ({ authenticatedPage: page }) => {
        const downloadURL = await uploadTestFile(page, 'stats-trending-link.txt', 'trending download content')
        const uploadId = downloadURL.match(/[?&]id=([^&]+)/)?.[1]
        expect(uploadId).toBeTruthy()
        const fileLink = page.getByRole('link', { name: 'stats-trending-link.txt' }).first()
        const href = await fileLink.getAttribute('href')
        expect(href).toBeTruthy()

        await page.evaluate(async (href) => {
            const r = await fetch(href)
            if (!r.ok) throw new Error(`download failed: ${r.status}`)
            await r.text()
        }, href)

        await page.goto('/#/admin/stats')
        await page.waitForLoadState('networkidle')

        await expect(page.getByText(/^Lifetime Usage \(since /)).toBeVisible({ timeout: 5_000 })
        await expect(page.getByText('Current Usage')).toBeVisible()
        await expect(page.getByText('Lifetime Usage')).toBeVisible()
        await expect(page.getByText('Current Feature Usage')).toBeVisible()
        await expect(page.getByText('Current TTL Distribution')).toBeVisible()
        await expect(page.getByText('Lifetime TTL Distribution')).toBeVisible()
        const activityPanel = page.locator('.glass-card').filter({ hasText: 'Activity' }).filter({ hasText: 'Lifetime' })
        await expect(activityPanel.getByText('Today', { exact: true })).toBeVisible()
        await expect(activityPanel.getByText('7 days', { exact: true })).toBeVisible()
        await expect(activityPanel.getByText('30 days', { exact: true })).toBeVisible()
        await expect(activityPanel.getByText('Lifetime', { exact: true })).toBeVisible()
        // The 4-metric selector drives both the tiles and the chart.
        await expect(activityPanel.getByRole('button', { name: 'Downloads', exact: true })).toBeVisible()
        await expect(activityPanel.getByRole('button', { name: 'Uploads', exact: true })).toBeVisible()
        await expect(activityPanel.getByRole('button', { name: 'Downloaded data', exact: true })).toBeVisible()
        await expect(activityPanel.getByRole('button', { name: 'Uploaded data', exact: true })).toBeVisible()
        await expect(page.getByText('Trending Uploads')).toBeVisible()
        await expect(page.getByText('Trending Files')).toBeVisible()

        // Trending window chips use the same Today/7 days/30 days/Lifetime
        // vocabulary as the Activity tiles (not the raw 1d/7d/30d/all API values,
        // and not the retired "All time" wording).
        const trendingCard = page.locator('.glass-card').filter({ hasText: 'Trending Uploads' })
        await expect(trendingCard.getByRole('button', { name: 'Today', exact: true })).toBeVisible()
        await expect(trendingCard.getByRole('button', { name: '7 days', exact: true })).toBeVisible()
        await expect(trendingCard.getByRole('button', { name: '30 days', exact: true })).toBeVisible()
        await expect(trendingCard.getByRole('button', { name: 'Lifetime', exact: true })).toBeVisible()

        await expect(page.getByRole('button', { name: uploadId }).first()).toBeVisible()
        // exact: true — this upload has no comment, so its headline reads
        // "Upload by local:admin" (B4), which would otherwise substring-match
        // the SAME "local:admin" query; the owner subtitle button's accessible
        // name is exactly "local:admin".
        await expect(page.getByRole('button', { name: 'local:admin', exact: true }).first()).toBeVisible()

        await page.getByRole('button', { name: 'local:admin', exact: true }).first().click()
        await expect(page).toHaveURL(/#\/admin\/users/)

        await page.goto('/#/admin/stats')
        await page.waitForLoadState('networkidle')
        await page.getByRole('button', { name: uploadId }).first().click()
        await expect(page).toHaveURL(new RegExp(`[?&]id=${uploadId}`))
    })

    test('admin trending uploads card toggles between downloads and downloaded data', async ({ authenticatedPage: page }) => {
        const downloadURL = await uploadTestFile(page, 'stats-trending-toggle.txt', 'trending toggle content')
        const uploadId = downloadURL.match(/[?&]id=([^&]+)/)?.[1]
        expect(uploadId).toBeTruthy()
        const fileLink = page.getByRole('link', { name: 'stats-trending-toggle.txt' }).first()
        const href = await fileLink.getAttribute('href')
        expect(href).toBeTruthy()

        await page.evaluate(async (href) => {
            const r = await fetch(href)
            if (!r.ok) throw new Error(`download failed: ${r.status}`)
            await r.text()
        }, href)

        await page.goto('/#/admin/stats')
        await page.waitForLoadState('networkidle')

        const trendingCard = page.locator('.glass-card').filter({ hasText: 'Trending Uploads' })
        const downloadsToggle = trendingCard.getByRole('button', { name: 'Downloads', exact: true })
        const bytesToggle = trendingCard.getByRole('button', { name: 'Downloaded data', exact: true })
        await expect(downloadsToggle).toBeVisible({ timeout: 5_000 })
        await expect(bytesToggle).toBeVisible()

        // Default sort (downloads): every row shows BOTH a downloads figure and a
        // humanized bytes figure (the toggle emphasizes, never hides, the
        // other metric).
        await expect(trendingCard.getByText(/\d+ downloads/).first()).toBeVisible()
        await expect(trendingCard.getByText(/[\d.]+\s*(B|KB|MB|GB)/).first()).toBeVisible()
        await expect(downloadsToggle).toHaveClass(/text-accent-300/)

        await bytesToggle.click()
        await page.waitForLoadState('networkidle')

        // Toggling emphasis: Downloaded data is now the active/emphasized metric.
        await expect(bytesToggle).toHaveClass(/text-accent-300/)
        await expect(trendingCard.getByText(/\d+ downloads/).first()).toBeVisible()
        await expect(trendingCard.getByText(/[\d.]+\s*(B|KB|MB|GB)/).first()).toBeVisible()
    })

    test('home stats and token stats expose current and lifetime usage', async ({ authenticatedPage: page }) => {
        const small = await createToken(page, 'stats-small-token')
        const large = await createToken(page, 'stats-large-token')
        await createTokenUploadWithFile(page, small.token, 'small-token.txt', 'small')
        await createTokenUploadWithFile(page, large.token, 'large-token.txt', 'large token content that is bigger')

        await page.goto('/#/home/stats')
        await page.waitForLoadState('networkidle')
        await expect(page.getByText(/^Lifetime Usage \(since /)).toBeVisible({ timeout: 5_000 })
        await expect(page.getByText('Current Usage')).toBeVisible()
        await expect(page.getByText('Lifetime Usage')).toBeVisible()

        // Home renders through the same shared StatsUsagePanel as Admin, scoped
        // to this user — distribution cards and the downloads windows row included.
        await expect(page.getByText('Current Feature Usage')).toBeVisible()
        await expect(page.getByText('Lifetime Feature Usage')).toBeVisible()
        await expect(page.getByText('Current TTL Distribution')).toBeVisible()
        await expect(page.getByText('Lifetime TTL Distribution')).toBeVisible()
        const activityPanel = page.locator('.glass-card').filter({ hasText: 'Activity' }).filter({ hasText: 'Lifetime' })
        await expect(activityPanel.getByText('Today', { exact: true })).toBeVisible()
        await expect(activityPanel.getByText('7 days', { exact: true })).toBeVisible()
        await expect(activityPanel.getByText('30 days', { exact: true })).toBeVisible()
        await expect(activityPanel.getByText('Lifetime', { exact: true })).toBeVisible()

        // Privacy: no admin/server-scope-only data leaks onto the user's Home page.
        // Home carries a SELF-scoped Trending Uploads panel (this user's own
        // uploads only, dedicated test below) — so a blanket "no Trending at all"
        // check would not hold; what must still never appear is the admin-only,
        // cross-user Trending Files card (deliberately no self-scoped files endpoint) and
        // the owner/anonymous chips that only make sense server-wide.
        await expect(page.getByText('Server Statistics')).not.toBeVisible()
        await expect(page.getByText('Trending Files')).not.toBeVisible()
        await expect(page.getByText('storage split')).not.toBeVisible()
        await expect(page.getByText('Anonymous', { exact: true })).not.toBeVisible()

        await page.goto('/#/home/tokens')
        await page.waitForLoadState('networkidle')
        await page.getByRole('button', { name: 'Current Size' }).click()
        await page.waitForLoadState('networkidle')

        const tokenCards = page.locator('main .glass-card').filter({ hasText: /stats-(small|large)-token/ })
        await expect(tokenCards.first()).toContainText('stats-large-token', { timeout: 5_000 })
        await expect(page).not.toHaveURL(new RegExp(large.token))

        await tokenCards.first().getByTitle('Toggle token details').click()
        await expect(tokenCards.first().getByText('Current uploads')).toBeVisible()
        await expect(tokenCards.first().getByText('Lifetime uploads')).toBeVisible()
        await expect(tokenCards.first().getByText('Last upload')).toBeVisible()
    })

    test('home stats shows the self-scoped trending panel with the user\'s own uploads', async ({ authenticatedPage: page }) => {
        const downloadURL = await uploadTestFile(page, 'stats-home-trending.txt', 'home trending content')
        const uploadId = downloadURL.match(/[?&]id=([^&]+)/)?.[1]
        expect(uploadId).toBeTruthy()
        const fileLink = page.getByRole('link', { name: 'stats-home-trending.txt' }).first()
        const href = await fileLink.getAttribute('href')
        expect(href).toBeTruthy()

        await page.evaluate(async (href) => {
            const r = await fetch(href)
            if (!r.ok) throw new Error(`download failed: ${r.status}`)
            await r.text()
        }, href)

        await page.goto('/#/home/stats')
        await page.waitForLoadState('networkidle')

        // Self-scoped title (distinct from Admin's cross-user "Trending Uploads"),
        // no Files card, and this user's own just-downloaded upload shows up.
        await expect(page.getByText('Your Trending Uploads')).toBeVisible({ timeout: 5_000 })
        await expect(page.getByText('Trending Files')).not.toBeVisible()
        await expect(page.getByRole('button', { name: uploadId }).first()).toBeVisible()

        // The metric toggle exists here too and drives the same emphasis/order
        // behavior as Admin's.
        const trendingCard = page.locator('.glass-card').filter({ hasText: 'Your Trending Uploads' })
        const bytesToggle = trendingCard.getByRole('button', { name: 'Downloaded data', exact: true })
        await expect(bytesToggle).toBeVisible()
        await bytesToggle.click()
        await page.waitForLoadState('networkidle')
        await expect(bytesToggle).toHaveClass(/text-accent-300/)

        // Clicking the upload navigates to its detail/download view, exactly like Admin's.
        await page.getByRole('button', { name: uploadId }).first().click()
        await expect(page).toHaveURL(new RegExp(`[?&]id=${uploadId}`))
    })

    test('daily activity chart renders bars, hover tooltip, and metric switch on admin + home', async ({ authenticatedPage: page }) => {
        // Seed a real download so today's rollup (downloads + bytes) is non-zero.
        await uploadTestFile(page, 'chart-e2e.txt', 'chart e2e download content')
        const fileLink = page.getByRole('link', { name: 'chart-e2e.txt' }).first()
        const href = await fileLink.getAttribute('href')
        expect(href).toBeTruthy()
        await page.evaluate(async (href) => {
            const r = await fetch(href)
            if (!r.ok) throw new Error(`download failed: ${r.status}`)
            await r.text()
        }, href)

        // Admin dashboard: chart is present with bars.
        await page.goto('/#/admin/stats')
        await page.waitForLoadState('networkidle')
        const chart = page.locator('[data-testid="activity-chart"]')
        await expect(chart).toBeVisible({ timeout: 5_000 })
        expect(await chart.locator('.chart-bar').count()).toBeGreaterThan(0)

        // Hovering the most-recent column shows a tooltip with a humanized value.
        const hits = chart.locator('.chart-hit')
        const count = await hits.count()
        await hits.nth(count - 1).hover()
        const tooltip = page.locator('[role="tooltip"]')
        await expect(tooltip).toBeVisible()
        await expect(tooltip).toContainText(/\d/)

        // Switching to a byte metric swaps the tooltip to a byte value. Scoped to
        // the Activity card specifically (there is a second, unrelated
        // "Downloaded data" toggle on this page — the Trending Uploads metric
        // toggle — so an unscoped lookup is now ambiguous).
        const activityPanel = page.locator('.glass-card').filter({ hasText: 'Activity' }).filter({ hasText: 'Lifetime' })
        await activityPanel.getByRole('button', { name: 'Downloaded data', exact: true }).click()
        await hits.nth(count - 1).hover()
        await expect(tooltip).toBeVisible()
        await expect(tooltip).toContainText('B')

        // Home dashboard renders the same chart for the user's own series.
        await page.goto('/#/home/stats')
        await page.waitForLoadState('networkidle')
        const homeChart = page.locator('[data-testid="activity-chart"]')
        await expect(homeChart).toBeVisible({ timeout: 5_000 })
        expect(await homeChart.locator('.chart-bar').count()).toBeGreaterThan(0)
    })
})
