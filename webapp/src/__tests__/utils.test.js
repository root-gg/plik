import { describe, it, expect } from 'vitest'
import {
    humanReadableSize,
    humanDuration,
    formatDate,
    formatCount,
    ttlToSeconds,
    secondsToTTL,
    generateRef,
    clampQuota,
    filterQuotaInput,
    bytesToGB,
    gbToBytes,
    secondsToBestUnit,
    unitToSeconds,
    quotaLabel,
    ttlLabel,
    defaultSizeHint,
    defaultTTLHint,
    buildEditForm,
    buildEditPayload,
    isTextFile,
    isMarkdownFile,
    isImageFile,
    isVideoFile,
    isAudioFile,
    isViewableFile,
    charsetFromContentType,
    MAX_VIEWABLE_SIZE,
    getUploadUrl,
} from '../utils.js'

// ── humanReadableSize ──

describe('humanReadableSize', () => {
    it('returns "0 B" for zero', () => {
        expect(humanReadableSize(0)).toBe('0 B')
    })

    it('returns empty string for null/undefined', () => {
        expect(humanReadableSize(null)).toBe('')
        expect(humanReadableSize(undefined)).toBe('')
    })

    it('formats bytes', () => {
        expect(humanReadableSize(500)).toBe('500 B')
    })

    it('formats kilobytes with uppercase KB (matches the distribution bucket labels\' casing)', () => {
        expect(humanReadableSize(1500)).toBe('1.50 KB')
    })

    it('formats megabytes', () => {
        expect(humanReadableSize(2_500_000)).toBe('2.50 MB')
    })

    it('formats gigabytes', () => {
        expect(humanReadableSize(1_000_000_000)).toBe('1.00 GB')
    })

    it('formats terabytes', () => {
        expect(humanReadableSize(3_000_000_000_000)).toBe('3.00 TB')
    })
})

// ── humanDuration ──

describe('humanDuration', () => {
    it('returns "unlimited" for 0', () => {
        expect(humanDuration(0)).toBe('unlimited')
    })

    it('returns "unlimited" for negative', () => {
        expect(humanDuration(-1)).toBe('unlimited')
    })

    it('returns "< 1 minute" for small values', () => {
        expect(humanDuration(30)).toBe('< 1 minute')
    })

    it('formats minutes', () => {
        expect(humanDuration(120)).toBe('2 minutes')
    })

    it('formats hours', () => {
        expect(humanDuration(7200)).toBe('2 hours')
    })

    it('formats days', () => {
        expect(humanDuration(172800)).toBe('2 days')
    })

    it('formats mixed durations', () => {
        expect(humanDuration(90061)).toBe('1 day 1 hour 1 minute')
    })

    it('returns null/undefined as unlimited', () => {
        expect(humanDuration(null)).toBe('unlimited')
        expect(humanDuration(undefined)).toBe('unlimited')
    })
})

// ── formatDate ──

describe('formatDate', () => {
    it('returns empty string for falsy input', () => {
        expect(formatDate(null)).toBe('')
        expect(formatDate('')).toBe('')
        expect(formatDate(undefined)).toBe('')
    })

    it('returns a formatted string for valid ISO date', () => {
        const result = formatDate('2024-06-15T12:30:00Z')
        // Just check it contains expected parts
        expect(result).toContain('2024')
        expect(result).toContain('Jun')
    })

    it('defaults to the active vue-i18n locale (i18n.js starts at "en")', () => {
        // No locale arg -> getLocale(); the app's i18n singleton starts at
        // 'en' and nothing in this test file switches it.
        expect(formatDate('2024-06-15T12:30:00Z')).toBe(formatDate('2024-06-15T12:30:00Z', 'en'))
    })

    it('threads an explicit locale override (French month names)', () => {
        const iso = '2024-06-15T12:30:00Z'
        const result = formatDate(iso, 'fr')
        const reference = new Date(iso).toLocaleDateString('fr', {
            year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
        })
        expect(result).toBe(reference)
        expect(result).not.toBe(formatDate(iso, 'en'))
    })
})

// ── formatCount ──

describe('formatCount', () => {
    it('returns "0" for non-finite input', () => {
        expect(formatCount(null)).toBe('0')
        expect(formatCount(undefined)).toBe('0')
        expect(formatCount(NaN)).toBe('0')
    })

    it('groups thousands using the default (active i18n) locale', () => {
        expect(formatCount(1234567)).toBe((1234567).toLocaleString('en'))
    })

    it('threads an explicit locale override (French grouping)', () => {
        const result = formatCount(3946, 'fr')
        const reference = (3946).toLocaleString('fr')
        // Assert via the reference, not a hardcoded space character — French
        // grouping may use a narrow no-break space (U+202F) depending on the
        // ICU data, not a plain space.
        expect(result).toBe(reference)
        expect(result).not.toBe(formatCount(3946, 'en'))
    })
})

// ── ttlToSeconds / secondsToTTL ──

describe('ttlToSeconds', () => {
    it('converts minutes', () => {
        expect(ttlToSeconds(5, 'minutes')).toBe(300)
    })

    it('converts hours', () => {
        expect(ttlToSeconds(2, 'hours')).toBe(7200)
    })

    it('converts days', () => {
        expect(ttlToSeconds(3, 'days')).toBe(259200)
    })

    it('defaults to days for unknown unit', () => {
        expect(ttlToSeconds(1, 'unknown')).toBe(86400)
    })
})

describe('secondsToTTL', () => {
    it('returns days for exact day multiples', () => {
        expect(secondsToTTL(172800)).toEqual({ value: 2, unit: 'days' })
    })

    it('returns hours for exact hour multiples', () => {
        expect(secondsToTTL(7200)).toEqual({ value: 2, unit: 'hours' })
    })

    it('returns minutes for anything else', () => {
        expect(secondsToTTL(150)).toEqual({ value: 3, unit: 'minutes' })
    })

    it('returns zero days for zero or negative', () => {
        expect(secondsToTTL(0)).toEqual({ value: 0, unit: 'days' })
        expect(secondsToTTL(-1)).toEqual({ value: 0, unit: 'days' })
    })
})

// ── generateRef ──

describe('generateRef', () => {
    it('generates unique refs', () => {
        const a = generateRef()
        const b = generateRef()
        expect(a).not.toBe(b)
    })

    it('starts with "ref-"', () => {
        expect(generateRef()).toMatch(/^ref-/)
    })
})

// ── clampQuota ──

describe('clampQuota', () => {
    it('returns 0 for empty/null/undefined', () => {
        expect(clampQuota('')).toBe(0)
        expect(clampQuota(null)).toBe(0)
        expect(clampQuota(undefined)).toBe(0)
    })

    it('returns 0 for NaN', () => {
        expect(clampQuota('abc')).toBe(0)
    })

    it('clamps below -1 to -1', () => {
        expect(clampQuota(-5)).toBe(-1)
    })

    it('clamps between -1 and 0 to 0', () => {
        expect(clampQuota(-0.5)).toBe(0)
    })

    it('preserves -1 (unlimited)', () => {
        expect(clampQuota(-1)).toBe(-1)
    })

    it('preserves 0 (default)', () => {
        expect(clampQuota(0)).toBe(0)
    })

    it('preserves positive values', () => {
        expect(clampQuota(10)).toBe(10)
        expect(clampQuota(0.5)).toBe(0.5)
    })
})

// ── filterQuotaInput ──

describe('filterQuotaInput', () => {
    it('allows digits', () => {
        expect(filterQuotaInput('123')).toBe('123')
    })

    it('allows leading minus', () => {
        expect(filterQuotaInput('-1')).toBe('-1')
    })

    it('strips minus not at start', () => {
        expect(filterQuotaInput('1-2')).toBe('12')
    })

    it('strips letters and symbols', () => {
        expect(filterQuotaInput('a1b2c')).toBe('12')
        expect(filterQuotaInput('!@#')).toBe('')
    })

    it('allows decimal point when allowDecimal is true', () => {
        expect(filterQuotaInput('1.5', true)).toBe('1.5')
    })

    it('allows only one decimal point', () => {
        expect(filterQuotaInput('1.2.3', true)).toBe('1.23')
    })

    it('strips decimal point when allowDecimal is false', () => {
        expect(filterQuotaInput('1.5', false)).toBe('15')
    })

    it('handles negative decimal', () => {
        expect(filterQuotaInput('-0.5', true)).toBe('-0.5')
    })

    it('returns empty string for empty input', () => {
        expect(filterQuotaInput('')).toBe('')
    })

    it('handles just minus sign', () => {
        expect(filterQuotaInput('-')).toBe('-')
    })
})

// ── bytesToGB / gbToBytes ──

describe('bytesToGB / gbToBytes roundtrip', () => {
    it('preserves 0', () => {
        expect(bytesToGB(0)).toBe(0)
        expect(gbToBytes(0)).toBe(0)
    })

    it('preserves -1', () => {
        expect(bytesToGB(-1)).toBe(-1)
        expect(gbToBytes(-1)).toBe(-1)
    })

    it('round-trips positive values', () => {
        const bytes = 2_500_000_000 // 2.5 GB
        const gb = bytesToGB(bytes)
        expect(gb).toBe(2.5)
        expect(gbToBytes(gb)).toBe(bytes)
    })

    it('round-trips 1 GB', () => {
        const bytes = 1_000_000_000
        expect(gbToBytes(bytesToGB(bytes))).toBe(bytes)
    })
})

// ── secondsToBestUnit / unitToSeconds ──

describe('secondsToBestUnit / unitToSeconds', () => {
    it('picks days for exact day multiples', () => {
        expect(secondsToBestUnit(172800)).toEqual({ value: 2, unit: 86400 })
    })

    it('picks hours for exact hour multiples', () => {
        expect(secondsToBestUnit(7200)).toEqual({ value: 2, unit: 3600 })
    })

    it('falls back to minutes', () => {
        expect(secondsToBestUnit(150)).toEqual({ value: 2.5, unit: 60 })
    })

    it('preserves 0 with minutes unit', () => {
        expect(secondsToBestUnit(0)).toEqual({ value: 0, unit: 60 })
    })

    it('preserves -1 with minutes unit', () => {
        expect(secondsToBestUnit(-1)).toEqual({ value: -1, unit: 60 })
    })

    it('round-trips via unitToSeconds', () => {
        const seconds = 172800
        const { value, unit } = secondsToBestUnit(seconds)
        expect(unitToSeconds(value, unit)).toBe(seconds)
    })

    it('unitToSeconds preserves 0', () => {
        expect(unitToSeconds(0, 3600)).toBe(0)
    })

    it('unitToSeconds preserves -1', () => {
        expect(unitToSeconds(-1, 86400)).toBe(-1)
    })
})

// ── quotaLabel / ttlLabel ──

describe('quotaLabel', () => {
    it('returns "default" for 0', () => {
        expect(quotaLabel(0)).toBe('default')
    })

    it('returns "default" for null/undefined', () => {
        expect(quotaLabel(null)).toBe('default')
        expect(quotaLabel(undefined)).toBe('default')
    })

    it('returns "unlimited" for -1', () => {
        expect(quotaLabel(-1)).toBe('unlimited')
    })

    it('returns human-readable size for positive', () => {
        expect(quotaLabel(1_000_000_000)).toBe('1.00 GB')
    })
})

describe('ttlLabel', () => {
    it('returns "default" for 0', () => {
        expect(ttlLabel(0)).toBe('default')
    })

    it('returns "unlimited" for -1', () => {
        expect(ttlLabel(-1)).toBe('unlimited')
    })

    it('returns seconds for < 60', () => {
        expect(ttlLabel(30)).toBe('30s')
    })

    it('returns minutes for < 3600', () => {
        expect(ttlLabel(300)).toBe('5m')
    })

    it('returns hours for < 86400', () => {
        expect(ttlLabel(7200)).toBe('2h')
    })

    it('returns days for >= 86400', () => {
        expect(ttlLabel(172800)).toBe('2d')
    })
})

// ── defaultSizeHint / defaultTTLHint ──

describe('defaultSizeHint', () => {
    it('returns generic hint when no config value', () => {
        expect(defaultSizeHint(0)).toBe('0 = default, -1 = unlimited')
    })

    it('includes server default when set', () => {
        const hint = defaultSizeHint(10_000_000_000)
        expect(hint).toContain('10.00 GB')
        expect(hint).toContain('-1 = unlimited')
    })
})

describe('defaultTTLHint', () => {
    it('returns generic hint when no config value', () => {
        expect(defaultTTLHint(0)).toBe('0 = default, -1 = unlimited')
    })

    it('includes server default when set', () => {
        const hint = defaultTTLHint(86400)
        expect(hint).toContain('1')
        expect(hint).toContain('days')
    })
})

// ── buildEditForm / buildEditPayload ──

describe('buildEditForm / buildEditPayload roundtrip', () => {
    const user = {
        id: 'u1',
        provider: 'local',
        login: 'testuser',
        name: 'Test User',
        email: 'test@example.com',
        admin: true,
        maxFileSize: 5_000_000_000,    // 5 GB
        maxUserSize: 10_000_000_000,   // 10 GB
        maxTTL: 172800,                 // 2 days
    }

    it('builds form from user', () => {
        const { form, ttlUnit } = buildEditForm(user)
        expect(form.id).toBe('u1')
        expect(form.provider).toBe('local')
        expect(form.maxFileSize).toBe(5)
        expect(form.maxUserSize).toBe(10)
        expect(form.maxTTL).toBe(2)
        expect(ttlUnit).toBe(86400) // days
        expect(form.password).toBe('')
    })

    it('round-trips back to original values', () => {
        const { form, ttlUnit } = buildEditForm(user)
        const payload = buildEditPayload(form, ttlUnit)
        expect(payload.maxFileSize).toBe(5_000_000_000)
        expect(payload.maxUserSize).toBe(10_000_000_000)
        expect(payload.maxTTL).toBe(172800)
    })

    it('strips empty password from payload', () => {
        const { form, ttlUnit } = buildEditForm(user)
        const payload = buildEditPayload(form, ttlUnit)
        expect(payload.password).toBeUndefined()
    })

    it('preserves password when set', () => {
        const { form, ttlUnit } = buildEditForm(user)
        form.password = 'newpass'
        const payload = buildEditPayload(form, ttlUnit)
        expect(payload.password).toBe('newpass')
    })

    it('handles zero quotas (default)', () => {
        const { form, ttlUnit } = buildEditForm({ ...user, maxFileSize: 0, maxUserSize: 0, maxTTL: 0 })
        expect(form.maxFileSize).toBe(0)
        expect(form.maxUserSize).toBe(0)
        expect(form.maxTTL).toBe(0)
        const payload = buildEditPayload(form, ttlUnit)
        expect(payload.maxFileSize).toBe(0)
        expect(payload.maxUserSize).toBe(0)
        expect(payload.maxTTL).toBe(0)
    })
})

// ── isTextFile ──

describe('isTextFile', () => {
    it('returns true for small text file', () => {
        expect(isTextFile({ fileSize: 1000, fileType: 'text/plain' })).toBe(true)
    })

    it('returns false for file over size limit', () => {
        expect(isTextFile({ fileSize: MAX_VIEWABLE_SIZE + 1, fileType: 'text/plain' })).toBe(false)
    })

    it('returns false for binary mime type', () => {
        expect(isTextFile({ fileSize: 1000, fileType: 'application/octet-stream' })).toBe(false)
    })

    it('returns true at exactly the size limit', () => {
        expect(isTextFile({ fileSize: MAX_VIEWABLE_SIZE, fileType: 'text/plain' })).toBe(true)
    })

    it('returns true for text/html', () => {
        expect(isTextFile({ fileSize: 100, fileType: 'text/html' })).toBe(true)
    })

    it('handles missing fileType', () => {
        expect(isTextFile({ fileSize: 100 })).toBe(false)
    })

    it('uses size field as fallback', () => {
        expect(isTextFile({ size: 100, fileType: 'text/plain' })).toBe(true)
    })

    // Server-side isText field (migration 0009)

    it('returns true for application/json when isText is true', () => {
        expect(isTextFile({ fileSize: 1000, fileType: 'application/json', isText: true })).toBe(true)
    })

    it('returns true for application/x-perl when isText is true', () => {
        expect(isTextFile({ fileSize: 1000, fileType: 'application/x-perl', isText: true })).toBe(true)
    })

    it('returns false for application/octet-stream when isText is false', () => {
        expect(isTextFile({ fileSize: 1000, fileType: 'application/octet-stream', isText: false })).toBe(false)
    })

    it('returns false when isText is true but file exceeds size limit', () => {
        expect(isTextFile({ fileSize: MAX_VIEWABLE_SIZE + 1, fileType: 'application/json', isText: true })).toBe(false)
    })

    it('falls back to text/* prefix when isText is absent (pre-migration)', () => {
        expect(isTextFile({ fileSize: 1000, fileType: 'text/plain' })).toBe(true)
    })

    it('returns false for application/json when isText is absent (pre-migration)', () => {
        expect(isTextFile({ fileSize: 1000, fileType: 'application/json' })).toBe(false)
    })
})

// ── isMarkdownFile ──

describe('isMarkdownFile', () => {
    it('returns true for .md file with text/plain', () => {
        expect(isMarkdownFile({ fileName: 'readme.md', fileType: 'text/plain' })).toBe(true)
    })

    it('returns true for .markdown file with text/plain', () => {
        expect(isMarkdownFile({ fileName: 'notes.markdown', fileType: 'text/plain' })).toBe(true)
    })

    it('is case-insensitive on extension', () => {
        expect(isMarkdownFile({ fileName: 'README.MD', fileType: 'text/plain' })).toBe(true)
        expect(isMarkdownFile({ fileName: 'doc.Markdown', fileType: 'text/plain' })).toBe(true)
    })

    it('returns true for text/plain with charset', () => {
        expect(isMarkdownFile({ fileName: 'readme.md', fileType: 'text/plain; charset=utf-8' })).toBe(true)
    })

    it('returns false for non-text MIME type', () => {
        expect(isMarkdownFile({ fileName: 'readme.md', fileType: 'application/octet-stream' })).toBe(false)
    })

    it('returns false for non-markdown text file', () => {
        expect(isMarkdownFile({ fileName: 'plain.txt', fileType: 'text/plain' })).toBe(false)
    })

    it('returns false when fileName is missing', () => {
        expect(isMarkdownFile({ fileType: 'text/plain' })).toBe(false)
    })

    it('returns false when fileType is missing', () => {
        expect(isMarkdownFile({ fileName: 'readme.md' })).toBe(false)
    })
})

// ── isImageFile ──

describe('isImageFile', () => {
    it('returns true for image/png', () => {
        expect(isImageFile({ fileType: 'image/png' })).toBe(true)
    })

    it('returns true for image/jpeg', () => {
        expect(isImageFile({ fileType: 'image/jpeg' })).toBe(true)
    })

    it('returns true for image/gif', () => {
        expect(isImageFile({ fileType: 'image/gif' })).toBe(true)
    })

    it('returns false for image/svg+xml (content-type neutralized for security)', () => {
        expect(isImageFile({ fileType: 'image/svg+xml' })).toBe(false)
    })

    it('returns true for image/webp', () => {
        expect(isImageFile({ fileType: 'image/webp' })).toBe(true)
    })

    it('returns false for text/plain', () => {
        expect(isImageFile({ fileType: 'text/plain' })).toBe(false)
    })

    it('returns false for application/octet-stream', () => {
        expect(isImageFile({ fileType: 'application/octet-stream' })).toBe(false)
    })

    it('returns false when fileType is missing', () => {
        expect(isImageFile({})).toBe(false)
    })
})

// ── isVideoFile ──

describe('isVideoFile', () => {
    it('returns true for video/mp4', () => {
        expect(isVideoFile({ fileType: 'video/mp4' })).toBe(true)
    })

    it('returns true for video/webm', () => {
        expect(isVideoFile({ fileType: 'video/webm' })).toBe(true)
    })

    it('returns true for video/ogg', () => {
        expect(isVideoFile({ fileType: 'video/ogg' })).toBe(true)
    })

    it('returns false for text/plain', () => {
        expect(isVideoFile({ fileType: 'text/plain' })).toBe(false)
    })

    it('returns false for application/octet-stream', () => {
        expect(isVideoFile({ fileType: 'application/octet-stream' })).toBe(false)
    })

    it('returns false when fileType is missing', () => {
        expect(isVideoFile({})).toBe(false)
    })
})

// ── isAudioFile ──

describe('isAudioFile', () => {
    it('returns true for audio/mpeg', () => {
        expect(isAudioFile({ fileType: 'audio/mpeg' })).toBe(true)
    })

    it('returns true for audio/ogg', () => {
        expect(isAudioFile({ fileType: 'audio/ogg' })).toBe(true)
    })

    it('returns true for audio/wav', () => {
        expect(isAudioFile({ fileType: 'audio/wav' })).toBe(true)
    })

    it('returns false for text/plain', () => {
        expect(isAudioFile({ fileType: 'text/plain' })).toBe(false)
    })

    it('returns false for application/octet-stream', () => {
        expect(isAudioFile({ fileType: 'application/octet-stream' })).toBe(false)
    })

    it('returns false when fileType is missing', () => {
        expect(isAudioFile({})).toBe(false)
    })
})

// ── isViewableFile ──

describe('isViewableFile', () => {
    it('returns true for text files', () => {
        expect(isViewableFile({ fileType: 'text/plain', fileSize: 100 })).toBe(true)
    })

    it('returns true for image files', () => {
        expect(isViewableFile({ fileType: 'image/png' })).toBe(true)
    })

    it('returns false for binary files', () => {
        expect(isViewableFile({ fileType: 'application/octet-stream', fileSize: 100 })).toBe(false)
    })

    it('returns false for text files exceeding size limit', () => {
        expect(isViewableFile({ fileType: 'text/plain', fileSize: 10 * 1024 * 1024 })).toBe(false)
    })

    it('returns true for large images (no size limit)', () => {
        expect(isViewableFile({ fileType: 'image/png', fileSize: 50 * 1024 * 1024 })).toBe(true)
    })

    it('returns true for video files', () => {
        expect(isViewableFile({ fileType: 'video/mp4' })).toBe(true)
    })

    it('returns true for audio files', () => {
        expect(isViewableFile({ fileType: 'audio/mpeg' })).toBe(true)
    })

    it('returns false for SVG files (content-type neutralized for security)', () => {
        expect(isViewableFile({ fileType: 'image/svg+xml' })).toBe(false)
    })
})

// ── getUploadUrl ──

describe('getUploadUrl', () => {
    it('builds hash-based URL', () => {
        const url = getUploadUrl({ id: 'abc123' })
        expect(url).toContain('#/?id=abc123')
    })
})

// ── charsetFromContentType ──

describe('charsetFromContentType', () => {
    it('returns utf-8 when no charset is present', () => {
        expect(charsetFromContentType('text/plain')).toBe('utf-8')
    })

    it('returns utf-8 for explicit utf-8 charset', () => {
        expect(charsetFromContentType('text/plain; charset=utf-8')).toBe('utf-8')
    })

    it('returns utf-16be', () => {
        expect(charsetFromContentType('text/plain; charset=utf-16be')).toBe('utf-16be')
    })

    it('returns utf-16le', () => {
        expect(charsetFromContentType('text/plain; charset=utf-16le')).toBe('utf-16le')
    })

    it('returns iso-8859-1', () => {
        expect(charsetFromContentType('text/plain; charset=iso-8859-1')).toBe('iso-8859-1')
    })

    it('returns windows-1252', () => {
        expect(charsetFromContentType('text/plain; charset=windows-1252')).toBe('windows-1252')
    })

    it('is case-insensitive on the Charset key', () => {
        expect(charsetFromContentType('text/plain; Charset=UTF-16BE')).toBe('UTF-16BE')
    })

    it('handles extra params before charset', () => {
        expect(charsetFromContentType('text/html; boundary=x; charset=latin1')).toBe('latin1')
    })

    it('returns utf-8 for empty string', () => {
        expect(charsetFromContentType('')).toBe('utf-8')
    })

    it('returns utf-8 for null', () => {
        expect(charsetFromContentType(null)).toBe('utf-8')
    })

    it('returns utf-8 for undefined', () => {
        expect(charsetFromContentType(undefined)).toBe('utf-8')
    })
})
