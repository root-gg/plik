import { randomBytes } from 'node:crypto'
import { test, expect, uploadTestFile } from './fixtures.js'

/**
 * Tests for upload failure handling and retry functionality.
 *
 * Strategy: intercept resumable upload appends (PATCH /resumable/*) via page.route()
 * to simulate server errors that end the upload (Upload-Complete: ?1), verify the
 * error UI, then unroute and retry.
 */
/**
 * Loses the responses to the first upload creation and the first append, which the server
 * processes, and drops the second append before it reaches the server.
 */
async function interruptResumableUploads(page) {
    let creations = 0
    let appends = 0
    await page.route('**/resumable/**', async (route) => {
        const method = route.request().method()
        if (method === 'POST') creations++
        if (method === 'PATCH') appends++
        if ((method === 'POST' && creations === 1) || (method === 'PATCH' && appends === 1)) {
            const response = await route.fetch()
            expect(response.ok()).toBe(true)
            return route.abort('connectionreset')
        }
        if (method === 'PATCH' && appends === 2) {
            return route.abort('connectionreset')
        }
        return route.continue()
    })
}

/**
 * Waits for an uploaded file in the file list and checks that its download matches the content.
 */
async function expectDownload(page, fileName, content) {
    const link = page.getByRole('link', { name: fileName })
    await expect(link).toBeVisible({ timeout: 20_000 })
    await expect(page.getByRole('button', { name: 'Retry', exact: true })).not.toBeVisible()

    const download = await page.request.get(await link.getAttribute('href'))
    expect(download.ok()).toBe(true)
    expect(Buffer.compare(await download.body(), content)).toBe(0)
}

test.describe('Upload failure and retry', () => {
    test('failed upload shows error message and Retry button', async ({ page }) => {
        // Intercept file upload XHR with a 500 error
        await page.route('**/resumable/**', (route) => {
            if (route.request().method() === 'PATCH') {
                return route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    headers: { 'Upload-Complete': '?1' },
                    body: JSON.stringify({ message: 'Internal Server Error' }),
                })
            }
            return route.continue()
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'fail-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('will fail'),
        })

        // Click Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()

        // Wait for navigation to download view
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // File should show error message and Retry button
        await expect(page.getByText(/upload failed|internal server error/i).first()).toBeVisible({ timeout: 10_000 })
        await expect(page.getByRole('button', { name: 'Retry' }).first()).toBeVisible()
    })

    test('Retry button re-uploads the file successfully', async ({ page }) => {
        // Intercept to fail on first attempt
        let shouldFail = true
        await page.route('**/resumable/**', (route) => {
            if (route.request().method() === 'PATCH' && shouldFail) {
                return route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    headers: { 'Upload-Complete': '?1' },
                    body: JSON.stringify({ message: 'Temporary failure' }),
                })
            }
            return route.continue()
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'retry-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('retry me'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Wait for error
        await expect(page.getByRole('button', { name: 'Retry' }).first()).toBeVisible({ timeout: 10_000 })

        // Remove the failure intercept and retry
        shouldFail = false
        await page.getByRole('button', { name: 'Retry' }).first().click()

        // File should upload and error should disappear
        await expect(page.getByRole('button', { name: 'Retry' })).not.toBeVisible({ timeout: 10_000 })

        // The file should now be in the uploaded files list
        await expect(page.getByText('retry-test.txt').first()).toBeVisible()
    })

    test('Retry Failed button retries all failed files', async ({ page }) => {
        // Intercept to fail on all initial attempts
        let shouldFail = true
        await page.route('**/resumable/**', (route) => {
            if (route.request().method() === 'PATCH' && shouldFail) {
                return route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    headers: { 'Upload-Complete': '?1' },
                    body: JSON.stringify({ message: 'Batch failure' }),
                })
            }
            return route.continue()
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload 2 files
        const input = page.locator('input[type="file"]')
        await input.setInputFiles([
            { name: 'batch1.txt', mimeType: 'text/plain', buffer: Buffer.from('file 1') },
            { name: 'batch2.txt', mimeType: 'text/plain', buffer: Buffer.from('file 2') },
        ])

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Wait for both to fail
        await expect(page.getByRole('button', { name: 'Retry' }).first()).toBeVisible({ timeout: 10_000 })

        // "Retry Failed" button should appear
        const retryAllBtn = page.getByRole('button', { name: /Retry Failed/i })
        await expect(retryAllBtn).toBeVisible({ timeout: 5_000 })

        // Remove failure and click Retry Failed
        shouldFail = false
        await retryAllBtn.click()

        // All retry buttons should disappear
        await expect(page.getByRole('button', { name: 'Retry' }).first()).not.toBeVisible({ timeout: 10_000 })

        // Both files should be uploaded
        await expect(page.getByText('batch1.txt').first()).toBeVisible()
        await expect(page.getByText('batch2.txt').first()).toBeVisible()
    })

    for (const action of ['Cancel upload', 'Cancel All']) {
        test(`${action} removes the file after a delayed cancellation`, async ({ page }) => {
            let started
            const transferStarted = new Promise(resolve => { started = resolve })
            await page.route('**/resumable/**', async (route) => {
                if (route.request().method() === 'PATCH') {
                    started()
                    return
                }
                if (route.request().method() === 'DELETE') {
                    await new Promise(resolve => setTimeout(resolve, 500))
                }
                return route.continue()
            })

            await page.goto('/')
            await page.waitForLoadState('networkidle')
            await page.locator('input[type="file"]').setInputFiles({
                name: 'cancel-test.bin',
                mimeType: 'application/octet-stream',
                buffer: Buffer.from('cancel me'),
            })

            await page.getByRole('button', { name: 'Upload', exact: true }).click()
            await transferStarted
            await page.getByRole('button', { name: action, exact: true }).click()

            await expect(page.getByText('Removed', { exact: true })).toBeVisible()
            const id = page.url().match(/[?&]id=([^&]+)/)[1]
            await expect.poll(async () => {
                const upload = await (await page.request.get(`/upload/${id}`)).json()
                return upload.files.map(file => file.status)
            }).toEqual(['deleted'])
        })
    }

    test('interrupted chunks are resumed and the file arrives intact', async ({ page }) => {
        const content = randomBytes(1024 * 1024)
        await interruptResumableUploads(page)

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('input[type="file"]').setInputFiles({
            name: 'resume.bin',
            mimeType: 'application/octet-stream',
            buffer: content,
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })

        await expectDownload(page, 'resume.bin', content)
    })

    test('interrupted chunks of a file added to an upload are resumed', async ({ page }) => {
        await uploadTestFile(page, 'first.txt', 'first file')
        // Wait for the first upload to end, so the Upload button stages the added file
        await expect(page.getByRole('link', { name: 'first.txt' })).toBeVisible({ timeout: 5_000 })
        await page.waitForLoadState('networkidle')
        const content = randomBytes(1024 * 1024)
        await interruptResumableUploads(page)

        await page.locator('input[type="file"]').setInputFiles({
            name: 'added.bin',
            mimeType: 'application/octet-stream',
            buffer: content,
        })
        await page.getByRole('button', { name: 'Upload', exact: true }).click()

        await expectDownload(page, 'added.bin', content)

        // The retried creation did not add the file twice
        const id = page.url().match(/[?&]id=([^&]+)/)[1]
        const upload = await (await page.request.get(`/upload/${id}`)).json()
        expect(upload.files.map(f => f.fileName).sort()).toEqual(['added.bin', 'first.txt'])
    })

    test('Retry resumes an added file from what the server has', async ({ page }) => {
        await uploadTestFile(page, 'first.txt', 'first file')
        // Wait for the first upload to end, so the Upload button stages the added file
        await expect(page.getByRole('link', { name: 'first.txt' })).toBeVisible({ timeout: 5_000 })
        await page.waitForLoadState('networkidle')
        const content = randomBytes(1024 * 1024)
        const offsets = []
        await page.route('**/resumable/**', (route) => {
            if (route.request().method() !== 'PATCH') {
                return route.continue()
            }
            offsets.push(Number(route.request().headers()['upload-offset']))
            if (offsets.length === 2) {
                // An error that stops this attempt without ending the upload
                return route.fulfill({ status: 400, body: 'Bad Request' })
            }
            return route.continue()
        })

        await page.locator('input[type="file"]').setInputFiles({
            name: 'added.bin',
            mimeType: 'application/octet-stream',
            buffer: content,
        })
        await page.getByRole('button', { name: 'Upload', exact: true }).click()

        await page.getByRole('button', { name: 'Retry', exact: true }).click({ timeout: 10_000 })
        await expectDownload(page, 'added.bin', content)

        // The retry continued from the bytes the server already had, in the same file
        expect(offsets[2]).toBeGreaterThan(0)
        const id = page.url().match(/[?&]id=([^&]+)/)[1]
        const upload = await (await page.request.get(`/upload/${id}`)).json()
        expect(upload.files.map(f => f.fileName).sort()).toEqual(['added.bin', 'first.txt'])
    })
})
