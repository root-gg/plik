import { test, expect, uploadTestFile } from './fixtures.js'

test.describe('Navigation and routing', () => {
    test('main upload page loads', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // The Upload link should be visible in the nav bar (it's a nav link, not a button)
        await expect(page.getByRole('link', { name: 'Upload', exact: true })).toBeVisible()
    })

    test('invalid upload ID shows error', async ({ page }) => {
        await page.goto('/#/?id=nonexistent123')
        await page.waitForLoadState('networkidle')

        // Error message should appear in the download view
        await expect(page.locator('.text-danger-500').first()).toBeVisible({ timeout: 5_000 })
    })

    test('clients page loads', async ({ page }) => {
        await page.goto('/#/clients')
        await page.waitForLoadState('networkidle')

        // Should show CLI client page content
        await expect(page.getByText(/client|download|plik/i).first()).toBeVisible()
    })
})

test.describe('Home view tab routing', () => {
    test('default tab is stats', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        // Stats tab should be active (shows User Configuration panel)
        const configPanel = page.locator('.glass-card').filter({ hasText: 'User Configuration' })
        await expect(configPanel).toBeVisible({ timeout: 5_000 })
    })

    test('/home/uploads shows uploads tab', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home/uploads')
        await page.waitForLoadState('networkidle')

        // Should show uploads content (empty state or upload cards)
        const content = page.getByText('No uploads yet').or(page.locator('.glass-card'))
        await expect(content.first()).toBeVisible({ timeout: 5_000 })
    })

    test('/home/tokens shows tokens tab', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home/tokens')
        await page.waitForLoadState('networkidle')

        // Should show token creation input
        await expect(page.getByPlaceholder('Comment (optional)')).toBeVisible({ timeout: 5_000 })
    })

    test('clicking sidebar updates URL path', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home')
        await page.waitForLoadState('networkidle')

        // Click Uploads sidebar button
        await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
        await page.waitForURL(/home\/uploads/)

        // Click Tokens sidebar button
        await page.getByRole('button', { name: 'Tokens', exact: true }).click()
        await page.waitForURL(/home\/tokens/)

        // Click Stats sidebar button
        await page.getByRole('button', { name: 'Stats', exact: true }).click()
        await page.waitForURL(/home\/stats/)
    })

    test('browser back/forward navigates between tabs', async ({ authenticatedPage: page }) => {
        await page.goto('/#/home/stats')
        await page.waitForLoadState('networkidle')

        // Navigate: stats → uploads → tokens (creates history entries)
        await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
        await page.waitForURL(/home\/uploads/)
        await page.getByRole('button', { name: 'Tokens', exact: true }).click()
        await page.waitForURL(/home\/tokens/)
        await expect(page.getByPlaceholder('Comment (optional)')).toBeVisible({ timeout: 5_000 })

        // Back: tokens → uploads
        await page.goBack()
        await page.waitForURL(/home\/uploads/)
        const uploadsContent = page.getByText('No uploads yet').or(page.locator('.glass-card'))
        await expect(uploadsContent.first()).toBeVisible({ timeout: 5_000 })

        // Back: uploads → stats
        await page.goBack()
        await page.waitForURL(/home\/stats/)
        const configPanel = page.locator('.glass-card').filter({ hasText: 'User Configuration' })
        await expect(configPanel).toBeVisible({ timeout: 5_000 })

        // Forward: stats → uploads
        await page.goForward()
        await page.waitForURL(/home\/uploads/)
        await expect(uploadsContent.first()).toBeVisible({ timeout: 5_000 })

        // Forward: uploads → tokens
        await page.goForward()
        await page.waitForURL(/home\/tokens/)
        await expect(page.getByPlaceholder('Comment (optional)')).toBeVisible({ timeout: 5_000 })
    })
})

test.describe('Admin view tab routing', () => {
    test('default tab is stats', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Stats tab should be active (shows Server Configuration panel)
        const configPanel = page.locator('.glass-card').filter({ hasText: 'Server Configuration' })
        await expect(configPanel).toBeVisible({ timeout: 5_000 })
    })

    test('/admin/users shows users tab', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin/users')
        await page.waitForLoadState('networkidle')

        // Should show user search input
        await expect(page.getByPlaceholder(/Search users/)).toBeVisible({ timeout: 5_000 })
    })

    test('/admin/uploads shows uploads tab', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin/uploads')
        await page.waitForLoadState('networkidle')

        // Should show sort controls and uploads content
        const content = page.getByText('No uploads').or(page.locator('.glass-card'))
        await expect(content.first()).toBeVisible({ timeout: 5_000 })
    })

    test('clicking sidebar updates URL path', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin')
        await page.waitForLoadState('networkidle')

        // Click Users sidebar button
        await page.getByRole('button', { name: 'Users', exact: true }).click()
        await page.waitForURL(/admin\/users/)

        // Click Uploads sidebar button
        await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
        await page.waitForURL(/admin\/uploads/)

        // Click Stats sidebar button
        await page.getByRole('button', { name: 'Stats', exact: true }).click()
        await page.waitForURL(/admin\/stats/)
    })

    test('/admin/users?provider=local preserves filter', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin/users?provider=local')
        await page.waitForLoadState('networkidle')

        // The 'local' provider button should be active (has accent color)
        const localBtn = page.getByRole('button', { name: 'local', exact: true })
        await expect(localBtn).toHaveClass(/text-accent/, { timeout: 5_000 })
    })

    test('browser back/forward navigates between tabs', async ({ authenticatedPage: page }) => {
        await page.goto('/#/admin/stats')
        await page.waitForLoadState('networkidle')

        // Navigate: stats → users → uploads (creates history entries)
        await page.getByRole('button', { name: 'Users', exact: true }).click()
        await page.waitForURL(/admin\/users/)
        await expect(page.getByPlaceholder(/Search users/)).toBeVisible({ timeout: 5_000 })

        await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
        await page.waitForURL(/admin\/uploads/)

        // Back: uploads → users
        await page.goBack()
        await page.waitForURL(/admin\/users/)
        await expect(page.getByPlaceholder(/Search users/)).toBeVisible({ timeout: 5_000 })

        // Back: users → stats
        await page.goBack()
        await page.waitForURL(/admin\/stats/)
        const configPanel = page.locator('.glass-card').filter({ hasText: 'Server Configuration' })
        await expect(configPanel).toBeVisible({ timeout: 5_000 })

        // Forward: stats → users
        await page.goForward()
        await page.waitForURL(/admin\/users/)
        await expect(page.getByPlaceholder(/Search users/)).toBeVisible({ timeout: 5_000 })

        // Forward: users → uploads
        await page.goForward()
        await page.waitForURL(/admin\/uploads/)
    })

    test('back/forward preserves filter query params', async ({ authenticatedPage: page }) => {
        // Start on users with a provider filter
        await page.goto('/#/admin/users?provider=local')
        await page.waitForLoadState('networkidle')
        await expect(page.getByRole('button', { name: 'local', exact: true }))
            .toHaveClass(/text-accent/, { timeout: 5_000 })

        // Navigate to uploads tab
        await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
        await page.waitForURL(/admin\/uploads/)

        // Back: should return to users with provider=local still in URL
        await page.goBack()
        await page.waitForURL(/admin\/users/)
        await expect(page).toHaveURL(/provider=local/)
        await expect(page.getByRole('button', { name: 'local', exact: true }))
            .toHaveClass(/text-accent/, { timeout: 5_000 })
    })
})
