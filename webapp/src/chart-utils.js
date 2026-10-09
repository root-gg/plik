// Pure geometry / scale / label helpers for ActivityChart.vue.
//
// Kept dependency-free (aside from the shared humanized formatters in utils.js
// and the active-locale getter in i18n.js — reading i18n.global.locale.value
// has no DOM/runtime dependency of its own) and side-effect-free so the scale
// maths, tick selection and label formatting can be unit-tested with exact
// values, independent of any DOM or Vue runtime. Locale-sensitive formatters
// (formatDayShort, formatValue's count branch) accept an optional `locale`
// override for exactly this reason.

import { formatSizeOrZero, formatCount } from './utils.js'
import { getLocale } from './i18n.js'

/**
 * True when a metric key measures bytes (humanized as data sizes) rather than a
 * discrete count. The activity series exposes four measures: 'downloads' and
 * 'uploads' are counts; 'downloadedBytes' and 'uploadedBytes' are byte volumes.
 */
export function isByteMetric(metric) {
    return metric === 'downloadedBytes' || metric === 'uploadedBytes'
}

/**
 * Round a value up to a "nice" axis ceiling: the smallest of
 * {1, 2, 2.5, 5, 10} × 10^n that is >= value. Non-positive → 0 (the caller
 * renders the empty state instead of a degenerate axis).
 */
export function niceMax(value) {
    const v = Number(value)
    if (!(v > 0)) return 0
    const exp = Math.floor(Math.log10(v))
    const base = Math.pow(10, exp)
    for (const s of [1, 2, 2.5, 5, 10]) {
        const candidate = s * base
        if (v <= candidate + 1e-9) return candidate
    }
    return 10 * base
}

/**
 * Y-axis scale for the chart: a nice ceiling plus at most three gridline
 * values (baseline, midpoint, top). All-zero data → a single [0] tick so the
 * empty state still draws a baseline.
 */
export function niceScale(maxValue) {
    const top = niceMax(maxValue)
    if (top <= 0) return { max: 0, ticks: [0] }
    return { max: top, ticks: [0, top / 2, top] }
}

/**
 * Compact count label for y-axis ticks: 1200 → "1.2k", 1_500_000 → "1.5M".
 * Values below 1000 are shown as plain rounded integers.
 */
export function formatCompactCount(value) {
    const n = Number(value)
    if (!Number.isFinite(n)) return '0'
    const abs = Math.abs(n)
    if (abs < 1000) return String(Math.round(n))
    for (const u of [{ limit: 1e9, suffix: 'B' }, { limit: 1e6, suffix: 'M' }, { limit: 1e3, suffix: 'k' }]) {
        if (abs >= u.limit) {
            const scaled = Math.round((n / u.limit) * 10) / 10
            return `${scaled}${u.suffix}`
        }
    }
    return String(Math.round(n))
}

/**
 * Compact byte label for y-axis ticks: no thousands decimals so the label
 * stays narrow enough for the axis gutter (2000 → "2 KB", 1.5e6 → "1.5 MB",
 * 1e9 → "1 GB"). Sub-10 values keep one decimal; 10+ round to an integer.
 */
export function formatCompactBytes(bytes) {
    const n = Number(bytes)
    if (!Number.isFinite(n) || n <= 0) return '0'
    // Uppercase "KB" to match humanReadableSize (utils.js) and the
    // distribution bucket labels — one consistent casing app-wide.
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    const k = 1000
    const i = Math.min(units.length - 1, Math.floor(Math.log(n) / Math.log(k)))
    const val = n / Math.pow(k, i)
    const rounded = val >= 10 ? Math.round(val) : Math.round(val * 10) / 10
    return `${rounded} ${units[i]}`
}

/**
 * Humanized y-axis tick label: bytes via the compact byte formatter (e.g.
 * "1.5 MB"), download counts via the compact count formatter (e.g. "1.2k").
 * Kept narrow on purpose — full-precision values live in the hover tooltip.
 */
export function formatAxisLabel(value, metric) {
    if (isByteMetric(metric)) return formatCompactBytes(value)
    return formatCompactCount(value)
}

/**
 * Y-axis tick values to actually render, deduping a midpoint that would
 * collide with the baseline's or top's rendered label. A tiny scale (e.g.
 * niceScale(1) -> ticks [0, 0.5, 1]) rounds its midpoint up to the same label
 * as the top ("1"), producing a visually duplicated "0 / 1 / 1" axis; dropping
 * the midpoint in that case leaves the honest, non-duplicated "0 / 1". The
 * baseline and top ticks are always kept — only the midpoint can be dropped —
 * and a scale whose midpoint label is genuinely distinct (e.g. niceScale(3) ->
 * "0 / 3 / 5") is returned unchanged.
 */
export function dedupeAxisTicks(ticks, metric) {
    if (!Array.isArray(ticks) || ticks.length < 3) return ticks || []
    const [base, mid, top] = ticks
    const midLabel = formatAxisLabel(mid, metric)
    if (midLabel === formatAxisLabel(base, metric) || midLabel === formatAxisLabel(top, metric)) {
        return [base, top]
    }
    return ticks
}

/**
 * Humanized tooltip value: bytes via humanReadableSize, counts thousands-grouped
 * (e.g. "1,234"). Bytes never render an empty string (0 → "0 B"). `locale`
 * (count metrics only) defaults to the active vue-i18n locale — see
 * formatCount in utils.js.
 */
export function formatValue(value, metric, locale) {
    if (isByteMetric(metric)) return formatSizeOrZero(value)
    return formatCount(value, locale)
}

/** Maximum of the selected metric across a series (0 for empty/invalid input). */
export function seriesMax(series, metric) {
    if (!Array.isArray(series) || series.length === 0) return 0
    let max = 0
    for (const point of series) {
        const v = Number(point?.[metric]) || 0
        if (v > max) max = v
    }
    return max
}

/** True when the selected metric is all-zero (or the series is empty). */
export function isEmptySeries(series, metric) {
    return seriesMax(series, metric) <= 0
}

/**
 * Band scale for evenly spaced bars: splits plotWidth into `count` equal bands,
 * each holding a centred bar occupying (1 - gapRatio) of its band.
 */
export function bandScale(count, plotWidth, gapRatio = 0.3) {
    if (!(count > 0) || !(plotWidth > 0)) return { band: 0, barWidth: 0, offset: 0 }
    const g = Math.min(Math.max(gapRatio, 0), 0.9)
    const band = plotWidth / count
    const barWidth = band * (1 - g)
    return { band, barWidth, offset: (band - barWidth) / 2 }
}

/** Pixel height of a bar for `value` on a [0, max] axis of height plotHeight. */
export function barHeight(value, max, plotHeight) {
    const v = Number(value) || 0
    if (!(max > 0) || !(plotHeight > 0) || v <= 0) return 0
    return Math.min(plotHeight, (v / max) * plotHeight)
}

/**
 * SVG path for a bar with rounded top corners only (square base sitting on the
 * axis). Zero-width/height bars return '' so nothing is drawn.
 */
export function barPath(x, y, w, h, r = 2) {
    if (!(w > 0) || !(h > 0)) return ''
    const rr = Math.min(r, w / 2, h)
    return `M${x},${y + h}V${y + rr}Q${x},${y} ${x + rr},${y}H${x + w - rr}Q${x + w},${y} ${x + w},${y + rr}V${y + h}Z`
}

/**
 * Indices to label on the x-axis: the most recent bucket and every `every`th
 * bucket before it, in ascending order (e.g. 30 days → [1, 8, 15, 22, 29]).
 */
export function xLabelIndices(count, every = 7) {
    const out = []
    if (!(count > 0) || !(every > 0)) return out
    for (let i = count - 1; i >= 0; i -= every) out.push(i)
    return out.reverse()
}

/**
 * Format a "YYYY-MM-DD" UTC calendar day as a short month/day label (e.g.
 * "May 4"), parsed and rendered in UTC so the label never drifts a day.
 * `locale` defaults to the active vue-i18n locale (getLocale()) so the chart
 * axis follows the app's language picker instead of the browser's own
 * locale; pass an explicit locale to override (used by tests).
 */
export function formatDayShort(dayStr, locale) {
    if (!dayStr) return ''
    const d = new Date(`${dayStr}T00:00:00Z`)
    if (Number.isNaN(d.getTime())) return ''
    return d.toLocaleDateString(locale || getLocale(), { month: 'short', day: 'numeric', timeZone: 'UTC' })
}
