<script setup>
import { computed } from 'vue'
import { humanReadableSize, getUploadUrl, formatDate } from '../utils.js'
import { getFileURL } from '../api.js'
import UploadBadges from './UploadBadges.vue'

const props = defineProps({
    upload: { type: Object, required: true },
    tokenLabel: { type: String, default: '' },  // pre-formatted token label
    showUser: { type: Boolean, default: false },
})

const emit = defineEmits(['delete', 'filter-token', 'filter-user'])

// Total size is the sum of this upload's own files, computed client-side —
// the API does not (and should not) ship a redundant precomputed field for
// data already present in upload.files.
const totalSize = computed(() => (props.upload.files || []).reduce((sum, file) => sum + (file.fileSize || 0), 0))
</script>

<template>
  <div class="glass-card p-4">
    <div class="flex flex-col sm:flex-row gap-4">
      <!-- Upload meta -->
      <div class="sm:w-1/3 text-sm space-y-1">
        <a :href="getUploadUrl(upload)"
           class="font-mono text-accent-400 hover:text-accent-300 transition-colors">
          {{ upload.id }}
        </a>
        <p class="text-surface-500">{{ $t('uploadCard.uploaded') }} {{ formatDate(upload.createdAt) }}</p>
        <p class="text-surface-500">{{ $t('uploadCard.expires') }} {{ upload.expireAt ? formatDate(upload.expireAt) : $t('common.never') }}</p>
        <p class="text-surface-500">
          {{ $t('uploadCard.totalSize') }} <span class="text-surface-300 tabular-nums">{{ humanReadableSize(totalSize) }}</span>
        </p>
        <p v-if="upload.downloadCount !== undefined" class="text-surface-500">
          {{ $t('uploadCard.downloads') }} <span class="text-surface-300 tabular-nums">{{ upload.downloadCount || 0 }}</span>
        </p>
        <p v-if="upload.downloadedBytes !== undefined" class="text-surface-500">
          {{ $t('uploadCard.downloadedData') }} <span class="text-surface-300 tabular-nums">{{ humanReadableSize(upload.downloadedBytes) }}</span>
        </p>
        <p v-if="upload.downloadCount !== undefined" class="text-surface-500">
          {{ $t('uploadCard.lastDownload') }} {{ upload.lastDownloadedAt ? formatDate(upload.lastDownloadedAt) : $t('common.never') }}
        </p>
        <UploadBadges :upload="upload" size="sm" class="mt-1" />
        <p v-if="showUser && upload.user" class="text-surface-500">
          {{ $t('uploadCard.user') }}
          <button @click="emit('filter-user', upload.user)"
                  class="text-accent-400 hover:text-accent-300 transition-colors">
            {{ upload.user }}
          </button>
        </p>
        <p v-if="upload.token" class="text-surface-500">
          {{ $t('uploadCard.token') }}
          <button @click="emit('filter-token', upload.token)"
                  class="text-accent-400 hover:text-accent-300 transition-colors">
            {{ tokenLabel || upload.token }}
          </button>
        </p>
      </div>

      <!-- Files -->
      <div class="flex-1 min-w-0 text-sm space-y-1">
        <div v-for="file in (upload.files || [])"
             :key="file.id"
             class="flex items-center justify-between gap-2"
             :class="{ 'opacity-50': file.status !== 'uploaded' }">
          <div class="flex items-center gap-1.5 min-w-0">
            <!-- Status badges -->
            <span v-if="file.status === 'missing'"
                  class="shrink-0 w-4 h-4 rounded-full bg-warning-500/15 text-warning-500 text-[10px] font-bold flex items-center justify-center cursor-default"
                  :title="$t('uploadCard.missing')">m</span>
            <span v-else-if="file.status === 'uploading'"
                  class="shrink-0 w-4 h-4 rounded-full bg-accent-500/15 text-accent-400 text-[10px] font-bold flex items-center justify-center cursor-default"
                  :title="$t('fileRow.uploading')">u</span>
            <span v-else-if="file.status === 'removed'"
                  class="shrink-0 w-4 h-4 rounded-full bg-danger-500/15 text-danger-500 text-[10px] font-bold flex items-center justify-center cursor-default"
                  :title="$t('fileRow.removed')">r</span>
            <span v-else-if="file.status === 'deleted'"
                  class="shrink-0 w-4 h-4 rounded-full bg-danger-500/15 text-danger-500 text-[10px] font-bold flex items-center justify-center cursor-default"
                  :title="$t('fileRow.deleted')">d</span>
            <!-- File name: link for uploaded, strikethrough for deleted/removed, plain otherwise -->
            <a v-else-if="file.status === 'uploaded'"
               :href="getFileURL(upload.id, file.id, file.fileName, upload.stream)"
               class="text-surface-300 hover:text-accent-400 transition-colors truncate">
              {{ file.fileName }}
            </a>
            <span v-else-if="file.status === 'removed' || file.status === 'deleted'"
                  class="text-surface-500 truncate line-through">
              {{ file.fileName }}
            </span>
            <span v-else class="text-surface-500 truncate">
              {{ file.fileName }}
            </span>
          </div>
          <span class="text-surface-500 shrink-0">{{ humanReadableSize(file.fileSize) }}</span>
        </div>
        <p v-if="!upload.files || upload.files.length === 0"
           class="text-surface-500 italic">{{ $t('uploadCard.noFiles') }}</p>
      </div>

      <!-- Actions -->
      <div class="sm:w-20 flex sm:flex-col items-center sm:justify-center gap-2">
        <button @click="emit('delete', upload)"
                class="text-xs text-red-400 hover:text-red-300 border border-red-500/30
                       rounded-lg px-3 py-1.5 hover:bg-red-500/10 transition-colors">
          {{ $t('uploadCard.remove') }}
        </button>
      </div>
    </div>
  </div>
</template>
