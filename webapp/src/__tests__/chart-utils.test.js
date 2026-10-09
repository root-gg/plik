import { describe, it, expect } from 'vitest'
import {
    niceMax,
    niceScale,
    formatCompactCount,
    formatCompactBytes,
    formatAxisLabel,
    dedupeAxisTicks,
    formatValue,
    seriesMax,
    isEmptySeries,
    bandScale,
    barHeight,
    barPath,
    xLabelIndices,
    formatDayShort,
} from '../chart-utils.js'

describe('niceMax', () => {
    it('returns 0 for non-positive input (empty-state signal)', () => {
        expect(niceMax(0)).toBe(0)
        expect(niceMax(-5)).toBe(0)
        expect(niceMax(NaN)).toBe(0)
    })

    it('rounds up to the nearest 1/2/2.5/5/10 × 10^n', () => {
        expect(niceMax(1)).toBe(1)
        expect(niceMax(3)).toBe(5)
        expect(niceMax(7)).toBe(10)
        expect(niceMax(10)).toBe(10)
        expect(niceMax(42)).toBe(50)
        expect(niceMax(250)).toBe(250)
        expect(niceMax(1234)).toBe(2000)
    })
})

describe('niceScale', () => {
    it('gives a single baseline tick for all-zero data', () => {
        expect(niceScale(0)).toEqual({ max: 0, ticks: [0] })
    })

    it('gives baseline / midpoint / top ticks', () => {
        expect(niceScale(42)).toEqual({ max: 50, ticks: [0, 25, 50] })
        expect(niceScale(1234)).toEqual({ max: 2000, ticks: [0, 1000, 2000] })
    })
})

describe('formatCompactCount', () => {
    it('shows plain integers below 1000', () => {
        expect(formatCompactCount(0)).toBe('0')
        expect(formatCompactCount(5)).toBe('5')
        expect(formatCompactCount(999)).toBe('999')
    })

    it('abbreviates thousands / millions / billions', () => {
        expect(formatCompactCount(1000)).toBe('1k')
        expect(formatCompactCount(1200)).toBe('1.2k')
        expect(formatCompactCount(12000)).toBe('12k')
        expect(formatCompactCount(1500000)).toBe('1.5M')
        expect(formatCompactCount(2340000000)).toBe('2.3B')
    })

    it('guards against non-finite input', () => {
        expect(formatCompactCount(NaN)).toBe('0')
        expect(formatCompactCount(undefined)).toBe('0')
    })
})

describe('formatCompactBytes', () => {
    it('keeps axis byte labels narrow (no thousands decimals)', () => {
        expect(formatCompactBytes(0)).toBe('0')
        expect(formatCompactBytes(500)).toBe('500 B')
        // Uppercase "KB" matches humanReadableSize (utils.js) and the
        // distribution bucket labels — one consistent casing app-wide.
        expect(formatCompactBytes(2000)).toBe('2 KB')
        expect(formatCompactBytes(1500000)).toBe('1.5 MB')
        expect(formatCompactBytes(512000000)).toBe('512 MB')
        expect(formatCompactBytes(1000000000)).toBe('1 GB')
    })

    it('guards against non-positive / non-finite input', () => {
        expect(formatCompactBytes(-5)).toBe('0')
        expect(formatCompactBytes(NaN)).toBe('0')
    })
})

describe('formatAxisLabel', () => {
    it('uses compact counts for downloads', () => {
        expect(formatAxisLabel(1200, 'downloads')).toBe('1.2k')
    })

    it('uses compact bytes for the byte metrics', () => {
        expect(formatAxisLabel(0, 'downloadedBytes')).toBe('0')
        expect(formatAxisLabel(2000, 'downloadedBytes')).toBe('2 KB')
        expect(formatAxisLabel(1000000000, 'uploadedBytes')).toBe('1 GB')
    })
})

describe('dedupeAxisTicks', () => {
    it('drops the midpoint when max=1 rounds it to the same label as the top ("0/1/1")', () => {
        const ticks = niceScale(1).ticks // [0, 0.5, 1]
        expect(ticks.map((v) => formatAxisLabel(v, 'downloads'))).toEqual(['0', '1', '1'])

        const deduped = dedupeAxisTicks(ticks, 'downloads')
        expect(deduped).toEqual([0, 1])
        expect(deduped.map((v) => formatAxisLabel(v, 'downloads'))).toEqual(['0', '1'])
    })

    it('keeps all three ticks when max=3 has no duplicate ("0/3/5")', () => {
        const ticks = niceScale(3).ticks // [0, 2.5, 5]
        expect(ticks.map((v) => formatAxisLabel(v, 'downloads'))).toEqual(['0', '3', '5'])

        const deduped = dedupeAxisTicks(ticks, 'downloads')
        expect(deduped).toEqual(ticks)
    })

    it('passes through degenerate (single-tick) scales unchanged', () => {
        expect(dedupeAxisTicks([0], 'downloads')).toEqual([0])
        expect(dedupeAxisTicks([], 'downloads')).toEqual([])
    })
})

describe('formatValue', () => {
    it('groups download counts with thousands separators', () => {
        expect(formatValue(1234, 'downloads')).toBe('1,234')
        expect(formatValue(0, 'downloads')).toBe('0')
    })

    it('humanizes byte values and never returns an empty string', () => {
        expect(formatValue(0, 'downloadedBytes')).toBe('0 B')
        expect(formatValue(1048576, 'uploadedBytes')).toBe('1.05 MB')
    })

    it('threads an explicit locale override for count metrics', () => {
        const result = formatValue(3946, 'downloads', 'fr')
        expect(result).toBe((3946).toLocaleString('fr'))
        expect(result).not.toBe(formatValue(3946, 'downloads', 'en'))
    })
})

describe('seriesMax / isEmptySeries', () => {
    const series = [
        { day: '2026-05-01', downloads: 0, downloadedBytes: 0 },
        { day: '2026-05-02', downloads: 3, downloadedBytes: 1024 },
        { day: '2026-05-03', downloads: 7, downloadedBytes: 500 },
    ]

    it('finds the max of the selected metric', () => {
        expect(seriesMax(series, 'downloads')).toBe(7)
        expect(seriesMax(series, 'downloadedBytes')).toBe(1024)
    })

    it('returns 0 for empty or invalid series', () => {
        expect(seriesMax([], 'downloads')).toBe(0)
        expect(seriesMax(null, 'downloads')).toBe(0)
    })

    it('detects all-zero metrics', () => {
        expect(isEmptySeries(series, 'downloads')).toBe(false)
        expect(isEmptySeries([{ day: 'x', downloads: 0, downloadedBytes: 0 }], 'downloads')).toBe(true)
        expect(isEmptySeries([], 'downloadedBytes')).toBe(true)
    })
})

describe('bandScale', () => {
    it('splits the plot into centred equal bands', () => {
        expect(bandScale(4, 100, 0.5)).toEqual({ band: 25, barWidth: 12.5, offset: 6.25 })
    })

    it('returns zeros for degenerate input', () => {
        expect(bandScale(0, 100)).toEqual({ band: 0, barWidth: 0, offset: 0 })
        expect(bandScale(4, 0)).toEqual({ band: 0, barWidth: 0, offset: 0 })
    })
})

describe('barHeight', () => {
    it('scales value proportionally to the axis max', () => {
        expect(barHeight(50, 100, 80)).toBe(40)
        expect(barHeight(100, 100, 80)).toBe(80)
    })

    it('is 0 for zero value or a degenerate axis', () => {
        expect(barHeight(0, 100, 80)).toBe(0)
        expect(barHeight(5, 0, 80)).toBe(0)
        expect(barHeight(5, 100, 0)).toBe(0)
    })

    it('never exceeds the plot height', () => {
        expect(barHeight(150, 100, 80)).toBe(80)
    })
})

describe('barPath', () => {
    it('draws a bar with rounded top corners only', () => {
        expect(barPath(0, 0, 10, 20, 2)).toBe('M0,20V2Q0,0 2,0H8Q10,0 10,2V20Z')
    })

    it('returns an empty path for zero width or height', () => {
        expect(barPath(0, 0, 10, 0, 2)).toBe('')
        expect(barPath(0, 0, 0, 20, 2)).toBe('')
    })

    it('clamps the corner radius to half the width / full height', () => {
        expect(barPath(0, 0, 2, 1, 5)).toBe('M0,1V1Q0,0 1,0H1Q2,0 2,1V1Z')
    })
})

describe('xLabelIndices', () => {
    it('labels the newest bucket and every 7th before it', () => {
        expect(xLabelIndices(30, 7)).toEqual([1, 8, 15, 22, 29])
    })

    it('handles short and empty ranges', () => {
        expect(xLabelIndices(7, 7)).toEqual([6])
        expect(xLabelIndices(0, 7)).toEqual([])
    })
})

describe('formatDayShort', () => {
    it('formats a UTC calendar day as a short month/day label', () => {
        // Loose assertion (matching the formatDate test) so a non-en CI locale
        // cannot break the build: check the month + day parts are present.
        const result = formatDayShort('2026-05-04')
        expect(result).toContain('May')
        expect(result).toContain('4')
    })

    it('parses in UTC (no off-by-one day drift)', () => {
        // A negative-offset local timezone would roll this back to Dec 31 if
        // parsed as local time — assert the UTC day is preserved.
        const result = formatDayShort('2026-01-01')
        expect(result).toContain('Jan')
        expect(result).not.toContain('Dec')
    })

    it('returns an empty string for missing / invalid input', () => {
        expect(formatDayShort('')).toBe('')
        expect(formatDayShort('not-a-date')).toBe('')
    })

    it('defaults to the active vue-i18n locale (i18n.js starts at "en")', () => {
        expect(formatDayShort('2026-05-04')).toBe(formatDayShort('2026-05-04', 'en'))
    })

    it('threads an explicit locale override (French chart axis)', () => {
        const result = formatDayShort('2026-05-04', 'fr')
        const reference = new Date('2026-05-04T00:00:00Z')
            .toLocaleDateString('fr', { month: 'short', day: 'numeric', timeZone: 'UTC' })
        expect(result).toBe(reference)
        expect(result).not.toBe(formatDayShort('2026-05-04', 'en'))
    })
})
