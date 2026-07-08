import { test, expect } from './fixtures.js'

test.describe('Theme Picker — visibility', () => {
    test('picker is visible when multiple themes are available (default: all built-ins)', async ({ page, withThemes }) => {
        await withThemes(["*"])  // wildcard = all built-ins
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toBeVisible()
    })

    test('picker is hidden when only one theme is configured', async ({ page, withThemes }) => {
        await withThemes(['dark'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toHaveCount(0)
    })

    test('picker is hidden when themes is an empty-equivalent single entry', async ({ page, withThemes }) => {
        await withThemes(['auto'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toHaveCount(0)
    })

    test('picker shows when exactly two themes configured', async ({ page, withThemes }) => {
        await withThemes(['dark', 'light'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toBeVisible()
    })
})

test.describe('Theme Picker — dropdown', () => {
    test('clicking opens dropdown with theme options', async ({ page, withThemes }) => {
        await withThemes(['dark', 'light', 'nord'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Dropdown should not be visible initially
        await expect(page.locator('#theme-option-dark')).toHaveCount(0)

        // Click the picker toggle
        await page.locator('#theme-picker-toggle').click()

        // Dropdown should now show the configured themes
        await expect(page.locator('#theme-option-dark')).toBeVisible()
        await expect(page.locator('#theme-option-light')).toBeVisible()
        await expect(page.locator('#theme-option-nord')).toBeVisible()

        // Themes NOT in the list should not appear
        await expect(page.locator('#theme-option-matrix')).toHaveCount(0)
    })

    test('clicking outside closes the dropdown', async ({ page, withThemes }) => {
        await withThemes(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()
        await expect(page.locator('#theme-option-dark')).toBeVisible()

        // Click outside (on the body)
        await page.locator('body').click({ position: { x: 10, y: 300 } })

        // Dropdown should close
        await expect(page.locator('#theme-option-dark')).toHaveCount(0)
    })

    test('selecting a theme closes the dropdown', async ({ page, withThemes }) => {
        await withThemes(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-light').click()

        // Dropdown should close after selection
        await expect(page.locator('#theme-option-light')).toHaveCount(0)
    })
})

test.describe('Theme Picker — theme application', () => {
    test('selecting a theme applies it via data-theme', async ({ page, withThemes }) => {
        await withThemes(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-light').click()

        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).toBe('light')
    })

    test('selected theme injects the CSS file', async ({ page, withThemes }) => {
        await withThemes(['dark', 'light', 'nord'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()
        const nordBtn = page.locator('#theme-option-nord')
        await expect(nordBtn).toBeVisible()
        await nordBtn.click()

        // Wait for dropdown to close (confirms click was processed)
        await expect(nordBtn).toHaveCount(0)

        // Use auto-retrying assertion for data-theme
        await expect(page.locator('html')).toHaveAttribute('data-theme', 'nord')

        const link = page.locator('link[href*="nord.css"]')
        await expect(link).toHaveCount(1)
    })

    test('active theme shows checkmark', async ({ page, withThemes }) => {
        await withThemes(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Open picker and select nord
        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-nord').click()

        // Reopen picker — nord should have the accent color (checkmark)
        await page.locator('#theme-picker-toggle').click()
        const nordOption = page.locator('#theme-option-nord')
        await expect(nordOption).toHaveClass(/text-accent-400/)
    })
})

test.describe('Theme Picker — localStorage persistence', () => {
    test('selected theme persists across page reload', async ({ page, withThemes }) => {
        await withThemes(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Select light theme
        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-light').click()

        // Verify localStorage was set
        const stored = await page.evaluate(() => localStorage.getItem('plik-theme'))
        expect(stored).toBe('light')

        // Reload the page (route intercept persists)
        await page.reload({ waitUntil: 'networkidle' })

        // Theme should still be light
        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).toBe('light')
    })

    test('localStorage theme overrides settings.json default', async ({ page, withThemes }) => {
        // Pre-seed localStorage before any navigation
        await page.addInitScript(() => localStorage.setItem('plik-theme', 'nord'))

        await withThemes(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // data-theme should be 'nord' from localStorage despite settings.json default being 'auto'
        await expect(page.locator('html')).toHaveAttribute('data-theme', 'nord')
    })

    test('stale localStorage is ignored when theme is not in available list', async ({ page, withThemes }) => {
        // Pre-seed localStorage with a theme not in the restricted list
        await page.addInitScript(() => localStorage.setItem('plik-theme', 'auto'))

        await withThemes(['nord'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Should fall back to the only available theme, not the stale 'auto'
        await expect(page.locator('html')).toHaveAttribute('data-theme', 'nord')
    })
})

test.describe('Theme Picker — custom theme objects', () => {
    test('supports object entries with custom labels', async ({ page, withThemes }) => {
        await withThemes([
            'dark',
            { name: 'nord', label: 'Arctic Night' },
        ])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()

        // The custom label should appear
        const nordOption = page.locator('#theme-option-nord')
        await expect(nordOption).toBeVisible()
        await expect(nordOption).toContainText('Arctic Night')
    })
})

test.describe('Theme Picker — layout', () => {
    test('has "Theme" text label in the button', async ({ page, withThemes }) => {
        await withThemes(["*"])
        // Use wider viewport so the desktop nav is visible
        await page.setViewportSize({ width: 1280, height: 720 })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toBeVisible()
        await expect(picker).toContainText('Theme')
    })
})

test.describe('Theme Picker — defaultDarkTheme / defaultLightTheme', () => {
    test('auto resolves to defaultDarkTheme when OS prefers dark', async ({ page, withThemes }) => {
        await page.emulateMedia({ colorScheme: 'dark' })
        await withThemes(["*"], { defaultDarkTheme: 'solarized-dark' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.locator('html')).toHaveAttribute('data-theme', 'solarized-dark')
    })

    test('auto resolves to defaultLightTheme when OS prefers light', async ({ page, withThemes }) => {
        await page.emulateMedia({ colorScheme: 'light' })
        await withThemes(["*"], { defaultLightTheme: 'solarized-light' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.locator('html')).toHaveAttribute('data-theme', 'solarized-light')
    })

    test('custom default theme injects its CSS file', async ({ page, withThemes }) => {
        await page.emulateMedia({ colorScheme: 'dark' })
        await withThemes(["*"], { defaultDarkTheme: 'nord' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.locator('html')).toHaveAttribute('data-theme', 'nord')
        const link = page.locator('link[href*="nord.css"]')
        await expect(link).toHaveCount(1)
    })

    test('missing defaults fall back to dark/light', async ({ page, withThemes }) => {
        // Default settings (no defaultDarkTheme/defaultLightTheme specified)
        await page.emulateMedia({ colorScheme: 'dark' })
        await withThemes(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')

        // Switch OS preference to light and reload
        await page.emulateMedia({ colorScheme: 'light' })
        await page.reload({ waitUntil: 'networkidle' })

        await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
    })
})

test.describe('Theme Picker — wildcard themes', () => {
    test('"*" expands to all built-in themes plus custom entries', async ({ page, withThemes }) => {
        await withThemes(['*', { name: 'acme', label: 'Acme Corp' }])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()

        // Built-in themes should be present
        await expect(page.locator('#theme-option-auto')).toBeVisible()
        await expect(page.locator('#theme-option-dark')).toBeVisible()
        await expect(page.locator('#theme-option-light')).toBeVisible()
        await expect(page.locator('#theme-option-nord')).toBeVisible()

        // Custom theme should also appear
        await expect(page.locator('#theme-option-acme')).toBeVisible()
        await expect(page.locator('#theme-option-acme')).toContainText('Acme Corp')
    })
})

// FLAKY: pre-existing flake (verified on the clean base commit, not caused by
// the improved-stats branch), ~25% failure rate observed over 20 runs
// (`--repeat-each=20 --workers=1`), always the same failure: `html`'s
// data-theme stays "nord" instead of "light", stable across the assertion's
// full 5s auto-retry window (not a slow-to-settle value). Root cause is a
// genuine app-code race in webapp/src/main.js's `Promise.all([loadConfig(),
// loadSettings(), checkSession()])`: checkSession() -> syncThemeFromUser()
// (webapp/src/authStore.js / webapp/src/settings.js) validates the user's DB
// theme against settings.themes, which can still hold its unrestricted
// module-level default (['*']) if /me resolves before /settings.json is
// applied, so 'nord' passes validation and gets set permanently — nothing
// re-validates it once settings.themes narrows afterward. This is a
// product-code ordering bug, not a missing wait in this test (the assertion
// below already uses Playwright's auto-retrying toHaveAttribute). Diagnosed,
// not fixed, per the bounded investigation rule. See task-19-report.md for
// repro evidence.
test.describe('Theme Picker — stale DB theme', () => {
    test('user theme from DB is ignored when not in available themes list', async ({ authenticatedPage: page, withThemes }) => {
        // Set the user's theme to 'nord' via PATCH /me
        await page.evaluate(async () => {
            const xsrfMatch = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
            const xsrf = xsrfMatch ? xsrfMatch[1] : ''
            const headers = { 'Content-Type': 'application/json' }
            if (xsrf) headers['X-XSRFToken'] = xsrf

            await fetch('/me', {
                method: 'PATCH',
                credentials: 'same-origin',
                headers,
                body: JSON.stringify({ theme: 'nord' }),
            })
        })

        // Now restrict available themes to only 'dark' — 'nord' is NOT in the list
        await withThemes(['light'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // The stale DB theme ('nord') should be ignored — 'dark' should be applied
        await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
    })
})
