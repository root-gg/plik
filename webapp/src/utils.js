// Utility functions

/**
 * Format bytes into human-readable size string
 */
export function humanReadableSize(bytes) {
    if (bytes === 0) return '0 B'
    if (!bytes) return ''

    const units = ['B', 'kB', 'MB', 'GB', 'TB']
    const k = 1000
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    const size = (bytes / Math.pow(k, i)).toFixed(i > 0 ? 2 : 0)

    return `${size} ${units[i]}`
}

/**
 * Format a TTL (seconds) into a human-readable duration
 */
export function humanDuration(seconds) {
    if (!seconds || seconds <= 0) return 'unlimited'

    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)

    const parts = []
    if (days > 0) parts.push(`${days} day${days > 1 ? 's' : ''}`)
    if (hours > 0) parts.push(`${hours} hour${hours > 1 ? 's' : ''}`)
    if (minutes > 0) parts.push(`${minutes} minute${minutes > 1 ? 's' : ''}`)

    return parts.join(' ') || '< 1 minute'
}

/**
 * Format a date for display
 */
export function formatDate(dateStr) {
    if (!dateStr) return ''
    const d = new Date(dateStr)
    return d.toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    })
}

/**
 * Convert TTL value + unit to seconds
 */
export function ttlToSeconds(value, unit) {
    const multipliers = {
        minutes: 60,
        hours: 3600,
        days: 86400,
    }
    return value * (multipliers[unit] || 86400)
}

/**
 * Convert TTL in seconds to best value + unit pair
 */
export function secondsToTTL(seconds) {
    if (seconds <= 0) return { value: 0, unit: 'days' }

    if (seconds % 86400 === 0) return { value: seconds / 86400, unit: 'days' }
    if (seconds % 3600 === 0) return { value: seconds / 3600, unit: 'hours' }
    return { value: Math.round(seconds / 60), unit: 'minutes' }
}

/**
 * Generate a unique reference ID for local file tracking
 */
let refCounter = 0
export function generateRef() {
    return `ref-${Date.now()}-${++refCounter}`
}

/**
 * Encode basic auth header value
 */
export function encodeBasicAuth(login, password) {
    return btoa(`${login}:${password}`)
}

// ── Quota & unit conversion helpers ──
// Used by HomeView and AdminView for user/admin edit forms

const GB = 1000 * 1000 * 1000

export const TTL_UNITS = [
    { label: 'minutes', i18nKey: 'uploadSidebar.minutes', seconds: 60 },
    { label: 'hours', i18nKey: 'uploadSidebar.hours', seconds: 3600 },
    { label: 'days', i18nKey: 'uploadSidebar.days', seconds: 86400 },
]

/**
 * Build the hash-based URL for an upload
 */
export function getUploadUrl(upload) {
    return `${window.location.origin}${window.location.pathname}#/?id=${upload.id}`
}

/**
 * Display label for a quota value (bytes)
 */
export function quotaLabel(value, t) {
    if (!value || value === 0) return t ? t('common.default') : 'default'
    if (value === -1) return t ? t('common.unlimited') : 'unlimited'
    return humanReadableSize(value)
}

/**
 * Display label for a TTL value (seconds)
 */
export function ttlLabel(seconds, t) {
    if (!seconds || seconds === 0) return t ? t('common.default') : 'default'
    if (seconds === -1) return t ? t('common.unlimited') : 'unlimited'
    if (seconds < 60) return seconds + 's'
    if (seconds < 3600) return Math.floor(seconds / 60) + 'm'
    if (seconds < 86400) return Math.floor(seconds / 3600) + 'h'
    return Math.floor(seconds / 86400) + 'd'
}

/**
 * Convert bytes to GB for display in form inputs.
 * Preserves 0 (default) and -1 (unlimited).
 */
export function bytesToGB(bytes) {
    if (bytes <= 0) return bytes
    return parseFloat((bytes / GB).toFixed(4))
}

/**
 * Convert GB from form input back to bytes.
 * Preserves 0 (default) and -1 (unlimited).
 */
export function gbToBytes(gb) {
    if (gb <= 0) return gb
    return Math.round(gb * GB)
}

/**
 * Convert seconds to { value, unit } using the best-fitting unit (seconds).
 * Preserves 0 and -1 as-is.
 */
export function secondsToBestUnit(seconds) {
    if (seconds <= 0) return { value: seconds, unit: 60 }
    if (seconds % 86400 === 0) return { value: seconds / 86400, unit: 86400 }
    if (seconds % 3600 === 0) return { value: seconds / 3600, unit: 3600 }
    return { value: seconds / 60, unit: 60 }
}

/**
 * Convert { value, unit (seconds) } back to total seconds.
 * Preserves 0 and -1 as-is.
 */
export function unitToSeconds(value, unit) {
    if (value <= 0) return value
    return Math.round(value * unit)
}

/**
 * Clamp a quota input value: empty/NaN → 0, < -1 → -1, between -1 and 0 → 0
 */
export function clampQuota(val) {
    if (val === '' || val === null || val === undefined) return 0
    const n = Number(val)
    if (isNaN(n)) return 0
    if (n < -1) return -1
    if (n > -1 && n < 0) return 0
    return n
}

/**
 * Filter raw text input for quota fields.
 * Allows digits, an optional leading '-', and an optional '.' when allowDecimal is true.
 * Returns the sanitized string (caller converts to Number on blur via clampQuota).
 */
export function filterQuotaInput(raw, allowDecimal = false) {
    let out = ''
    let hasDot = false
    for (let i = 0; i < raw.length; i++) {
        const ch = raw[i]
        if (ch === '-' && i === 0 && out === '') {
            out += ch
        } else if (ch === '.' && allowDecimal && !hasDot) {
            hasDot = true
            out += ch
        } else if (ch >= '0' && ch <= '9') {
            out += ch
        }
    }
    return out
}

/**
 * Hint text for size quota inputs showing the server default
 */
export function defaultSizeHint(configVal, t) {
    const hint0 = t ? t('common.defaultHint') : '0 = default, -1 = unlimited'
    if (!configVal || configVal <= 0 || isNaN(configVal)) return hint0
    const hint = t ? t('common.defaultHintValue', { value: humanReadableSize(configVal) }) : `0 = default (${humanReadableSize(configVal)}), -1 = unlimited`
    return hint
}

/**
 * Hint text for TTL quota inputs showing the server default
 */
export function defaultTTLHint(configVal, t) {
    const hint0 = t ? t('common.defaultHint') : '0 = default, -1 = unlimited'
    if (!configVal || configVal <= 0 || isNaN(configVal)) return hint0
    const ttl = secondsToBestUnit(configVal)
    const unit = TTL_UNITS.find(u => u.seconds === ttl.unit)
    const unitLabel = (t && unit) ? t(unit.i18nKey) : (unit ? unit.label : 's')
    const hint = t ? t('common.defaultHintValue', { value: `${ttl.value} ${unitLabel}` }) : `0 = default (${ttl.value} ${unitLabel}), -1 = unlimited`
    return hint
}

/**
 * Build a form object from a user record for editing.
 * Converts bytes → GB and seconds → best unit for display.
 * Returns { form, ttlUnit }.
 */
export function buildEditForm(user) {
    const ttl = secondsToBestUnit(user.maxTTL || 0)
    return {
        form: {
            id: user.id,
            provider: user.provider,
            login: user.login,
            name: user.name || '',
            email: user.email || '',
            password: '',
            admin: user.admin || false,
            maxFileSize: bytesToGB(user.maxFileSize || 0),
            maxUserSize: bytesToGB(user.maxUserSize || 0),
            maxTTL: ttl.value,
        },
        ttlUnit: ttl.unit,
    }
}

/**
 * Convert an edit form back into an API-ready payload.
 * Converts GB → bytes and unit value → seconds.
 * Strips empty password field.
 */
export function buildEditPayload(form, ttlUnit) {
    const payload = { ...form }
    if (!payload.password) delete payload.password
    payload.maxFileSize = gbToBytes(payload.maxFileSize)
    payload.maxUserSize = gbToBytes(payload.maxUserSize)
    payload.maxTTL = unitToSeconds(payload.maxTTL, ttlUnit)
    return payload
}

// ── Text-file detection ──
// Used by FileRow to determine if a file can be viewed in the code editor

/** Max file size viewable in the code editor (5 MB) */
export const MAX_VIEWABLE_SIZE = 5 * 1024 * 1024

/**
 * Determine if a file object is a viewable text file.
 * The server detects MIME types via Go's http.DetectContentType,
 * which returns text/plain for all text-like content.
 */
export function isTextFile(file) {
    const size = file.fileSize || file.size || 0
    if (size > MAX_VIEWABLE_SIZE) return false

    const mime = (file.fileType || '').toLowerCase()
    return mime.startsWith('text/')
}

/**
 * Determine if a file is a Markdown file (viewable with rendered preview).
 * Checks both the filename extension and that the MIME type is text/plain
 * (Go's http.DetectContentType returns text/plain for .md files).
 */
export function isMarkdownFile(file) {
    const name = (file.fileName || '').toLowerCase()
    const mime = (file.fileType || '').toLowerCase()
    if (!mime.startsWith('text/')) return false
    return name.endsWith('.md') || name.endsWith('.markdown')
}

/**
 * Determine if a file is an image (viewable as an inline preview).
 * Checks that the MIME type starts with image/.
 * No size limit — browsers handle large images natively.
 */
export function isImageFile(file) {
    const mime = (file.fileType || '').toLowerCase()
    return mime.startsWith('image/')
}

/**
 * Determine if a file is a video (playable inline via native <video>).
 * Checks that the MIME type starts with video/.
 * No size limit — browsers handle streaming playback natively.
 */
export function isVideoFile(file) {
    const mime = (file.fileType || '').toLowerCase()
    return mime.startsWith('video/')
}

/**
 * Determine if a file is audio (playable inline via native <audio>).
 * Checks that the MIME type starts with audio/.
 */
export function isAudioFile(file) {
    const mime = (file.fileType || '').toLowerCase()
    return mime.startsWith('audio/')
}

/**
 * Determine if a file can be previewed inline (text, image, video, or audio).
 * Combines detection helpers for a single viewability check.
 */
export function isViewableFile(file) {
    return isTextFile(file) || isImageFile(file) || isVideoFile(file) || isAudioFile(file)
}
