import { test, expect } from './fixtures.js'

test.describe('Error display components', () => {
    test.describe('Download view — ErrorState', () => {
        test('shows error state for invalid upload ID', async ({ page }) => {
            await page.goto('/#/?id=nonexistentuploadid')
            await page.waitForLoadState('networkidle')

            // ErrorState component should be visible with error message
            const errorCard = page.locator('.glass-card.border-danger-500\\/50')
            await expect(errorCard).toBeVisible({ timeout: 5_000 })

            // Should contain the error message text
            await expect(errorCard).toContainText('not found')

            // Should have a "Try again" button
            const retryBtn = errorCard.getByRole('button', { name: 'Try again' })
            await expect(retryBtn).toBeVisible()
        })

        test('retry button re-fetches upload', async ({ page }) => {
            await page.goto('/#/?id=nonexistentuploadid')
            await page.waitForLoadState('networkidle')

            const errorCard = page.locator('.glass-card.border-danger-500\\/50')
            await expect(errorCard).toBeVisible({ timeout: 5_000 })

            // Click retry — should still show error (upload doesn't exist)
            const retryPromise = page.waitForResponse(resp =>
                resp.url().includes('/upload/nonexistentuploadid')
            )
            await errorCard.getByRole('button', { name: 'Try again' }).click()
            await retryPromise

            // Error should still be visible (upload still doesn't exist)
            await expect(errorCard).toBeVisible()
        })
    })

    test.describe('Home view — ErrorBanner', () => {
        test('shows error banner when API fails', async ({ authenticatedPage: page }) => {
            // Intercept the uploads API to return 500
            await page.route('**/me/uploads*', (route) => {
                route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    body: JSON.stringify({ message: 'Internal Server Error' }),
                })
            })

            await page.goto('/#/home')
            await page.waitForLoadState('networkidle')

            // Click Uploads tab to trigger the intercepted call
            await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
            await page.waitForLoadState('networkidle')

            // ErrorBanner should be visible
            const banner = page.locator('main .glass-card.border-danger-500\\/50')
            await expect(banner).toBeVisible({ timeout: 5_000 })
            await expect(banner).toContainText('Could not load uploads')

            // Click dismiss button (× icon)
            await banner.locator('button').last().click()

            // Banner should disappear
            await expect(banner).not.toBeVisible()
        })
    })

    test.describe('Admin view — ErrorBanner', () => {
        test('shows error banner when API fails', async ({ authenticatedPage: page }) => {
            // Intercept the admin uploads API to return 500
            await page.route('**/uploads?*', (route) => {
                const url = route.request().url()
                // Only intercept admin uploads listing, not individual upload fetches
                if (url.includes('/uploads?') || url.endsWith('/uploads')) {
                    route.fulfill({
                        status: 500,
                        contentType: 'application/json',
                        body: JSON.stringify({ message: 'Internal Server Error' }),
                    })
                } else {
                    route.continue()
                }
            })

            await page.goto('/#/admin')
            await page.waitForLoadState('networkidle')

            // Click Uploads tab to trigger the intercepted call
            await page.locator('aside').getByRole('button', { name: 'Uploads', exact: true }).click()
            await page.waitForLoadState('networkidle')

            // ErrorBanner should be visible
            const banner = page.locator('main .glass-card.border-danger-500\\/50')
            await expect(banner).toBeVisible({ timeout: 5_000 })
            await expect(banner).toContainText('Could not load uploads')

            // Click dismiss button (× icon)
            await banner.locator('button').last().click()

            // Banner should disappear
            await expect(banner).not.toBeVisible()
        })
    })
})
