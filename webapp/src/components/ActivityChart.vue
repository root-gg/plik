<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
    seriesMax,
    niceScale,
    bandScale,
    barHeight,
    barPath,
    isEmptySeries,
    isByteMetric,
    formatAxisLabel,
    dedupeAxisTicks,
    formatValue,
    formatDayShort,
    xLabelIndices,
} from '../chart-utils.js'

// Presentational 30-day daily activity chart — hand-rolled SVG, no charting
// dependency. A single-series column chart (bars are honest for discrete daily
// buckets). The measure is CONTROLLED by the parent via the `metric` prop (the
// segmented selector lives on the Activity card in StatsUsagePanel), so this
// component holds no selection state of its own. All colour/text/grid comes from
// theme tokens (scoped CSS below + token utility classes) so re-theming needs no
// change here.
const props = defineProps({
    // Dense, oldest-first [{ day: 'YYYY-MM-DD', downloads, downloadedBytes, uploads, uploadedBytes }]
    series: { type: Array, default: () => [] },
    // Selected measure key: 'downloads' | 'uploads' | 'downloadedBytes' | 'uploadedBytes'
    metric: { type: String, default: 'downloads' },
})

const { t: $t } = useI18n()

const hovered = ref(null)

// Grow-in animation only when the user has not asked for reduced motion.
// jsdom leaves matchMedia undefined, so guard it; the scoped @media rule is a
// second, CSS-level safety net.
const prefersReducedMotion = (() => {
    try { return !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches) }
    catch { return false }
})()
const animateBars = !prefersReducedMotion

// ── Logical drawing space (viewBox units). The SVG scales to container width. ──
const VIEW_W = 320
const VIEW_H = 132
// Left gutter fits humanized y-axis labels (incl. compact bytes like "1.5 MB").
const PAD = { top: 8, right: 6, bottom: 16, left: 40 }
const plotX = PAD.left
const plotY = PAD.top
const plotW = VIEW_W - PAD.left - PAD.right
const plotH = VIEW_H - PAD.top - PAD.bottom
const r2 = (n) => Math.round(n * 100) / 100

const empty = computed(() => isEmptySeries(props.series, props.metric))

const layout = computed(() => {
    const s = props.series || []
    const count = s.length
    const scale = niceScale(seriesMax(s, props.metric))
    const { band, barWidth, offset } = bandScale(count, plotW, 0.34)

    const cols = s.map((p, i) => {
        const value = Number(p?.[props.metric]) || 0
        const h = barHeight(value, scale.max, plotH)
        const x = plotX + offset + i * band
        const y = plotY + plotH - h
        return {
            i,
            day: p?.day || '',
            value,
            hitX: r2(plotX + i * band),
            hitW: r2(band),
            center: r2(x + barWidth / 2),
            path: barPath(r2(x), r2(y), r2(barWidth), r2(h), 1.5),
        }
    })

    const yTicks = dedupeAxisTicks(scale.max > 0 ? scale.ticks : [0], props.metric).map((v) => ({
        value: v,
        y: r2(plotY + plotH - (scale.max > 0 ? (v / scale.max) * plotH : 0)),
        label: formatAxisLabel(v, props.metric),
    }))

    const xTicks = xLabelIndices(count, 7).map((i) => ({
        x: cols[i]?.center ?? 0,
        label: formatDayShort(s[i]?.day),
    }))

    return { count, cols, bars: cols.filter((c) => c.path), yTicks, xTicks, baselineY: r2(plotY + plotH) }
})

const hoveredCol = computed(() => (hovered.value == null ? null : layout.value.cols[hovered.value] || null))
const tooltipDay = computed(() => (hoveredCol.value ? formatDayShort(hoveredCol.value.day) : ''))
const tooltipValue = computed(() => (hoveredCol.value ? formatValue(hoveredCol.value.value, props.metric) : ''))

// Flip the tooltip's anchor edge across the mid-line so it never clips left/right;
// pinned near the top of the plot so it never clips above.
const tooltipStyle = computed(() => {
    const c = hoveredCol.value
    if (!c) return {}
    const frac = c.center / VIEW_W
    const style = { top: '2px' }
    if (frac <= 0.5) style.left = `${(frac * 100).toFixed(2)}%`
    else style.right = `${((1 - frac) * 100).toFixed(2)}%`
    return style
})

// Per-metric aria/empty copy: bytes metrics read as "data", count metrics as events.
const ariaLabelKey = computed(() => ({
    downloads: 'statsPanel.chartAriaDownloads',
    uploads: 'statsPanel.chartAriaUploads',
    downloadedBytes: 'statsPanel.chartAriaDownloadedData',
    uploadedBytes: 'statsPanel.chartAriaUploadedData',
}[props.metric] || 'statsPanel.chartAriaDownloads'))

const emptyMessageKey = computed(() => ({
    downloads: 'statsPanel.chartEmptyDownloads',
    uploads: 'statsPanel.chartEmptyUploads',
    downloadedBytes: 'statsPanel.chartEmptyDownloadedData',
    uploadedBytes: 'statsPanel.chartEmptyUploadedData',
}[props.metric] || 'statsPanel.chartEmptyDownloads'))

const ariaLabel = computed(() => $t(ariaLabelKey.value, { days: (props.series || []).length }))
const emptyMessage = computed(() => $t(emptyMessageKey.value))

// Kept for tests/readability: whether the current metric is a byte measure.
const bytesMode = computed(() => isByteMetric(props.metric))

const rightEdge = VIEW_W - PAD.right
</script>

<template>
  <div>
    <div class="relative">
      <svg :viewBox="`0 0 ${VIEW_W} ${VIEW_H}`" role="img" :aria-label="ariaLabel"
           class="block w-full h-auto overflow-visible" data-testid="activity-chart" :data-bytes-mode="bytesMode"
           @mouseleave="hovered = null">
        <!-- Recessive horizontal gridlines -->
        <line v-for="t in layout.yTicks" :key="'g' + t.value" class="chart-grid"
              :x1="plotX" :x2="rightEdge" :y1="t.y" :y2="t.y" />
        <!-- Baseline / x-axis -->
        <line class="chart-baseline" :x1="plotX" :x2="rightEdge" :y1="layout.baselineY" :y2="layout.baselineY" />
        <!-- Y-axis tick labels (humanized) -->
        <text v-for="t in layout.yTicks" :key="'yl' + t.value" class="chart-tick"
              :x="plotX - 4" :y="t.y + 2.8" text-anchor="end">{{ t.label }}</text>
        <!-- Hovered column highlight -->
        <rect v-if="hoveredCol" class="chart-col-hl"
              :x="hoveredCol.hitX" :y="plotY" :width="hoveredCol.hitW" :height="plotH" />
        <!-- Bars (rounded top only) -->
        <path v-for="bar in layout.bars" :key="'b' + bar.i" :d="bar.path"
              class="chart-bar" :class="{ 'chart-bar--hot': hovered === bar.i, 'chart-bar--animate': animateBars }" />
        <!-- X-axis tick labels (sparse) -->
        <text v-for="t in layout.xTicks" :key="'xl' + t.x" class="chart-tick"
              :x="t.x" :y="VIEW_H - 4" text-anchor="middle">{{ t.label }}</text>
        <!-- Full-column hover hit targets (transparent, on top) -->
        <rect v-for="col in layout.cols" :key="'h' + col.i" class="chart-hit"
              :x="col.hitX" :y="plotY" :width="col.hitW" :height="plotH"
              @mouseenter="hovered = col.i" />
      </svg>

      <!-- Honest empty state: axis + baseline stay, quiet line over the plot -->
      <div v-if="empty" class="absolute inset-0 flex items-center justify-center pointer-events-none">
        <span class="text-xs text-surface-500">{{ emptyMessage }}</span>
      </div>

      <!-- Hover tooltip (day + humanized value); day is UTC, captioned once below -->
      <div v-if="hoveredCol" role="tooltip" :style="tooltipStyle" data-testid="activity-chart-tooltip"
           class="absolute z-10 pointer-events-none whitespace-nowrap rounded-md border border-surface-600/60 bg-surface-800 px-2 py-1 text-xs shadow-lg">
        <span class="text-surface-400">{{ tooltipDay }}</span>
        <span class="mx-1 text-surface-600">·</span>
        <span class="text-surface-100 tabular-nums font-medium">{{ tooltipValue }}</span>
      </div>
    </div>

    <p class="mt-2 text-right text-[10px] text-surface-500 uppercase tracking-wider">{{ $t('statsPanel.chartUtcDays') }}</p>
  </div>
</template>

<style scoped>
.chart-bar {
    fill: var(--color-accent-500);
    transform-box: fill-box;
    transform-origin: bottom;
}

.chart-bar--hot {
    fill: var(--color-accent-400);
}

.chart-bar--animate {
    animation: chart-grow 0.5s ease-out;
}

.chart-col-hl {
    fill: var(--color-accent-400);
    opacity: 0.1;
}

.chart-grid {
    stroke: var(--color-surface-700);
    stroke-width: 0.5;
}

.chart-baseline {
    stroke: var(--color-surface-600);
    stroke-width: 0.75;
}

.chart-tick {
    fill: var(--color-surface-400);
    font-size: 8px;
}

.chart-hit {
    fill: transparent;
    cursor: pointer;
}

@keyframes chart-grow {
    from { transform: scaleY(0); }
    to { transform: scaleY(1); }
}

@media (prefers-reduced-motion: reduce) {
    .chart-bar--animate { animation: none; }
}
</style>
