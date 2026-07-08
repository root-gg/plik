import { test, expect, ADMIN_LOGIN } from './fixtures.js'

// ── Helpers ──

/**
 * Create an upload via API with the given settings.
 * Adds one small file so the upload is visible in listing.
 * Returns the upload object (including id).
 */
async function apiCreateUpload(page, settings = {}) {
    return page.evaluate(async (settings) => {
        const xsrfMatch = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
        const xsrf = xsrfMatch ? xsrfMatch[1] : ''
        const headers = { 'Content-Type': 'application/json' }
        if (xsrf) headers['X-XSRFToken'] = xsrf

        const r = await fetch('/upload', {
            method: 'POST',
            credentials: 'same-origin',
            headers,
            body: JSON.stringify(settings),
        })
        if (!r.ok) throw new Error(`createUpload failed: ${r.status} ${await r.text()}`)
        return r.json()
    }, settings)
}

/**
 * Delete an upload via API.
 */
async function apiDeleteUpload(page, uploadId) {
    return page.evaluate(async ({ uploadId }) => {
        const xsrfMatch = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
        const xsrf = xsrfMatch ? xsrfMatch[1] : ''
        const headers = {}
        if (xsrf) headers['X-XSRFToken'] = xsrf

        const r = await fetch(`/upload/${uploadId}`, {
            method: 'DELETE',
            credentials: 'same-origin',
            headers,
        })
        if (!r.ok) throw new Error(`deleteUpload failed: ${r.status}`)
    }, { uploadId })
}

/**
 * Navigate to admin uploads tab and wait for it to load.
 */
async function goToAdminUploads(page) {
    await page.goto('/#/admin')
    await page.waitForLoadState('networkidle')
    await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
    await page.waitForLoadState('networkidle')
}

/**
 * Navigate to home uploads tab and wait for it to load.
 */
async function goToHomeUploads(page) {
    await page.goto('/#/home/uploads')
    await page.waitForLoadState('networkidle')
}

// ── Admin badge filter tests ──

test.describe('Admin badge filter controls', () => {
    test('filter bar shows all badge filter buttons', async ({ authenticatedPage: page }) => {
        await goToAdminUploads(page)

        const main = page.locator('main')
        await expect(main.getByText('Filter:')).toBeVisible({ timeout: 5_000 })
        await expect(main.getByRole('button', { name: 'one-shot' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'removable' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'stream' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'extend TTL' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'password' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'encrypted' })).toBeVisible()
    })

    test('one-shot filter shows only matching uploads', async ({ authenticatedPage: page }) => {
        // Ensure we have the right context
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Create two uploads: one with oneShot, one without
        const oneShotUpload = await apiCreateUpload(page, { oneShot: true })
        const normalUpload = await apiCreateUpload(page, {})

        try {
            await goToAdminUploads(page)
            const main = page.locator('main')

            // Both uploads should be visible initially
            await expect(main.getByText(oneShotUpload.id)).toBeVisible({ timeout: 5_000 })
            await expect(main.getByText(normalUpload.id)).toBeVisible()

            // Click one-shot filter
            await main.getByRole('button', { name: 'one-shot' }).click()
            await page.waitForLoadState('networkidle')

            // Only one-shot upload should be visible
            await expect(main.getByText(oneShotUpload.id)).toBeVisible({ timeout: 5_000 })
            await expect(main.getByText(normalUpload.id)).not.toBeVisible()

            // Toggle off — both should reappear
            await main.getByRole('button', { name: 'one-shot' }).click()
            await page.waitForLoadState('networkidle')
            await expect(main.getByText(oneShotUpload.id)).toBeVisible({ timeout: 5_000 })
            await expect(main.getByText(normalUpload.id)).toBeVisible()
        } finally {
            await apiDeleteUpload(page, oneShotUpload.id)
            await apiDeleteUpload(page, normalUpload.id)
        }
    })

    test('removable filter shows only matching uploads', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        const removableUpload = await apiCreateUpload(page, { removable: true })
        const normalUpload = await apiCreateUpload(page, {})

        try {
            await goToAdminUploads(page)
            const main = page.locator('main')

            await expect(main.getByText(removableUpload.id)).toBeVisible({ timeout: 5_000 })
            await expect(main.getByText(normalUpload.id)).toBeVisible()

            await main.getByRole('button', { name: 'removable' }).click()
            await page.waitForLoadState('networkidle')

            await expect(main.getByText(removableUpload.id)).toBeVisible({ timeout: 5_000 })
            await expect(main.getByText(normalUpload.id)).not.toBeVisible()
        } finally {
            await apiDeleteUpload(page, removableUpload.id)
            await apiDeleteUpload(page, normalUpload.id)
        }
    })

    test('multiple filters can be combined', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Upload with both oneShot + removable
        const bothUpload = await apiCreateUpload(page, { oneShot: true, removable: true })
        // Upload with only oneShot
        const onlyOneShotUpload = await apiCreateUpload(page, { oneShot: true })
        // Upload with neither
        const normalUpload = await apiCreateUpload(page, {})

        try {
            await goToAdminUploads(page)
            const main = page.locator('main')

            // Activate both filters
            await main.getByRole('button', { name: 'one-shot' }).click()
            await page.waitForLoadState('networkidle')
            await main.getByRole('button', { name: 'removable' }).click()
            await page.waitForLoadState('networkidle')

            // Only the upload with both settings should be visible
            await expect(main.getByText(bothUpload.id)).toBeVisible({ timeout: 5_000 })
            await expect(main.getByText(onlyOneShotUpload.id)).not.toBeVisible()
            await expect(main.getByText(normalUpload.id)).not.toBeVisible()
        } finally {
            await apiDeleteUpload(page, bothUpload.id)
            await apiDeleteUpload(page, onlyOneShotUpload.id)
            await apiDeleteUpload(page, normalUpload.id)
        }
    })

    test('badge filter state persists in URL', async ({ authenticatedPage: page }) => {
        await goToAdminUploads(page)
        const main = page.locator('main')

        // Click one-shot filter
        await main.getByRole('button', { name: 'one-shot' }).click()
        await page.waitForLoadState('networkidle')

        // URL should contain oneShot=true
        expect(page.url()).toContain('oneShot=true')

        // Reload the page — filter should still be active
        await page.reload({ waitUntil: 'networkidle' })

        // The one-shot button should have the active styling (ring class)
        const oneShotBtn = main.getByRole('button', { name: 'one-shot' })
        await expect(oneShotBtn).toHaveClass(/ring-1/, { timeout: 5_000 })
    })

    test('badge filter state restored on back/forward navigation', async ({ authenticatedPage: page }) => {
        // Navigate directly to admin uploads so we start clean
        await page.goto('/#/admin/uploads')
        await page.waitForLoadState('networkidle')
        const main = page.locator('main')
        const oneShotBtn = main.getByRole('button', { name: 'one-shot' })

        // Activate one-shot filter — pushes a new history entry
        await oneShotBtn.click()
        await page.waitForLoadState('networkidle')
        expect(page.url()).toContain('oneShot=true')
        await expect(oneShotBtn).toHaveClass(/ring-1/)

        // Also activate removable — pushes another history entry
        const removableBtn = main.getByRole('button', { name: 'removable' })
        await removableBtn.click()
        await page.waitForLoadState('networkidle')
        expect(page.url()).toContain('removable=true')
        await expect(removableBtn).toHaveClass(/ring-1/)

        // Go back — removable should deactivate, one-shot stays
        await page.goBack({ waitUntil: 'networkidle' })
        await expect(removableBtn).not.toHaveClass(/ring-1/, { timeout: 5_000 })
        await expect(oneShotBtn).toHaveClass(/ring-1/)

        // Go forward — removable reactivates
        await page.goForward({ waitUntil: 'networkidle' })
        await expect(removableBtn).toHaveClass(/ring-1/, { timeout: 5_000 })
    })
})

// ── Home badge filter tests ──

test.describe('Home badge filter controls', () => {
    test('filter bar shows all badge filter buttons', async ({ authenticatedPage: page }) => {
        await goToHomeUploads(page)

        const main = page.locator('main')
        await expect(main.getByText('Filter:')).toBeVisible({ timeout: 5_000 })
        await expect(main.getByRole('button', { name: 'one-shot' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'removable' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'stream' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'extend TTL' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'password' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'encrypted' })).toBeVisible()
    })

    test('one-shot filter shows only matching uploads', async ({ authenticatedPage: page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const oneShotUpload = await apiCreateUpload(page, { oneShot: true })
        const normalUpload = await apiCreateUpload(page, {})

        try {
            await goToHomeUploads(page)
            const main = page.locator('main')

            // Both uploads should be visible
            await expect(main.getByText(oneShotUpload.id)).toBeVisible({ timeout: 5_000 })
            await expect(main.getByText(normalUpload.id)).toBeVisible()

            // Click one-shot filter
            await main.getByRole('button', { name: 'one-shot' }).click()
            await page.waitForLoadState('networkidle')

            // Only one-shot upload visible
            await expect(main.getByText(oneShotUpload.id)).toBeVisible({ timeout: 5_000 })
            await expect(main.getByText(normalUpload.id)).not.toBeVisible()

            // Toggle off
            await main.getByRole('button', { name: 'one-shot' }).click()
            await page.waitForLoadState('networkidle')
            await expect(main.getByText(normalUpload.id)).toBeVisible({ timeout: 5_000 })
        } finally {
            await apiDeleteUpload(page, oneShotUpload.id)
            await apiDeleteUpload(page, normalUpload.id)
        }
    })

    test('badge filter state persists in URL', async ({ authenticatedPage: page }) => {
        await goToHomeUploads(page)
        const main = page.locator('main')

        // Click removable filter
        await main.getByRole('button', { name: 'removable' }).click()
        await page.waitForLoadState('networkidle')

        // URL should contain removable=true
        expect(page.url()).toContain('removable=true')

        // Reload — filter should still be active
        await page.reload({ waitUntil: 'networkidle' })
        const removableBtn = main.getByRole('button', { name: 'removable' })
        await expect(removableBtn).toHaveClass(/ring-1/, { timeout: 5_000 })
    })

    test('badge filter state restored on back/forward', async ({ authenticatedPage: page }) => {
        // Navigate directly to home uploads to start clean
        await page.goto('/#/home/uploads')
        await page.waitForLoadState('networkidle')
        const main = page.locator('main')

        // Activate removable filter — pushes history entry
        const removableBtn = main.getByRole('button', { name: 'removable' })
        await removableBtn.click()
        await page.waitForLoadState('networkidle')
        expect(page.url()).toContain('removable=true')
        await expect(removableBtn).toHaveClass(/ring-1/)

        // Also activate stream — pushes another entry
        const streamBtn = main.getByRole('button', { name: 'stream' })
        await streamBtn.click()
        await page.waitForLoadState('networkidle')
        expect(page.url()).toContain('stream=true')
        await expect(streamBtn).toHaveClass(/ring-1/)

        // Go back — stream should deactivate, removable stays
        await page.goBack({ waitUntil: 'networkidle' })
        await expect(streamBtn).not.toHaveClass(/ring-1/, { timeout: 5_000 })
        await expect(removableBtn).toHaveClass(/ring-1/)

        // Go forward — stream reactivates
        await page.goForward({ waitUntil: 'networkidle' })
        await expect(streamBtn).toHaveClass(/ring-1/, { timeout: 5_000 })
    })
})

// ── Sort controls tests ──

test.describe('Sort controls', () => {
    test('admin: sort buttons are visible and functional', async ({ authenticatedPage: page }) => {
        await goToAdminUploads(page)
        const main = page.locator('main')

        // Sort and Order buttons should be visible
        await expect(main.getByRole('button', { name: 'Date' })).toBeVisible({ timeout: 5_000 })
        await expect(main.getByRole('button', { name: 'Size' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'Downloads' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'Desc' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'Asc' })).toBeVisible()

        // Date should be active by default
        await expect(main.getByRole('button', { name: 'Date' })).toHaveClass(/text-accent-400/)

        // Click Size — should become active and persist in URL
        await main.getByRole('button', { name: 'Size' }).click()
        await page.waitForLoadState('networkidle')
        await expect(main.getByRole('button', { name: 'Size' })).toHaveClass(/text-accent-400/)
        expect(page.url()).toContain('sort=size')

        // Click Downloads — should become active and persist in URL
        await main.getByRole('button', { name: 'Downloads' }).click()
        await page.waitForLoadState('networkidle')
        await expect(main.getByRole('button', { name: 'Downloads' })).toHaveClass(/text-accent-400/)
        expect(page.url()).toContain('sort=downloads')
    })

    test('home: sort buttons are visible and functional', async ({ authenticatedPage: page }) => {
        await goToHomeUploads(page)
        const main = page.locator('main')
        const badTokenResponses = []
        page.on('response', response => {
            if (response.url().includes('/me/token') && response.status() >= 400) {
                badTokenResponses.push(response.status())
            }
        })

        await expect(main.getByRole('button', { name: 'Date' })).toBeVisible({ timeout: 5_000 })
        await expect(main.getByRole('button', { name: 'Size' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'Downloads' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'Desc' })).toBeVisible()
        await expect(main.getByRole('button', { name: 'Asc' })).toBeVisible()

        // Click Size
        await main.getByRole('button', { name: 'Size' }).click()
        await page.waitForLoadState('networkidle')
        await expect(main.getByRole('button', { name: 'Size' })).toHaveClass(/text-accent-400/)
        expect(page.url()).toContain('sort=size')

        // Click Downloads
        await main.getByRole('button', { name: 'Downloads' }).click()
        await page.waitForLoadState('networkidle')
        await expect(main.getByRole('button', { name: 'Downloads' })).toHaveClass(/text-accent-400/)
        expect(page.url()).toContain('sort=downloads')

        // Reload — sort should persist
        await page.reload({ waitUntil: 'networkidle' })
        await expect(main.getByRole('button', { name: 'Downloads' })).toHaveClass(/text-accent-400/, { timeout: 5_000 })
        expect(badTokenResponses).toEqual([])
    })

})

// ── Direct URL navigation tests (new-tab simulation) ──

test.describe('Direct URL with combined filters', () => {
    test('admin: navigating to URL with sort + badge filters applies all state', async ({ authenticatedPage: page }) => {
        // Navigate directly with multiple params — simulates paste-in-new-tab
        await page.goto('/#/admin/uploads?sort=downloads&oneShot=true&removable=true')
        await page.waitForLoadState('networkidle')
        const main = page.locator('main')

        // Sort should be "Downloads"
        await expect(main.getByRole('button', { name: 'Downloads' })).toHaveClass(/text-accent-400/, { timeout: 5_000 })
        await expect(main.getByRole('button', { name: 'Date' })).not.toHaveClass(/text-accent-400/)

        // Badge filters should be active
        await expect(main.getByRole('button', { name: 'one-shot' })).toHaveClass(/ring-1/)
        await expect(main.getByRole('button', { name: 'removable' })).toHaveClass(/ring-1/)

        // Other filters should NOT be active
        await expect(main.getByRole('button', { name: 'stream' })).not.toHaveClass(/ring-1/)
        await expect(main.getByRole('button', { name: 'password' })).not.toHaveClass(/ring-1/)
    })

    test('home: navigating to URL with sort + badge filters applies all state', async ({ authenticatedPage: page }) => {
        const badTokenResponses = []
        page.on('response', response => {
            if (response.url().includes('/me/token') && response.status() >= 400) {
                badTokenResponses.push(response.status())
            }
        })

        await page.goto('/#/home/uploads?sort=downloads&stream=true&password=true')
        await page.waitForLoadState('networkidle')
        const main = page.locator('main')

        // Sort should be "Downloads"
        await expect(main.getByRole('button', { name: 'Downloads' })).toHaveClass(/text-accent-400/, { timeout: 5_000 })
        await expect(main.getByRole('button', { name: 'Date' })).not.toHaveClass(/text-accent-400/)

        // Badge filters should be active
        await expect(main.getByRole('button', { name: 'stream' })).toHaveClass(/ring-1/)
        await expect(main.getByRole('button', { name: 'password' })).toHaveClass(/ring-1/)

        // Other filters should NOT be active
        await expect(main.getByRole('button', { name: 'one-shot' })).not.toHaveClass(/ring-1/)
        await expect(main.getByRole('button', { name: 'removable' })).not.toHaveClass(/ring-1/)
        expect(badTokenResponses).toEqual([])
    })
})
