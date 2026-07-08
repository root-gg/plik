<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { humanReadableSize, formatSizeOrZero, formatDate } from '../utils.js'
import SegmentedControl from './SegmentedControl.vue'

// Reusable trending panel: window chips + a Trending Uploads card (with a
// Downloads / Downloaded data metric toggle) + an optional Trending Files
// card. Extracted verbatim from AdminView.vue so Admin's rendered
// output/classes/i18n keys/behavior are unchanged — this component is a pure
// presentational shell: all state (window/sort/data fetching) lives in the
// parent, exactly like the inline markup it replaces.
//
// `mode` distinguishes the admin (server-wide, cross-user) card from the
// self-scoped one used on Home: 'self' trending is strictly the caller's own
// uploads (server-enforced via GET /me/stats/trending/uploads), so per-row
// owner/anonymous chips and the admin-only "view user in users tab" action —
// which only make sense when trending spans multiple users — are omitted.
const props = defineProps({
    // Current trending window: '1d' | '7d' | '30d' | 'all'
    window: { type: String, required: true },
    // Current trending-uploads sort: 'downloads' | 'downloadedBytes'
    sort: { type: String, default: 'downloads' },
    uploads: { type: Array, default: () => [] },
    files: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
    // Home's self-scoped trending has no Trending Files card — there is no
    // self-scoped files endpoint, since files have no lifetime byte column and
    // file-grain byte trending is out of scope. Admin always passes true.
    showFiles: { type: Boolean, default: true },
    mode: { type: String, default: 'admin' }, // 'admin' | 'self'
})

const emit = defineEmits(['update:window', 'update:sort', 'open-upload', 'view-user'])

const { t: $t } = useI18n()

// Window chips: display the same Today / 7 days / 30 days / Lifetime
// vocabulary as the Activity tiles (statsPanel.window*), so "Lifetime" means
// the same thing everywhere on this page — never reintroduce a separate
// downloads*/"All time" key set, since "Lifetime" is the one canonical label
// across the app. The API param values stay the short '1d'/'7d'/'30d'/'all' form.
const windowOptions = computed(() => [
    { value: '1d', label: $t('statsPanel.windowToday') },
    { value: '7d', label: $t('statsPanel.window7d') },
    { value: '30d', label: $t('statsPanel.window30d') },
    { value: 'all', label: $t('statsPanel.windowLifetime') },
])

// Metric toggle: reuses the Activity panel's own metric labels (statsPanel.*)
// rather than minting new ones — same concept, same words: the user-facing
// label is always "downloaded data", never "egress".
const metricOptions = computed(() => [
    { value: 'downloads', label: $t('statsPanel.metricDownloads') },
    { value: 'downloadedBytes', label: $t('statsPanel.metricDownloadedData') },
])

// "129 downloads" — reuses the same count+lowercased-label pattern already
// used for the files count below ("2 files"), so no new i18n key is needed.
const downloadsWord = computed(() => $t('statsPanel.metricDownloads').toLowerCase())

// Self-scoped trending (Home) gets its own "Your Trending Uploads" title
// instead of Admin's cross-user "Trending Uploads" — the outer "Trending"
// heading above stays shared (still a truthful, non-leaking generic label),
// but the uploads card itself should read as "yours", not "the server's".
const uploadsTitle = computed(() => props.mode === 'self' ? $t('homeView.trendingUploads') : $t('adminView.trendingUploads'))

function uploadHeadline(item) {
    if (item.comments) return item.comments
    if (item.user) return $t('adminView.uploadByOwner', { owner: item.user })
    return $t('adminView.anonymousUploadLabel')
}
</script>

<template>
  <div class="glass-card p-5 space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
      <h3 class="text-sm text-surface-400 uppercase tracking-wider">{{ $t('adminView.trending') }}</h3>
      <SegmentedControl :options="windowOptions" :model-value="window" :aria-label="$t('adminView.trending')"
                         @update:model-value="emit('update:window', $event)" />
    </div>
    <div v-if="loading" class="text-center py-8 text-surface-500">{{ $t('common.loading') }}</div>
    <div v-else class="grid grid-cols-1 gap-4" :class="showFiles ? 'lg:grid-cols-2' : ''">
      <div>
        <div class="flex flex-wrap items-center justify-between gap-2 mb-2">
          <p class="text-xs text-surface-500 uppercase tracking-wider">{{ uploadsTitle }}</p>
          <SegmentedControl :options="metricOptions" :model-value="sort" :aria-label="uploadsTitle"
                             @update:model-value="emit('update:sort', $event)" />
        </div>
        <div class="space-y-2">
          <div v-for="item in uploads" :key="item.id" class="rounded-lg border border-surface-700/50 p-3 text-sm">
            <div class="flex items-start justify-between gap-3">
              <!-- Human-first headline: the comment when present, else "Upload by
                   <owner>" / "Anonymous upload" — mirrors Trending Files'
                   filename-first headline instead of leading with the ID. -->
              <button type="button"
                      @click="emit('open-upload', item.id)"
                      class="min-w-0 text-left text-surface-200 hover:text-accent-300 transition-colors"
                      :title="$t('adminView.viewUploads')">
                <span class="block truncate">{{ uploadHeadline(item) }}</span>
              </button>
              <!-- Both metrics always shown; the sorted-by one is emphasized
                   (brighter/bold, and listed first) — mirrors how the Activity
                   panel emphasizes its selected metric. -->
              <span class="shrink-0 text-right tabular-nums">
                <template v-if="sort === 'downloadedBytes'">
                  <span class="text-surface-200 font-medium">{{ formatSizeOrZero(item.downloadedBytes) }}</span>
                  <span class="text-surface-500"> · {{ item.downloadCount }} {{ downloadsWord }}</span>
                </template>
                <template v-else>
                  <span class="text-surface-200 font-medium">{{ item.downloadCount }} {{ downloadsWord }}</span>
                  <span class="text-surface-500"> · {{ formatSizeOrZero(item.downloadedBytes) }}</span>
                </template>
              </span>
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-surface-500">
              <span>{{ item.files }} {{ $t('adminView.files').toLowerCase() }} · {{ item.lastDownloadedAt ? formatDate(item.lastDownloadedAt) : $t('common.never') }}</span>
              <!-- ID demoted to a small monospace subtitle (same link target as before) -->
              <button type="button"
                      @click="emit('open-upload', item.id)"
                      class="max-w-full truncate font-mono text-accent-400 hover:text-accent-300 transition-colors"
                      :title="$t('adminView.viewUploads')">
                {{ item.id }}
              </button>
              <template v-if="mode === 'admin'">
                <button v-if="item.user"
                        type="button"
                        @click="emit('view-user', item.user)"
                        class="max-w-full truncate text-accent-400 hover:text-accent-300 transition-colors"
                        :title="$t('adminView.viewUserInUsersTab')">
                  {{ item.user }}
                </button>
                <span v-else>{{ $t('adminView.anonymous') }}</span>
              </template>
            </div>
          </div>
          <p v-if="uploads.length === 0" class="text-sm text-surface-500">{{ $t('adminView.noTrendingUploads') }}</p>
        </div>
      </div>
      <div v-if="showFiles">
        <p class="text-xs text-surface-500 uppercase tracking-wider mb-2">{{ $t('adminView.trendingFiles') }}</p>
        <div class="space-y-2">
          <div v-for="item in files" :key="item.id" class="rounded-lg border border-surface-700/50 p-3 text-sm">
            <div class="flex items-start justify-between gap-3">
              <button type="button"
                      @click="emit('open-upload', item.uploadID, item.id)"
                      class="min-w-0 text-left text-surface-200 hover:text-accent-300 transition-colors"
                      :title="$t('adminView.viewUploads')">
                <span class="block truncate">{{ item.name || item.id }}</span>
              </button>
              <span class="text-surface-200 tabular-nums">{{ item.downloadCount }}</span>
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-surface-500">
              <span>{{ humanReadableSize(item.size) }} · {{ item.lastDownloadedAt ? formatDate(item.lastDownloadedAt) : $t('common.never') }}</span>
              <button v-if="item.uploadID"
                      type="button"
                      @click="emit('open-upload', item.uploadID)"
                      class="max-w-full truncate font-mono text-accent-400 hover:text-accent-300 transition-colors"
                      :title="$t('adminView.viewUploads')">
                {{ item.uploadID }}
              </button>
              <button v-if="item.user"
                      type="button"
                      @click="emit('view-user', item.user)"
                      class="max-w-full truncate text-accent-400 hover:text-accent-300 transition-colors"
                      :title="$t('adminView.viewUserInUsersTab')">
                {{ item.user }}
              </button>
              <span v-else>{{ $t('adminView.anonymous') }}</span>
            </div>
          </div>
          <p v-if="files.length === 0" class="text-sm text-surface-500">{{ $t('adminView.noTrendingFiles') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
