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
 * @param {Object} upload - The upload object { id, stream, uploadToken }
 * @param {Object} fileEntry - { id, fileName, file (File object) }
 * @param {Function} onProgress - callback(percent)
 * @param {string} basicAuth - optional base64 auth string
 * @param {Function} onStart - optional callback fired when upload connection opens (loadstart)
 * @returns {{ promise: Promise<Object>, abort: Function }} - promise resolves to file metadata, abort cancels the upload
 */
export function uploadFile(upload, fileEntry, onProgress, basicAuth, onStart) {
    const mode = upload.stream ? 'stream' : 'file'
    let url
    if (fileEntry.id) {
        url = `${base}/${mode}/${upload.id}/${fileEntry.id}/${encodeURIComponent(fileEntry.fileName)}`
    } else {
        // Adding file to existing upload
        url = `${base}/${mode}/${upload.id}`
    }

    const xhr = new XMLHttpRequest()
    xhr.open('POST', url)

    if (upload.uploadToken) {
        xhr.setRequestHeader('X-UploadToken', upload.uploadToken)
    }
    if (basicAuth) {
        xhr.setRequestHeader('Authorization', 'Basic ' + basicAuth)
    }

    // XSRF token for mutating request
    const xsrf = getXsrfToken()
    if (xsrf) xhr.setRequestHeader('X-XSRFToken', xsrf)

    const promise = new Promise((resolve, reject) => {
        xhr.upload.addEventListener('progress', (e) => {
            if (e.lengthComputable && onProgress) {
                onProgress(Math.round((e.loaded / e.total) * 100))
            }
        })

        if (onStart) {
            xhr.upload.addEventListener('loadstart', () => onStart())
        }

        xhr.addEventListener('load', () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                try {
                    resolve(JSON.parse(xhr.responseText))
                } catch {
                    resolve(null)
                }
            } else {
                let message = t('api.uploadFailed', { status: xhr.status })
                try {
                    const body = JSON.parse(xhr.responseText)
                    message = body.message || message
                } catch {
                    // Server returns plain text errors (not JSON)
                    if (xhr.responseText) message = xhr.responseText
                }
                reject({ status: xhr.status, message })
            }
        })

        xhr.addEventListener('error', () => {
            reject({ status: 0, message: t('api.uploadConnectionLost') })
        })

        xhr.addEventListener('abort', () => {
            reject({ status: 0, message: t('api.uploadCancelled'), cancelled: true })
        })
    })

    // Send the file as form data
    const formData = new FormData()
    formData.append('file', fileEntry.file, fileEntry.fileName)
    xhr.send(formData)

    return { promise, abort: () => xhr.abort() }
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
