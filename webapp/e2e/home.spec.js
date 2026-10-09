import { test, expect, uploadTestFile, ADMIN_LOGIN, ADMIN_PASSWORD } from './fixtures.js'

test.describe('Home view', () => {
    test('shows user sidebar info', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        // Should show the admin user's login in the sidebar user card
        // Use a more specific locator to avoid matching nav links
        await expect(page.locator('.glass-card').filter({ hasText: 'admin' }).first()).toBeVisible()
    })

    test('uploads tab shows upload cards', async ({ authenticatedPage: page }) => {
        // Create an upload first so there's something to show
        await uploadTestFile(page, 'home-test.txt', 'for home view')

        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        // Click Uploads sidebar nav button
        await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
        await page.waitForLoadState('networkidle')

        // Should show at least one upload card with the file name
        await expect(page.getByText('home-test.txt')).toBeVisible({ timeout: 5_000 })
    })

    test('tokens tab renders', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        // Click Tokens sidebar nav button
        await page.getByRole('button', { name: 'Tokens', exact: true }).click()
        await page.waitForLoadState('networkidle')

        // Should show the token creation area with "Create token" button
        await expect(page.getByRole('button', { name: /Create token/i })).toBeVisible({ timeout: 5_000 })
    })

    test('create token', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        // Go to Tokens tab
        await page.getByRole('button', { name: 'Tokens', exact: true }).click()
        await page.waitForLoadState('networkidle')

        // Create a new token
        await page.getByRole('button', { name: /Create token/i }).click()
        await page.waitForLoadState('networkidle')

        // A token row should now be visible (tokens are long hex strings)
        // A "Revoke" button should appear for the new token
        await expect(page.getByRole('button', { name: /Revoke/i }).first()).toBeVisible({ timeout: 5_000 })
    })

    test('delete token uploads', async ({ authenticatedPage: page }) => {
        // Create a token via API
        const tokenData = await page.evaluate(async () => {
            const xsrf = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)?.[1] || ''
            const headers = { 'Content-Type': 'application/json' }
            if (xsrf) headers['X-XSRFToken'] = xsrf
            const r = await fetch('/me/token', {
                method: 'POST',
                credentials: 'same-origin',
                headers,
                body: JSON.stringify({}),
            })
            return r.json()
        })

        // Create an upload linked to that token via API
        await page.evaluate(async (token) => {
            const xsrf = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)?.[1] || ''
            const headers = { 'Content-Type': 'application/json', 'X-PlikToken': token }
            if (xsrf) headers['X-XSRFToken'] = xsrf
            const r = await fetch('/upload', {
                method: 'POST',
                credentials: 'same-origin',
                headers,
                body: JSON.stringify({}),
            })
            if (!r.ok) throw new Error(`Create upload failed: ${r.status}`)
        }, tokenData.token)

        // Navigate to tokens tab
        await page.goto('/#/home/tokens')
        await page.waitForLoadState('networkidle')

        // Click the first "Delete Uploads" button
        const deleteBtn = page.getByRole('button', { name: /Delete Uploads/i }).first()
        await expect(deleteBtn).toBeVisible({ timeout: 5_000 })
        await deleteBtn.click()

        // Confirm dialog should appear
        await expect(page.getByText('The token itself will not be revoked')).toBeVisible({ timeout: 5_000 })
        await page.getByRole('button', { name: 'Confirm' }).click()

        // Success toast should appear with "uploads removed"
        await expect(page.getByText(/uploads? removed/i)).toBeVisible({ timeout: 5_000 })

        // Toast should auto-dismiss
        await expect(page.getByText(/uploads? removed/i)).not.toBeVisible({ timeout: 5_000 })
    })

    test('clicking token navigates to uploads with token filter', async ({ authenticatedPage: page }) => {
        // Create a token with a comment via API
        const tokenData = await page.evaluate(async () => {
            const xsrf = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)?.[1] || ''
            const headers = { 'Content-Type': 'application/json' }
            if (xsrf) headers['X-XSRFToken'] = xsrf
            const r = await fetch('/me/token', {
                method: 'POST',
                credentials: 'same-origin',
                headers,
                body: JSON.stringify({ comment: 'filter-test' }),
            })
            return r.json()
        })

        // Create an upload linked to that token
        await page.evaluate(async (token) => {
            const xsrf = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)?.[1] || ''
            const headers = { 'Content-Type': 'application/json', 'X-PlikToken': token }
            if (xsrf) headers['X-XSRFToken'] = xsrf
            await fetch('/upload', {
                method: 'POST',
                credentials: 'same-origin',
                headers,
                body: JSON.stringify({}),
            })
        }, tokenData.token)

        // Navigate to tokens tab
        await page.goto('/#/home/tokens')
        await page.waitForLoadState('networkidle')

        // Click the token string link (the button with the token text)
        const tokenBtn = page.getByRole('button', { name: new RegExp(tokenData.token.substring(0, 8)) }).first()
        await expect(tokenBtn).toBeVisible({ timeout: 5_000 })
        await tokenBtn.click()

        // Should navigate to uploads tab
        await page.waitForURL(/home\/uploads/, { timeout: 5_000 })

        // Token filter chip should be visible with the truncated token string
        await expect(page.getByText(tokenData.token.substring(0, 12))).toBeVisible({ timeout: 5_000 })
    })

    test('upload card shows removed file with status badge', async ({ authenticatedPage: page }) => {
        // Upload a file
        await uploadTestFile(page, 'badge-test.txt', 'badge test content')

        // Delete the file on the download page
        const removeBtn = page.getByTitle('Remove file').first()
        await removeBtn.click()
        const dialog = page.locator('.fixed.inset-0.z-50 .glass-card')
        await expect(dialog).toBeVisible({ timeout: 3_000 })
        await dialog.getByRole('button', { name: 'Delete' }).click()
        await expect(page.getByText('Removed')).toBeVisible({ timeout: 5_000 })

        // Navigate to Home → Uploads tab
        await page.goto('/#/home/uploads')
        await page.waitForLoadState('networkidle')

        // The file should appear with line-through styling (span, not link)
        const fileName = page.locator('.line-through').filter({ hasText: 'badge-test.txt' })
        await expect(fileName).toBeVisible({ timeout: 5_000 })

        // The single-letter status badge (r or d) should be visible
        const badge = page.locator('[title="Removed"], [title="Deleted"]')
        await expect(badge).toBeVisible()
        await expect(badge).toHaveText(/^[rd]$/)

        // The file name should NOT be a download link
        await expect(page.getByRole('link', { name: 'badge-test.txt' })).not.toBeVisible()
    })
})

test.describe('User info card', () => {
    test('shows user login', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        const card = page.locator('aside .glass-card').first()
        await expect(card).toBeVisible({ timeout: 5_000 })
        await expect(card).toContainText('admin')
    })

    test('shows provider', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        const card = page.locator('aside .glass-card').first()
        await expect(card).toBeVisible({ timeout: 5_000 })
        await expect(card).toContainText('local')
    })

    test('admin badge shown for admin user', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        // The admin badge is a green rounded-full span with text "admin"
        const badge = page.locator('aside .glass-card .rounded-full').filter({ hasText: 'admin' })
        await expect(badge).toBeVisible({ timeout: 5_000 })
        // Verify the green styling
        await expect(badge).toHaveClass(/bg-emerald/)
    })
})

test.describe('User configuration panel', () => {
    test('shows user config labels', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        // Stats view (default) should show User Configuration panel
        const configPanel = page.locator('.glass-card').filter({ hasText: 'User Configuration' })
        await expect(configPanel).toBeVisible({ timeout: 5_000 })
        await expect(configPanel.getByText('Max File Size')).toBeVisible()
        await expect(configPanel.getByText('Max User Size')).toBeVisible()
        await expect(configPanel.getByText('Default TTL')).toBeVisible()
        await expect(configPanel.getByText('Max TTL')).toBeVisible()
    })
})

test.describe('User statistics panel', () => {
    test('shows user stats labels and values', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        const statsPanel = page.locator('.glass-card').filter({ hasText: 'User Statistics' })
        await expect(statsPanel).toBeVisible({ timeout: 5_000 })

        // Check labels
        await expect(statsPanel.getByText('Current Usage', { exact: true })).toBeVisible()
        await expect(statsPanel.getByText(/^Lifetime Usage \(since /)).toBeVisible()
        await expect(statsPanel.getByText('Uploads', { exact: true })).toHaveCount(2)
        await expect(statsPanel.getByText('Files', { exact: true })).toHaveCount(2)
        await expect(statsPanel.getByText('Total Size', { exact: true })).toHaveCount(2)

        // Check that stat values are present and not NaN
        const values = statsPanel.locator('.text-2xl.font-bold')
        const count = await values.count()
        expect(count).toBeGreaterThanOrEqual(6)

        for (let i = 0; i < count; i++) {
            const text = await values.nth(i).textContent()
            expect(text).toBeTruthy()
            expect(text).not.toBe('NaN')
        }
    })
})

test.describe('Edit account button', () => {
    test('visible for local provider', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        const btn = page.getByRole('button', { name: 'Edit account', exact: true })
        await expect(btn).toBeVisible({ timeout: 5_000 })
    })

    test('opens edit account modal', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        await page.getByRole('button', { name: 'Edit account', exact: true }).click()
        await expect(page.getByRole('heading', { name: 'Edit Account' })).toBeVisible({ timeout: 5_000 })
    })
})

test.describe('Delete account', () => {
    test('button visible when feature enabled (default)', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        await expect(
            page.getByRole('button', { name: 'Delete account', exact: true })
        ).toBeVisible({ timeout: 5_000 })
    })

    test('button hidden when feature disabled', async ({ authenticatedPage: page, withConfig }) => {
        await withConfig({ feature_delete_account: 'disabled' })
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        await expect(
            page.getByRole('button', { name: 'Delete account', exact: true })
        ).not.toBeVisible()
    })

    test('shows confirmation dialog', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        await page.getByRole('button', { name: 'Delete account', exact: true }).click()

        // Confirm dialog should appear with warning text
        await expect(page.getByText('Delete your account and ALL uploads')).toBeVisible({ timeout: 5_000 })
        await expect(page.getByRole('button', { name: 'Confirm' })).toBeVisible()
        await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()

        // Cancel — should stay on home
        await page.getByRole('button', { name: 'Cancel' }).click()
        await expect(page.getByText('Delete your account and ALL uploads')).not.toBeVisible()
    })

    test('deletes throwaway user and redirects to upload page', async ({ page }) => {
        // Create a throwaway user via admin API
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Login as admin first to create the user
        const xsrfCookie = await page.evaluate(() => {
            const match = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
            return match ? match[1] : ''
        })

        const headers = { 'Content-Type': 'application/json' }
        if (xsrfCookie) headers['X-XSRFToken'] = xsrfCookie

        await page.evaluate(async ({ creds, headers }) => {
            await fetch('/auth/local/login', {
                method: 'POST',
                credentials: 'same-origin',
                headers,
                body: JSON.stringify(creds),
            })
        }, { creds: { login: ADMIN_LOGIN, password: ADMIN_PASSWORD }, headers })

        // Re-read XSRF after login
        const xsrf2 = await page.evaluate(() => {
            const match = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
            return match ? match[1] : ''
        })
        const adminHeaders = { 'Content-Type': 'application/json' }
        if (xsrf2) adminHeaders['X-XSRFToken'] = xsrf2

        // Create throwaway user
        await page.evaluate(async ({ headers }) => {
            const r = await fetch('/user', {
                method: 'POST',
                credentials: 'same-origin',
                headers,
                body: JSON.stringify({
                    provider: 'local',
                    login: 'deleteme',
                    password: 'deleteme123',
                }),
            })
            if (r.status !== 200) throw new Error(`Create user failed: ${r.status}`)
        }, { headers: adminHeaders })

        // Logout admin
        await page.evaluate(async ({ headers }) => {
            await fetch('/auth/local/logout', {
                method: 'POST',
                credentials: 'same-origin',
                headers,
            })
        }, { headers: adminHeaders })

        // Login as throwaway user
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const xsrf3 = await page.evaluate(() => {
            const match = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
            return match ? match[1] : ''
        })
        const userHeaders = { 'Content-Type': 'application/json' }
        if (xsrf3) userHeaders['X-XSRFToken'] = xsrf3

        await page.evaluate(async ({ creds, headers }) => {
            const r = await fetch('/auth/local/login', {
                method: 'POST',
                credentials: 'same-origin',
                headers,
                body: JSON.stringify(creds),
            })
            if (r.status !== 200) throw new Error(`Login failed: ${r.status}`)
        }, { creds: { login: 'deleteme', password: 'deleteme123' }, headers: userHeaders })

        await page.reload({ waitUntil: 'networkidle' })

        // Navigate to home and delete account
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        await page.getByRole('button', { name: 'Delete account', exact: true }).click()
        await expect(page.getByText('Delete your account and ALL uploads')).toBeVisible({ timeout: 5_000 })

        await page.getByRole('button', { name: 'Confirm' }).click()
        await page.waitForLoadState('networkidle')

        // Should be redirected to upload page (session cleared)
        await expect(page.getByRole('heading', { name: 'Upload Settings' })).toBeVisible({ timeout: 5_000 })

        // Trying to access /home should redirect to login (no longer authenticated)
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')
        await expect(page.getByText('Sign in to your account')).toBeVisible({ timeout: 5_000 })
    })
})
