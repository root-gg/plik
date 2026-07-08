<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDate } from '../utils.js'
import StatTile from './StatTile.vue'
import DistributionCard from './DistributionCard.vue'
import ActivityChart from './ActivityChart.vue'
import SegmentedControl from './SegmentedControl.vue'

const { t: $t } = useI18n()

// Shared current/lifetime statistics dashboard. Given the canonical `usage`
// object it renders the current/lifetime tile columns, one merged Activity card
// (a 4-metric selector driving both the window tiles AND the daily chart), and
// the six side-by-side distribution cards (feature / TTL / file-size). Scope-
// specific bits are configurable so the same panel serves both the admin (server)
// and the user scope:
//   - `currentTiles` / `lifetimeTiles` — the top counters (differ per scope)
//   - `#storage` slot — admin-only authenticated/anonymous split
//   - `showActivity` — hidden for scopes without windows/series
//   - `emptyStateEnabled` — Home-only "get started" empty state (see below)
const props = defineProps({
    // Canonical usage object: { startedAt, downloads, uploads, current, lifetime }
    usage: { type: Object, required: true },
    // Panel heading, e.g. "Server Statistics" / "User Statistics"
    title: { type: String, required: true },
    // [{ label, value, format }] — scope-specific top counters
    currentTiles: { type: Array, required: true },
    lifetimeTiles: { type: Array, required: true },
    // Render the merged Activity card (selector + window tiles + chart)
    showActivity: { type: Boolean, default: true },
    // Override the "since" date; defaults to usage.startedAt
    since: { type: String, default: '' },
    // Single colour treatment for every distribution bar
    barClass: { type: String, default: 'bg-accent-400' },
    // Dense oldest-first 30-day activity series
    // [{day, downloads, downloadedBytes, uploads, uploadedBytes}].
    // null/undefined (or empty) → the daily chart is omitted (tiles still render);
    // the parents fetch it alongside stats and leave it null on failure.
    dailySeries: { type: Array, default: null },
    // Home-only opt-in (default false, so Admin's behavior is unchanged): when
    // true AND lifetime.uploads is 0 (a genuinely fresh, no-activity scope),
    // the six distribution cards are replaced with one encouraging "upload
    // your first file" panel instead of a wall of zero-filled cards. Admin
    // never opts in — a brand-new server showing zeros is expected
    // admin information, not something to be encouraged past.
    emptyStateEnabled: { type: Boolean, default: false },
})

const sinceDate = computed(() => props.since || props.usage?.startedAt || '')

// Printed exactly once, in the lifetime column title (the clearer location).
const lifetimeTitle = computed(() =>
    sinceDate.value
        ? $t('statsPanel.lifetimeUsageSince', { date: formatDate(sinceDate.value) })
        : $t('statsPanel.lifetimeUsage'))

// ── Activity card: 4-metric selector drives both the window tiles and the chart ──
// Each metric maps to a usage sub-object (downloads/uploads) and either its count
// counters (total + today/last7Days/last30Days windows, always available from
// stats) or its byte volume (lifetime = usage.*.bytes; the bounded byte windows
// are summed from the daily series, so they need the — best-effort — chart data).
const metricDefs = [
    { key: 'downloads', usageKey: 'downloads', kind: 'count', i18nKey: 'statsPanel.metricDownloads' },
    { key: 'uploads', usageKey: 'uploads', kind: 'count', i18nKey: 'statsPanel.metricUploads' },
    { key: 'downloadedBytes', usageKey: 'downloads', kind: 'bytes', i18nKey: 'statsPanel.metricDownloadedData' },
    { key: 'uploadedBytes', usageKey: 'uploads', kind: 'bytes', i18nKey: 'statsPanel.metricUploadedData' },
]

// SegmentedControl's [{value, label}] shape — same computed-with-labels form
// as TrendingPanel's own window/metric option lists, so all three segmented
// controls in the stats dashboards are fed the same way.
const metricOptions = computed(() => metricDefs.map((m) => ({ value: m.key, label: $t(m.i18nKey) })))

const metric = ref('downloads')
const activeMetric = computed(() => metricDefs.find((m) => m.key === metric.value) || metricDefs[0])

// Sum the selected byte field over the most recent n points of the daily series.
function sumSeries(field, n) {
    const s = props.dailySeries || []
    const slice = s.slice(Math.max(0, s.length - n))
    return slice.reduce((acc, p) => acc + (Number(p?.[field]) || 0), 0)
}

const windowTiles = computed(() => {
    const m = activeMetric.value
    const u = props.usage?.[m.usageKey] || {}
    const fmt = m.kind === 'bytes' ? 'size' : 'count'
    let today, last7, last30, lifetime
    if (m.kind === 'bytes') {
        // Byte windows are summed from the daily series (the API exposes only
        // lifetime bytes). When the series is unavailable (parents null it on
        // a fetch error — see the `dailySeries` prop doc), a summed 0 would
        // read as a false "0 B" right next to a real, non-zero Lifetime value
        // — null instead, so StatTile renders "—" (unavailable, not zero).
        // Count metrics don't have this: their windows come from the stats
        // object directly, unaffected by the series fetch.
        const seriesAvailable = Array.isArray(props.dailySeries)
        today = seriesAvailable ? sumSeries(m.key, 1) : null
        last7 = seriesAvailable ? sumSeries(m.key, 7) : null
        last30 = seriesAvailable ? sumSeries(m.key, 30) : null
        lifetime = u.bytes || 0
    } else {
        today = u.today || 0
        last7 = u.last7Days || 0
        last30 = u.last30Days || 0
        lifetime = u.total || 0
    }
    return [
        { key: 'today', label: $t('statsPanel.windowToday'), value: today, format: fmt },
        { key: '7d', label: $t('statsPanel.window7d'), value: last7, format: fmt },
        { key: '30d', label: $t('statsPanel.window30d'), value: last30, format: fmt },
        { key: 'lifetime', label: $t('statsPanel.windowLifetime'), value: lifetime, format: fmt },
    ]
})

function featureItems(period) {
    const f = period?.features || {}
    return [
        { key: 'password', label: $t('badges.password'), value: f.passwordUploads || 0 },
        { key: 'removable', label: $t('badges.removable'), value: f.removableUploads || 0 },
        { key: 'oneShot', label: $t('badges.oneShot'), value: f.oneShotUploads || 0 },
        { key: 'stream', label: $t('badges.stream'), value: f.streamUploads || 0 },
        { key: 'extendTTL', label: $t('badges.extendTTL'), value: f.extendTTLUploads || 0 },
        { key: 'e2ee', label: $t('badges.encrypted'), value: f.e2eeUploads || 0 },
        { key: 'comment', label: $t('statsPanel.comments'), value: f.commentUploads || 0 },
    ]
}

function ttlItems(period) {
    const t = period?.ttl || {}
    return [
        { key: 'none', label: $t('statsPanel.ttlNone'), value: t.noneUploads || 0 },
        { key: 'lt1h', label: $t('statsPanel.ttlLt1h'), value: t.lessThan1HourUploads || 0 },
        { key: '1h1d', label: $t('statsPanel.ttl1h1d'), value: t.oneHourToOneDayUploads || 0 },
        { key: '1d7d', label: $t('statsPanel.ttl1d7d'), value: t.oneDayToSevenDaysUploads || 0 },
        { key: '7d30d', label: $t('statsPanel.ttl7d30d'), value: t.sevenDaysTo30DaysUploads || 0 },
        { key: 'gt30d', label: $t('statsPanel.ttlGt30d'), value: t.greaterThan30DaysUploads || 0 },
    ]
}

function fileSizeItems(period) {
    const s = period?.fileSizes || {}
    return [
        { key: 'lt1m', label: $t('statsPanel.fileSizeLt1m'), value: s.lessThan1MBFiles || 0 },
        { key: '1m10m', label: $t('statsPanel.fileSize1m10m'), value: s.oneMBTo10MBFiles || 0 },
        { key: '10m100m', label: $t('statsPanel.fileSize10m100m'), value: s.tenMBTo100MBFiles || 0 },
        { key: '100m1g', label: $t('statsPanel.fileSize100m1g'), value: s.hundredMBTo1GBFiles || 0 },
        { key: '1g10g', label: $t('statsPanel.fileSize1g10g'), value: s.oneGBTo10GBFiles || 0 },
        { key: '10g100g', label: $t('statsPanel.fileSize10g100g'), value: s.tenGBTo100GBFiles || 0 },
        { key: 'gt100g', label: $t('statsPanel.fileSizeGt100g'), value: s.greaterThan100GBFiles || 0 },
    ]
}

const current = computed(() => props.usage?.current || {})
const lifetime = computed(() => props.usage?.lifetime || {})

// Zero-state: lifetime.uploads is the broadest "has anything ever
// happened here" signal (current is always <= lifetime), so it alone is
// enough to detect a genuinely fresh, no-activity scope.
const isAllZero = computed(() => (props.usage?.lifetime?.uploads || 0) === 0)
const showEmptyState = computed(() => props.emptyStateEnabled && isAllZero.value)
</script>

<template>
  <div class="space-y-4">
    <!-- Current / lifetime counters + storage split -->
    <div class="glass-card p-5 space-y-5">
      <h3 class="text-sm text-surface-400 uppercase tracking-wider">{{ title }}</h3>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div class="rounded-lg border border-surface-700/50 p-4">
          <p class="text-xs text-surface-500 uppercase tracking-wider">{{ $t('statsPanel.currentUsage') }}</p>
          <!-- On a low-churn/fresh scope this column can be byte-identical to
               Lifetime below — a one-line caption makes that intentional, not a bug. -->
          <p class="text-[11px] text-surface-600 mb-3">{{ $t('statsPanel.currentUsageCaption') }}</p>
          <div class="grid gap-4 text-center"
               :class="currentTiles.length >= 4 ? 'grid-cols-2 sm:grid-cols-4' : 'grid-cols-3'">
            <StatTile v-for="tile in currentTiles" :key="tile.label"
                      :label="tile.label" :value="tile.value" :format="tile.format || 'count'" />
          </div>
        </div>
        <div class="rounded-lg border border-surface-700/50 p-4">
          <p class="text-xs text-surface-500 uppercase tracking-wider">{{ lifetimeTitle }}</p>
          <p class="text-[11px] text-surface-600 mb-3">{{ $t('statsPanel.lifetimeUsageCaption') }}</p>
          <div class="grid gap-4 text-center"
               :class="lifetimeTiles.length >= 4 ? 'grid-cols-2 sm:grid-cols-4' : 'grid-cols-3'">
            <StatTile v-for="tile in lifetimeTiles" :key="tile.label"
                      :label="tile.label" :value="tile.value" :format="tile.format || 'count'" />
          </div>
        </div>
      </div>

      <!-- Scope-specific extra (admin: authenticated/anonymous storage split) -->
      <slot name="storage" />
    </div>

    <!-- Merged Activity card: metric selector drives BOTH the window tiles and the chart -->
    <div v-if="showActivity" class="glass-card p-5 space-y-4">
      <div class="flex items-center justify-between gap-3 flex-wrap">
        <h3 class="text-sm text-surface-400 uppercase tracking-wider">{{ $t('statsPanel.activity') }}</h3>
        <SegmentedControl v-model="metric" :options="metricOptions" :aria-label="$t('statsPanel.activity')" />
      </div>

      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <StatTile v-for="tile in windowTiles" :key="tile.key"
                  :label="tile.label" :value="tile.value" :format="tile.format" bordered value-class="text-accent-300" />
      </div>

      <!-- Daily chart for the selected metric (parents fetch the series; absent → tiles still render) -->
      <ActivityChart v-if="dailySeries && dailySeries.length" :series="dailySeries" :metric="metric" />
    </div>

    <!-- Zero-state (Home only, emptyStateEnabled + genuinely no activity ever):
         one encouraging panel instead of six zero-filled distribution cards.
         This is a distinct empty state, not a collapse/toggle over the
         same data — Admin never sees it (emptyStateEnabled defaults false). -->
    <div v-if="showEmptyState" class="glass-card p-8 text-center space-y-4">
      <p class="text-surface-300">{{ $t('statsPanel.emptyStateMessage') }}</p>
      <router-link to="/" class="btn-primary inline-flex items-center px-4 py-2 text-sm">
        {{ $t('statsPanel.emptyStateCta') }}
      </router-link>
    </div>
    <template v-else>
      <!-- File-size distribution (current | lifetime) -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <DistributionCard :title="$t('statsPanel.currentFileSizeDistribution')" :caption="$t('statsPanel.currentUsageCaption')"
                          :items="fileSizeItems(current)" :total="current.files || 0" :bar-class="barClass" />
        <DistributionCard :title="$t('statsPanel.lifetimeFileSizeDistribution')" :caption="$t('statsPanel.lifetimeUsageCaption')"
                          :items="fileSizeItems(lifetime)" :total="lifetime.files || 0" :bar-class="barClass" />
      </div>

      <!-- TTL distribution (current | lifetime) -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <DistributionCard :title="$t('statsPanel.currentTTLDistribution')" :caption="$t('statsPanel.currentUsageCaption')"
                          :items="ttlItems(current)" :total="current.uploads || 0" :bar-class="barClass" />
        <DistributionCard :title="$t('statsPanel.lifetimeTTLDistribution')" :caption="$t('statsPanel.lifetimeUsageCaption')"
                          :items="ttlItems(lifetime)" :total="lifetime.uploads || 0" :bar-class="barClass" />
      </div>

      <!-- Feature usage (current | lifetime) -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <DistributionCard :title="$t('statsPanel.currentFeatureUsage')" :caption="$t('statsPanel.currentUsageCaption')"
                          :items="featureItems(current)" :total="current.uploads || 0" :bar-class="barClass" />
        <DistributionCard :title="$t('statsPanel.lifetimeFeatureUsage')" :caption="$t('statsPanel.lifetimeUsageCaption')"
                          :items="featureItems(lifetime)" :total="lifetime.uploads || 0" :bar-class="barClass" />
      </div>
    </template>
  </div>
</template>
