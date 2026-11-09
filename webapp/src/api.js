// Plik API Service
// Thin fetch() wrappers for all Plik server endpoints

import i18n from './i18n.js'

const t = (...args) => i18n.global.t(...args)

const base = window.location.origin + window.location.pathname.replace(/\/$/, '')

// Impersonate state — set by authStore, injected in every request
let _impersonateUserId = null

export function setImpersonateUser(userId) {
    _impersonateUserId = userId || null
}

// Read the XSRF token from the plik-xsrf cookie
function getXsrfToken() {
    const match = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
    return match ? match[1] : ''
}

async function apiCall(url, method = 'GET', data = null, headers = {}) {
    const opts = {
        method,
        credentials: 'same-origin',
        headers: {
            'X-ClientApp': 'web_client',
            ...headers,
        },
    }

    // Impersonate header for admin user switching
    if (_impersonateUserId) {
        opts.headers['X-Plik-Impersonate'] = _impersonateUserId
    }

    // XSRF protection: send token on mutating requests
    if (method !== 'GET' && method !== 'HEAD') {
        const xsrf = getXsrfToken()
        if (xsrf) opts.headers['X-XSRFToken'] = xsrf
    }

    if (data && method !== 'GET') {
        opts.headers['Content-Type'] = 'application/json'
        opts.body = JSON.stringify(data)
    }

    let resp
    try {
        resp = await fetch(url, opts)
    } catch (err) {
        // Network errors (offline, DNS, CORS, server down)
        throw { status: 0, message: t('api.networkError'), originalError: err.message }
    }

    if (!resp.ok) {
        let message = t('api.unknownError')
        try {
            const text = await resp.text()
            try {
                const body = JSON.parse(text)
                message = body.message || body || message
            } catch {
                message = text || message
            }
        } catch {
            // Body unreadable, use default message
        }
        throw { status: resp.status, message }
    }

    // Try to parse as JSON first, fall back to text
    const text = await resp.text()
    if (!text) return null

    try {
        return JSON.parse(text)
    } catch {
        // If not valid JSON (like "ok" from DELETE), return the text
        return text
    }
}

// ── Config & Version ──

export function getConfig() {
    return apiCall(`${base}/config`)
}

export function getVersion() {
    return apiCall(`${base}/version`)
}

// ── Authentication ──

export function login(loginName, password) {
    return apiCall(`${base}/auth/local/login`, 'POST', { login: loginName, password })
}

export function oidcLogin() {
    return apiCall(`${base}/auth/oidc/login`)
}

export function logout() {
    return apiCall(`${base}/auth/logout`, 'GET')
}

export function approveCLIAuth(code, comment) {
    return apiCall(`${base}/auth/cli/approve`, 'POST', { code, comment })
}

export function getUser() {
    return apiCall(`${base}/me`)
}

export function deleteAccount() {
    return apiCall(`${base}/me`, 'DELETE')
}

export function patchMe(fields) {
    return apiCall(`${base}/me`, 'PATCH', fields)
}

export function getUserStatistics() {
    return apiCall(`${base}/me/stats`)
}

export function updateUser(user) {
    return apiCall(`${base}/user/${encodeURIComponent(user.id)}`, 'POST', user)
}

// ── Admin ──

export function getServerStats() {
    return apiCall(`${base}/stats`)
}

export function getAdminUsers({ provider, admin, sort, order, after, limit } = {}) {
    const params = new URLSearchParams()
    if (provider) params.set('provider', provider)
    if (admin !== undefined && admin !== '') params.set('admin', admin)
    if (sort) params.set('sort', sort)
    if (order) params.set('order', order)
    if (after) params.set('after', after)
    if (limit) params.set('limit', limit)
    const qs = params.toString()
    return apiCall(`${base}/users${qs ? '?' + qs : ''}`)
}

export function searchUsers({ q, provider, admin, limit } = {}) {
    const params = new URLSearchParams()
    if (q) params.set('q', q)
    if (provider) params.set('provider', provider)
    if (admin !== undefined && admin !== '') params.set('admin', admin)
    if (limit) params.set('limit', limit)
    const qs = params.toString()
    return apiCall(`${base}/users/search${qs ? '?' + qs : ''}`)
}

export function createUser(userData) {
    return apiCall(`${base}/user`, 'POST', userData)
}

export function deleteUser(userId) {
    return apiCall(`${base}/user/${encodeURIComponent(userId)}`, 'DELETE')
}

export function getAdminUploads({ user, token, sort, order, after, limit, oneShot, removable, stream, extendTTL, password, e2ee } = {}) {
    const params = new URLSearchParams()
    if (user) params.set('user', user)
    if (token) params.set('token', token)
    if (sort) params.set('sort', sort)
    if (order) params.set('order', order)
    if (after) params.set('after', after)
    if (limit) params.set('limit', limit)
    if (oneShot === true) params.set('oneShot', 'true')
    if (removable === true) params.set('removable', 'true')
    if (stream === true) params.set('stream', 'true')
    if (extendTTL === true) params.set('extendTTL', 'true')
    if (password === true) params.set('password', 'true')
    if (e2ee === true) params.set('e2ee', 'true')
    const qs = params.toString()
    return apiCall(`${base}/uploads${qs ? '?' + qs : ''}`)
}

// ── User Uploads ──

export function getUserUploads({ token, sort, after, order, limit, oneShot, removable, stream, extendTTL, password, e2ee } = {}) {
    const params = new URLSearchParams()
    if (token) params.set('token', token)
    if (sort) params.set('sort', sort)
    if (after) params.set('after', after)
    if (order) params.set('order', order)
    if (limit) params.set('limit', String(limit))
    if (oneShot === true) params.set('oneShot', 'true')
    if (removable === true) params.set('removable', 'true')
    if (stream === true) params.set('stream', 'true')
    if (extendTTL === true) params.set('extendTTL', 'true')
    if (password === true) params.set('password', 'true')
    if (e2ee === true) params.set('e2ee', 'true')
    const qs = params.toString()
    return apiCall(`${base}/me/uploads${qs ? '?' + qs : ''}`)
}

export function deleteUserUploads(token) {
    const qs = token ? `?token=${encodeURIComponent(token)}` : ''
    return apiCall(`${base}/me/uploads${qs}`, 'DELETE')
}

// ── Tokens ──

export function getUserTokens({ after, limit } = {}) {
    const params = new URLSearchParams()
    if (after) params.set('after', after)
    if (limit) params.set('limit', String(limit))
    const qs = params.toString()
    return apiCall(`${base}/me/token${qs ? '?' + qs : ''}`)
}

export function createToken(comment) {
    return apiCall(`${base}/me/token`, 'POST', comment ? { comment } : {})
}

export function revokeToken(tokenStr) {
    return apiCall(`${base}/me/token/${tokenStr}`, 'DELETE')
}

// ── Upload CRUD ──

export function createUpload(params) {
    return apiCall(`${base}/upload`, 'POST', params)
}

export function getUpload(id, uploadToken) {
    const headers = {}
    if (uploadToken) headers['X-UploadToken'] = uploadToken
    return apiCall(`${base}/upload/${id}`, 'GET', null, headers)
}

export function removeUpload(id, uploadToken) {
    const headers = {}
    if (uploadToken) headers['X-UploadToken'] = uploadToken
    return apiCall(`${base}/upload/${id}`, 'DELETE', null, headers)
}

// ── File Operations ──

/**
 * Upload a file using XMLHttpRequest (for progress tracking)
 * Non-stream files are sent as resumable uploads, in chunks that are retried and resumed when the connection drops.
 * @param {Object} upload - The upload object { id, stream, uploadToken }
 * @param {Object} fileEntry - { id, fileName, file (File object), reference }, a resumable upload sets the id of a new file
 * @param {Function} onProgress - callback(percent)
 * @param {string} basicAuth - optional base64 auth string
 * @param {Function} onStart - optional callback fired when upload connection opens (loadstart)
 * @param {Function} onRetry - optional callback(error) fired when a resumable upload waits to retry after an error
 * @returns {{ promise: Promise<Object>, abort: Function }} - promise resolves to file metadata, abort cancels the upload
 */
export function uploadFile(upload, fileEntry, onProgress, basicAuth, onStart, onRetry) {
    if (!upload.stream && fileEntry.file.size > 0) {
        return uploadFileResumable(upload, fileEntry, onProgress, basicAuth, onRetry)
    }

    const mode = upload.stream ? 'stream' : 'file'
    let url
    if (fileEntry.id) {
        url = `${base}/${mode}/${upload.id}/${fileEntry.id}/${encodeURIComponent(fileEntry.fileName)}`
    } else {
        // Adding file to existing upload
        url = `${base}/${mode}/${upload.id}`
    }

    // Send the file as form data
    const formData = new FormData()
    formData.append('file', fileEntry.file, fileEntry.fileName)

    const request = sendUploadRequest('POST', url, upload, basicAuth, formData, {
        onProgress: onProgress && ((loaded, total) => onProgress(Math.round((loaded / total) * 100))),
        onStart,
    })

    const promise = request.promise.then((xhr) => {
        try {
            return JSON.parse(xhr.responseText)
        } catch {
            return null
        }
    })

    return { promise, abort: () => request.xhr.abort() }
}

// Chunks are sized to take about CHUNK_SECONDS at the measured upload speed: short enough to stay
// within reverse proxy request timeouts (60 s in Traefik) when the bandwidth drops, and to lose
// little work when a chunk fails. MAX_CHUNK_SIZE stays under the 100 MB request limit of Cloudflare.
const CHUNK_SECONDS = 10
const MIN_CHUNK_SIZE = 256 * 1024
const MAX_CHUNK_SIZE = 64 * 1024 * 1024
const CHUNK_TIMEOUT = 60_000
const MAX_CHUNK_RETRIES = 10

/**
 * Upload a file as a resumable upload (IETF draft-ietf-httpbis-resumable-upload): add a file that has no ID yet to
 * the upload (upload creation), append chunks with PATCH and, after a failure, resume from the offset that HEAD returns
 * @returns {{ promise: Promise<Object>, abort: Function }} - abort also cancels the upload on the server
 */
function uploadFileResumable(upload, fileEntry, onProgress, basicAuth, onRetry) {
    const file = fileEntry.file
    const url = () => `${base}/resumable/${upload.id}/${fileEntry.id}/${encodeURIComponent(fileEntry.fileName)}`
    let request = null
    let cancelSleep = null

    const sleep = (ms) => new Promise((resolve, reject) => {
        const timer = setTimeout(resolve, ms)
        cancelSleep = () => {
            clearTimeout(timer)
            reject({ status: 0, message: t('api.uploadCancelled'), cancelled: true })
        }
    })

    const send = (method, target, body = null, headers = {}, onSent = undefined) => {
        request = sendUploadRequest(method, target, upload, basicAuth, body, { headers, timeout: CHUNK_TIMEOUT, onProgress: onSent })
        return request.promise
    }

    const promise = (async () => {
        let chunkSize = MIN_CHUNK_SIZE
        let retries = 0
        for (;;) {
            try {
                if (!fileEntry.id) {
                    // The reference makes the server return the same file if the creation is retried
                    const creation = `${base}/resumable/${upload.id}/${encodeURIComponent(fileEntry.fileName)}?reference=${encodeURIComponent(fileEntry.reference)}`
                    const created = await send('POST', creation, null, { 'Upload-Complete': '?0', 'Upload-Length': String(file.size) })
                    fileEntry.id = created.getResponseHeader('Location').split('/').at(-2)
                }
                const head = await send('HEAD', url())
                let offset = Number(head.getResponseHeader('Upload-Offset'))
                onProgress?.(Math.round((offset / file.size) * 100))
                for (;;) {
                    // The last append completes the upload, even without content when the server already has every byte
                    const end = Math.min(offset + chunkSize, file.size)
                    const complete = end === file.size
                    const started = Date.now()
                    const xhr = await send('PATCH', url(), file.slice(offset, end), {
                        'Content-Type': 'application/partial-upload',
                        'Upload-Offset': String(offset),
                        'Upload-Complete': complete ? '?1' : '?0',
                    }, onProgress && ((loaded) => onProgress(Math.round(((offset + loaded) / file.size) * 100))))
                    if (complete) return JSON.parse(xhr.responseText)
                    const seconds = (Date.now() - started) / 1000
                    chunkSize = Math.min(MAX_CHUNK_SIZE, Math.max(MIN_CHUNK_SIZE, Math.round(((end - offset) / seconds) * CHUNK_SECONDS)))
                    offset = end
                    retries = 0
                }
            } catch (err) {
                // Retry unless the server has processed the file (Upload-Complete: ?1)
                const retryable = err.status === 0 || (err.xhr.getResponseHeader('Upload-Complete') !== '?1'
                    && (err.status === 409 || err.status === 413 || err.status === 429 || err.status >= 500))
                if (err.cancelled || !retryable || retries >= MAX_CHUNK_RETRIES) throw err
                onRetry?.(err)
            }
            chunkSize = MIN_CHUNK_SIZE
            await sleep(Math.min(1000 * 2 ** retries, 30_000))
            retries++
        }
    })()

    const abort = () => {
        request.xhr.abort()
        cancelSleep?.()
        if (fileEntry.id) {
            return sendUploadRequest('DELETE', url(), upload, basicAuth, null).promise
                .catch((err) => console.warn('Failed to cancel upload:', err))
        }
    }

    return { promise, abort }
}

/**
 * Send a file upload request with XMLHttpRequest, which reports upload progress unlike fetch()
 * @param {string} method - HTTP method
 * @param {string} url - request URL
 * @param {Object} upload - The upload object { uploadToken }
 * @param {string} basicAuth - optional base64 auth string
 * @param {Blob|FormData|null} body - request body
 * @param {Object} options - { headers, timeout (ms), onProgress(loaded, total), onStart }
 * @returns {{ promise: Promise<XMLHttpRequest>, xhr: XMLHttpRequest }} - promise resolves to the request once it succeeds
 */
function sendUploadRequest(method, url, upload, basicAuth, body, { headers = {}, timeout = 0, onProgress, onStart } = {}) {
    const xhr = new XMLHttpRequest()
    xhr.open(method, url)
    xhr.timeout = timeout

    if (upload.uploadToken) {
        xhr.setRequestHeader('X-UploadToken', upload.uploadToken)
    }
    if (basicAuth) {
        xhr.setRequestHeader('Authorization', 'Basic ' + basicAuth)
    }
    for (const [name, value] of Object.entries(headers)) {
        xhr.setRequestHeader(name, value)
    }

    // XSRF token for mutating request
    const xsrf = getXsrfToken()
    if (xsrf) xhr.setRequestHeader('X-XSRFToken', xsrf)

    const promise = new Promise((resolve, reject) => {
        xhr.upload.addEventListener('progress', (e) => {
            if (e.lengthComputable && onProgress) {
                onProgress(e.loaded, e.total)
            }
        })

        if (onStart) {
            xhr.upload.addEventListener('loadstart', () => onStart())
        }

        xhr.addEventListener('load', () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                resolve(xhr)
            } else {
                let message = t('api.uploadFailed', { status: xhr.status })
                try {
                    const body = JSON.parse(xhr.responseText)
                    message = body.message || message
                } catch {
                    // Server returns plain text errors (not JSON)
                    if (xhr.responseText) message = xhr.responseText
                }
                reject({ status: xhr.status, message, xhr })
            }
        })

        const connectionLost = () => reject({ status: 0, message: t('api.uploadConnectionLost'), xhr })
        xhr.addEventListener('error', connectionLost)
        xhr.addEventListener('timeout', connectionLost)

        xhr.addEventListener('abort', () => {
            reject({ status: 0, message: t('api.uploadCancelled'), cancelled: true })
        })
    })

    xhr.send(body)

    return { promise, xhr }
}

export function removeFile(upload, file) {
    const mode = upload.stream ? 'stream' : 'file'
    const url = `${base}/${mode}/${upload.id}/${file.id}/${encodeURIComponent(file.fileName)}`
    const headers = {}
    if (upload.uploadToken) headers['X-UploadToken'] = upload.uploadToken
    return apiCall(url, 'DELETE', null, headers)
}

// ── URL Builders ──

let _downloadURL = ''

export function setDownloadURL(url) {
    _downloadURL = url || ''
}

function downloadBase() {
    return _downloadURL || base
}

export function getFileURL(uploadId, fileId, fileName, stream = false) {
    const mode = stream ? 'stream' : 'file'
    return `${downloadBase()}/${mode}/${uploadId}/${fileId}/${encodeURIComponent(fileName)}`
}

export function getArchiveURL(uploadId, fileName = 'archive.zip') {
    return `${downloadBase()}/archive/${uploadId}/${encodeURIComponent(fileName)}`
}

export function getAdminURL(uploadId, uploadToken) {
    const url = `${window.location.origin}${window.location.pathname}#/?id=${uploadId}`
    return uploadToken ? `${url}&uploadToken=${uploadToken}` : url
}

export function getQrCodeURL(url, size = 200) {
    return `${base}/qrcode?url=${encodeURIComponent(url)}&size=${size}`
}
