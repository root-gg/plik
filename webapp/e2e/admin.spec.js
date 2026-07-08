import { test, expect, ADMIN_LOGIN } from './fixtures.js'

test.describe('Admin view', () => {
    test('stats tab shows server info', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Stats view is shown by default — should show "Server Configuration" or "Server Statistics"
        await expect(page.getByText('Server Configuration')).toBeVisible({ timeout: 5_000 })
        await expect(page.getByText('Server Statistics')).toBeVisible()
    })

    test('uploads tab shows upload list', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Click Uploads sidebar nav button
        await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
        await page.waitForLoadState('networkidle')

        // Should show upload list (may be empty with "No uploads" or have entries)
        const content = page.getByText('No uploads').or(page.locator('.glass-card'))
        await expect(content.first()).toBeVisible({ timeout: 5_000 })
    })

    test('users tab shows admin user', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Click Users sidebar nav button
        await page.getByRole('button', { name: 'Users', exact: true }).click()
        await page.waitForLoadState('networkidle')

        // Should show the admin user login in the main content area
        // The admin user name appears twice in the same card (login + admin badge),
        // so use .first() to avoid strict mode violation
        const mainContent = page.locator('main')
        await expect(mainContent.getByText(ADMIN_LOGIN).first()).toBeVisible({ timeout: 5_000 })
    })

    test('create and delete user', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Click "Create User" sidebar button to open modal
        await page.getByRole('button', { name: 'Create User', exact: true }).click()

        // Wait for modal to appear
        await expect(page.getByText('Create User').nth(1)).toBeVisible({ timeout: 3_000 })

        // Fill the modal form
        await page.locator('input[placeholder="min 4 chars"]').fill('testuser')
        await page.locator('input[placeholder="min 8 chars"]').fill('testpass123')

        // Click the Create button in the modal
        await page.getByRole('button', { name: 'Create', exact: true }).click()
        await page.waitForLoadState('networkidle')

        // Switch to Users tab to see the new user
        await page.getByRole('button', { name: 'Users', exact: true }).click()
        await page.waitForLoadState('networkidle')

        // Wait for the new user to appear in the main content area
        const mainContent = page.locator('main')
        await expect(mainContent.getByText('testuser').first()).toBeVisible({ timeout: 5_000 })

        // Delete the user — find the Delete button in the testuser's row
        const userCards = mainContent.locator('.glass-card').filter({ hasText: 'testuser' })
        await userCards.first().getByRole('button', { name: 'Delete' }).click()

        // Confirm deletion in the dialog (use exact match to avoid matching disabled Delete buttons)
        await page.getByRole('button', { name: 'Confirm', exact: true }).click()
        await page.waitForLoadState('networkidle')

        // User should no longer appear (give some time for UI to update)
        await expect(mainContent.getByText('testuser')).not.toBeVisible({ timeout: 5_000 })
    })
})

test.describe('Admin server info card', () => {
    test('shows version', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Version should be displayed in sidebar (e.g. "v1.4.0-rc4")
        const versionText = page.locator('aside .glass-card .font-mono')
        await expect(versionText).toBeVisible({ timeout: 5_000 })
        await expect(versionText).toContainText('v')
    })

    test('shows Go version', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Go version text below the version (e.g. "go1.24.0")
        const sidebar = page.locator('aside .glass-card').first()
        await expect(sidebar).toContainText('go1.', { timeout: 5_000 })
    })

    test('release and mint badges green when true', async ({ authenticatedPage: page, withVersion }) => {
        await withVersion({ isRelease: true, isMint: true })
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        const sidebar = page.locator('aside .glass-card').first()
        const releaseBadge = sidebar.locator('.rounded-full').filter({ hasText: 'release' })
        const mintBadge = sidebar.locator('.rounded-full').filter({ hasText: 'mint' })
        await expect(releaseBadge).toBeVisible({ timeout: 5_000 })
        await expect(releaseBadge).toHaveClass(/bg-emerald/)
        await expect(mintBadge).toBeVisible()
        await expect(mintBadge).toHaveClass(/bg-emerald/)
    })

    test('release and mint badges red when false', async ({ authenticatedPage: page, withVersion }) => {
        await withVersion({ isRelease: false, isMint: false })
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        const sidebar = page.locator('aside .glass-card').first()
        const releaseBadge = sidebar.locator('.rounded-full').filter({ hasText: 'release' })
        const mintBadge = sidebar.locator('.rounded-full').filter({ hasText: 'mint' })
        await expect(releaseBadge).toBeVisible({ timeout: 5_000 })
        await expect(releaseBadge).toHaveClass(/bg-red/)
        await expect(mintBadge).toBeVisible()
        await expect(mintBadge).toHaveClass(/bg-red/)
    })
})

test.describe('Admin server configuration', () => {
    test('shows Max File Size', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        const configPanel = page.locator('.glass-card').filter({ hasText: 'Server Configuration' })
        await expect(configPanel).toBeVisible({ timeout: 5_000 })
        await expect(configPanel.getByText('Max File Size')).toBeVisible()
    })

    test('shows Max TTL', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        const configPanel = page.locator('.glass-card').filter({ hasText: 'Server Configuration' })
        await expect(configPanel).toBeVisible({ timeout: 5_000 })
        await expect(configPanel.getByText('Max TTL')).toBeVisible()
    })

    test('all config values are formatted', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        const configPanel = page.locator('.glass-card').filter({ hasText: 'Server Configuration' })
        await expect(configPanel).toBeVisible({ timeout: 5_000 })

        // All 4 labels should be visible with formatted values
        for (const label of ['Max File Size', 'Max User Size', 'Default TTL', 'Max TTL']) {
            await expect(configPanel.getByText(label)).toBeVisible()
        }

        // The config values should have formatted content (not empty)
        const values = configPanel.locator('.text-surface-200.font-medium')
        const count = await values.count()
        expect(count).toBe(4)

        for (let i = 0; i < count; i++) {
            const text = await values.nth(i).textContent()
            expect(text).toBeTruthy()
        }
    })
})

test.describe('Admin server statistics', () => {
    test('shows all stat labels', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        const statsPanel = page.locator('.glass-card').filter({ hasText: 'Server Statistics' })
        await expect(statsPanel).toBeVisible({ timeout: 5_000 })

        for (const label of ['Current Usage', 'Users', 'Uploads', 'Files', 'Total Size', 'Lifetime Users', 'Lifetime Uploads', 'Lifetime Files', 'Lifetime Size']) {
            await expect(statsPanel.getByText(label, { exact: true })).toBeVisible()
        }
        await expect(statsPanel.getByText(/^Lifetime Usage \(since /)).toBeVisible()

        const activityPanel = page.locator('.glass-card').filter({ hasText: 'Activity' }).filter({ hasText: 'Lifetime' })
        await expect(activityPanel).toBeVisible()
        for (const label of ['Today', '7 days', '30 days', 'Lifetime']) {
            await expect(activityPanel.getByText(label, { exact: true })).toBeVisible()
        }
    })

    test('stat values are not empty or NaN', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        const statsPanel = page.locator('.glass-card').filter({ hasText: 'Server Statistics' })
        await expect(statsPanel).toBeVisible({ timeout: 5_000 })

        // Check that the bold stat values exist and aren't NaN.
        const values = statsPanel.locator('.text-2xl.font-bold')
        const count = await values.count()
        expect(count).toBeGreaterThanOrEqual(8)

        for (let i = 0; i < count; i++) {
            const text = await values.nth(i).textContent()
            expect(text).toBeTruthy()
            expect(text).not.toBe('NaN')
        }

        const activityPanel = page.locator('.glass-card').filter({ hasText: 'Activity' }).filter({ hasText: 'Lifetime' })
        const windowValues = activityPanel.locator('.text-2xl.font-bold')
        await expect(windowValues).toHaveCount(4)
        for (let i = 0; i < 4; i++) {
            const text = await windowValues.nth(i).textContent()
            expect(text).toBeTruthy()
            expect(text).not.toBe('NaN')
        }
    })
})

// ── Helpers ──

/**
 * Create a local user via the admin API.
 * Returns the created user object from the API response.
 */
async function apiCreateUser(page, login, password, isAdmin = false) {
    // Navigate to the admin page first to ensure proper context
    await page.goto('/#/admin')
    await page.waitForLoadState('networkidle')

    return page.evaluate(async ({ login, password, isAdmin }) => {
        const xsrfMatch = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
        const xsrf = xsrfMatch ? xsrfMatch[1] : ''
        const headers = { 'Content-Type': 'application/json' }
        if (xsrf) headers['X-XSRFToken'] = xsrf

        const r = await fetch('/user', {
            method: 'POST',
            credentials: 'same-origin',
            headers,
            body: JSON.stringify({ login, password, provider: 'local', admin: isAdmin }),
        })
        if (!r.ok) throw new Error(`createUser failed: ${r.status} ${await r.text()}`)
        return r.json()
    }, { login, password, isAdmin })
}

/**
 * Delete a local user via the admin API.
 */
async function apiDeleteUser(page, userId) {
    return page.evaluate(async ({ userId }) => {
        const xsrfMatch = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
        const xsrf = xsrfMatch ? xsrfMatch[1] : ''
        const headers = {}
        if (xsrf) headers['X-XSRFToken'] = xsrf

        const r = await fetch(`/user/${encodeURIComponent(userId)}`, {
            method: 'DELETE',
            credentials: 'same-origin',
            headers,
        })
        if (!r.ok) throw new Error(`deleteUser failed: ${r.status}`)
    }, { userId })
}

/**
 * Navigate to users tab in admin view.
 */
async function goToUsersTab(page) {
    await page.goto('/#/admin')
    await page.waitForLoadState('networkidle')
    await page.getByRole('button', { name: 'Users', exact: true }).click()
    await page.waitForLoadState('networkidle')
}


// ── Admin filter/sort controls ──

test.describe('Admin users filter controls', () => {
    test('filter bar is visible with sort, provider, and role controls', async ({ authenticatedPage: page }) => {
        await goToUsersTab(page)

        const mainContent = page.locator('main')

        // Sort controls
        await expect(mainContent.getByText('Sort:')).toBeVisible({ timeout: 5_000 })
        await expect(mainContent.getByRole('button', { name: 'Date' })).toBeVisible()

        // Order controls
        await expect(mainContent.getByText('Order:')).toBeVisible()
        await expect(mainContent.getByRole('button', { name: 'Desc' })).toBeVisible()
        await expect(mainContent.getByRole('button', { name: 'Asc' })).toBeVisible()

        // Provider filter
        await expect(mainContent.getByText('Provider:')).toBeVisible()
        await expect(mainContent.getByRole('button', { name: 'All' }).first()).toBeVisible()
        await expect(mainContent.getByRole('button', { name: 'local' })).toBeVisible()

        // Role filter
        await expect(mainContent.getByText('Role:')).toBeVisible()
        await expect(mainContent.getByRole('button', { name: 'Admin', exact: true })).toBeVisible()
        await expect(mainContent.getByRole('button', { name: 'Non-Admin', exact: true })).toBeVisible()
    })

    test('creation date is displayed on user cards', async ({ authenticatedPage: page }) => {
        await goToUsersTab(page)

        // The admin user card should show a date (e.g. "2/23/2026" or locale equivalent)
        const mainContent = page.locator('main')
        const userCard = mainContent.locator('.glass-card.p-4').filter({ hasText: ADMIN_LOGIN }).first()
        await expect(userCard).toBeVisible({ timeout: 5_000 })

        // CreatedAt date — look for any paragraph with a year number (2020-2030)
        const dateText = userCard.locator('p').filter({ hasText: /20[2-3]\d/ })
        await expect(dateText.first()).toBeVisible()
        const text = await dateText.first().textContent()
        expect(text.trim()).toMatch(/20[2-3]\d/)
    })

    test('admin filter shows only admin users', async ({ authenticatedPage: page }) => {
        // Create a non-admin user for testing
        const user = await apiCreateUser(page, 'filtertest', 'filtertest1234', false)

        try {
            await goToUsersTab(page)
            const mainContent = page.locator('main')

            // Both users should be visible initially
            await expect(mainContent.getByText(ADMIN_LOGIN).first()).toBeVisible({ timeout: 5_000 })
            await expect(mainContent.getByText('filtertest').first()).toBeVisible()

            // Click "Admin" filter
            await mainContent.getByRole('button', { name: 'Admin', exact: true }).click()
            await page.waitForLoadState('networkidle')

            // Only admin user should be visible
            await expect(mainContent.locator('.glass-card.p-4').filter({ hasText: ADMIN_LOGIN }).first()).toBeVisible({ timeout: 5_000 })
            await expect(mainContent.locator('.glass-card.p-4').filter({ hasText: 'filtertest' })).toHaveCount(0)

            // Click "Non-Admin" filter
            await mainContent.getByRole('button', { name: 'Non-Admin' }).click()
            await page.waitForLoadState('networkidle')

            // Only non-admin user should be visible
            await expect(mainContent.locator('.glass-card.p-4').filter({ hasText: 'filtertest' }).first()).toBeVisible({ timeout: 5_000 })
            await expect(mainContent.locator('.glass-card.p-4').filter({ hasText: ADMIN_LOGIN })).toHaveCount(0)

            // Click "All" for Role to reset
            await mainContent.getByText('Role:', { exact: true }).locator('..').getByRole('button', { name: 'All', exact: true }).click()
            await page.waitForLoadState('networkidle')

            // Both visible again
            await expect(mainContent.locator('.glass-card.p-4').filter({ hasText: ADMIN_LOGIN }).first()).toBeVisible({ timeout: 5_000 })
            await expect(mainContent.locator('.glass-card.p-4').filter({ hasText: 'filtertest' }).first()).toBeVisible()
        } finally {
            if (user?.id) await apiDeleteUser(page, user.id)
        }
    })

    test('provider filter shows only matching users', async ({ authenticatedPage: page }) => {
        await goToUsersTab(page)
        const mainContent = page.locator('main')

        // Click "local" filter — admin is a local user
        await mainContent.getByRole('button', { name: 'local' }).click()
        await page.waitForLoadState('networkidle')

        // Admin should still be visible (it's a local user)
        await expect(mainContent.getByText(ADMIN_LOGIN).first()).toBeVisible({ timeout: 5_000 })

        // Click "google" — no google users exist
        await mainContent.getByRole('button', { name: 'google' }).click()
        await page.waitForLoadState('networkidle')

        // Should show "No users"
        await expect(mainContent.getByText('No users')).toBeVisible({ timeout: 5_000 })
    })
})

test.describe('Admin user uploads quick link', () => {
    test('view uploads button switches to uploads filtered by user', async ({ authenticatedPage: page }) => {
        // First upload a file so there's data
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const buffer = Buffer.from('e2e test content')
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'e2e-test.txt',
            mimeType: 'text/plain',
            buffer,
        })
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })

        // Go to admin users tab
        await goToUsersTab(page)
        const mainContent = page.locator('main')

        // Click the View Uploads (📁) button on the admin user card
        const userCard = mainContent.locator('.glass-card.p-4').filter({ hasText: ADMIN_LOGIN }).first()
        await expect(userCard).toBeVisible({ timeout: 5_000 })
        // Use title attribute since emoji may not render in headless Chromium
        const viewUploadsBtn = userCard.locator('button').filter({ hasText: /📁/ }).or(
            userCard.locator('button[title*="uploads" i]')
        ).first()
        await viewUploadsBtn.click()
        await page.waitForLoadState('networkidle')

        // Should switch to uploads view with user filter chip
        await expect(mainContent.getByText('user:').first()).toBeVisible({ timeout: 5_000 })
        await expect(mainContent.getByText(ADMIN_LOGIN).first()).toBeVisible()
    })
})

test.describe('Admin user search', () => {
    test('search input is visible on users tab', async ({ authenticatedPage: page }) => {
        await goToUsersTab(page)
        const searchInput = page.getByPlaceholder(/search users/i)
        await expect(searchInput).toBeVisible({ timeout: 5_000 })
    })

    test('typing in search shows dropdown with results', async ({ authenticatedPage: page }) => {
        // Create a user to search for
        const user = await apiCreateUser(page, 'searchuser', 'searchuser123', false)

        try {
            await goToUsersTab(page)
            const searchInput = page.getByPlaceholder(/search users/i)

            // Type a query and wait for the search API response
            const searchPromise = page.waitForResponse(resp =>
                resp.url().includes('/users/search') && resp.status() === 200
            )
            await searchInput.fill('searchuser')
            const response = await searchPromise

            // Verify the dropdown appeared with results
            const mainContent = page.locator('main')
            const dropdown = mainContent.locator('.absolute.z-20')
            await expect(dropdown).toBeVisible({ timeout: 5_000 })

            // Click the result — should filter user list to that user
            const resultBtn = dropdown.getByRole('button').filter({ hasText: 'searchuser' }).first()
            await expect(resultBtn).toBeVisible()
            await resultBtn.click()

            // Dropdown should close and user list should show only the selected user
            await expect(dropdown).not.toBeVisible()
            const userCards = mainContent.locator('.glass-card').filter({ hasText: 'searchuser' })
            await expect(userCards).toHaveCount(1)
        } finally {
            if (user?.id) await apiDeleteUser(page, user.id)
        }
    })
})

test.describe('Admin result counts', () => {
    test('users view shows result count', async ({ authenticatedPage: page }) => {
        await goToUsersTab(page)
        const mainContent = page.locator('main')
        await expect(mainContent.getByText(/Showing \d+ of \d+ users/)).toBeVisible({ timeout: 5_000 })
    })

    test('uploads view shows result count', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
        const mainContent = page.locator('main')
        await expect(mainContent.getByText(/Showing \d+ of \d+ uploads/)).toBeVisible({ timeout: 5_000 })
    })
})
