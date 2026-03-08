<script setup>
import { ref, computed } from 'vue'
import { humanReadableSize, isViewableFile as checkIsViewableFile } from '../utils.js'
import { getFileURL } from '../api.js'
import CopyButton from './CopyButton.vue'

const props = defineProps({
  file: { type: Object, required: true },
  uploadId: { type: String, default: '' },
  mode: { type: String, default: 'upload' }, // 'upload' | 'uploading' | 'download'
  canRemove: { type: Boolean, default: false },
  isStream: { type: Boolean, default: false },
  isOneShot: { type: Boolean, default: false },
  isE2ee: { type: Boolean, default: false },
})

const emit = defineEmits(['remove', 'update-name', 'show-qr', 'view', 'cancel', 'retry', 'decrypt-download'])

// For streaming uploads, files in 'uploading' status have a valid download URL
// (the server streams from uploader to downloader). Files in 'missing' status
// are not yet being uploaded and will 404 if downloaded.
const isDownloadable = computed(() =>
  props.file.status === 'uploaded' ||
  (props.isStream && props.file.status === 'uploading')
)

const isDeleted = computed(() =>
  props.file.status === 'removed' || props.file.status === 'deleted'
)

const isViewable = computed(() => {
  if (props.file.status !== 'uploaded') return false
  return checkIsViewableFile(props.file)
})

const showDetails = ref(false)

function onNameInput(e) {
  let name = e.target.textContent.trim()
  if (name.length > 1024) {
    name = name.slice(0, 1024)
    e.target.textContent = name
  }
  emit('update-name', name)
}

function onNameKeydown(e) {
  // Allow control keys, but block character input if at limit
  if (e.target.textContent.length >= 1024 && !e.ctrlKey && !e.metaKey &&
      e.key.length === 1 && !['Backspace', 'Delete'].includes(e.key)) {
    e.preventDefault()
  }
}

function onNamePaste(e) {
  e.stopPropagation()
  // Handle paste manually to enforce limit
  e.preventDefault()
  const text = e.clipboardData?.getData('text/plain') || ''
  const el = e.target
  const current = el.textContent || ''
  const sel = window.getSelection()
  const range = sel.rangeCount ? sel.getRangeAt(0) : null

  // Calculate how many chars we can insert
  let selectedLen = 0
  if (range && el.contains(range.startContainer)) {
    selectedLen = range.toString().length
  }
  const available = 1024 - current.length + selectedLen
  if (available <= 0) return

  const insert = text.replace(/\n/g, '').slice(0, available)
  if (range) {
    range.deleteContents()
    range.insertNode(document.createTextNode(insert))
    range.collapse(false)
    sel.removeAllRanges()
    sel.addRange(range)
  }
}

function fileUrl() {
  if (!props.uploadId || !props.file.id) return ''
  return getFileURL(props.uploadId, props.file.id, props.file.fileName, props.isStream)
}
</script>

<template>
  <div class="file-row animate-fade-in flex-wrap" :class="{ 'opacity-50': isDeleted }">
    <div class="flex flex-wrap items-center gap-2 md:gap-3 flex-1 min-w-0">
      <!-- File icon -->
      <div class="shrink-0">
        <svg class="w-5 h-5 text-surface-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
        </svg>
      </div>

      <!-- File name -->
      <div class="flex-1 min-w-0">
        <!-- Editable name (upload mode) -->
        <div v-if="mode === 'upload'" class="inline-flex items-center gap-1 min-w-0 w-full">
          <div class="text-sm text-surface-200 cursor-text outline-none
                      overflow-hidden text-ellipsis whitespace-nowrap
                      focus:overflow-x-auto focus:text-clip focus:whitespace-normal
                      hover:text-surface-200 focus:ring-1 focus:ring-accent-500/50 rounded px-1 -mx-1"
               contenteditable="true"
               @blur="onNameInput"
               @keydown="onNameKeydown"
               @keydown.enter.prevent="$event.target.blur()"
               @paste="onNamePaste">
            {{ file.fileName }}
          </div>
          <svg class="w-3 h-3 text-surface-500 shrink-0"
               fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
          </svg>
        </div>

        <!-- Download mode: caret toggles details, name is a link -->
        <div v-else-if="mode === 'download'" class="inline-flex items-center gap-1 min-w-0 w-full">
          <button class="shrink-0 p-0.5 text-surface-500 hover:text-surface-300 transition-colors"
                  :title="$t('fileRow.toggleDetails')"
                  @click="showDetails = !showDetails">
            <svg class="w-3 h-3 transition-transform duration-200"
                 :class="showDetails ? 'rotate-90' : ''"
                 fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
          <span v-if="isDeleted" class="text-sm text-surface-500 truncate line-through">
            {{ file.fileName }}
          </span>
          <a v-else-if="isDownloadable && !isE2ee"
             :href="fileUrl()"
             class="text-sm text-surface-200 hover:text-accent-400 transition-colors truncate"
             target="_blank"
             rel="noopener noreferrer">
            {{ file.fileName }}
          </a>
          <button v-else-if="isDownloadable && isE2ee"
                  class="text-sm text-surface-200 hover:text-accent-400 transition-colors truncate text-left"
                  @click="emit('decrypt-download', file)">
            {{ file.fileName }}
          </button>
          <span v-else class="text-sm text-surface-200 truncate">
            {{ file.fileName }}
          </span>
        </div>

        <!-- Static name -->
        <div v-else class="text-sm text-surface-200 truncate">
          {{ file.fileName }}
        </div>

        <!-- Progress bar (uploading mode) -->
        <div v-if="mode === 'uploading' && file.status === 'uploading'" class="mt-1.5">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: (file.progress || 0) + '%' }" />
          </div>
          <span class="text-xs text-surface-400 mt-0.5">{{ file.progress || 0 }}%</span>
        </div>

        <!-- Upload complete indicator -->
        <div v-if="mode === 'uploading' && file.status === 'uploaded'" class="mt-1">
          <span class="text-xs text-success-500 flex items-center gap-1">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
            {{ $t('fileRow.uploaded') }}
          </span>
        </div>

        <!-- Upload error indicator -->
        <div v-if="mode === 'uploading' && file.status === 'error'" class="mt-1 flex items-center gap-2">
          <span class="text-xs text-danger-500 flex items-center gap-1">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            {{ file.error || $t('fileRow.uploadFailed') }}
          </span>
          <button class="text-xs text-accent-400 hover:text-accent-300 transition-colors flex items-center gap-0.5"
                  :title="$t('fileRow.retry')"
                  @click="emit('retry', file)">
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {{ $t('fileRow.retry') }}
          </button>
        </div>
      </div>

      <!-- File size -->
      <div class="text-sm text-surface-400 shrink-0 tabular-nums">
        {{ humanReadableSize(file.fileSize || file.size) }}
      </div>

      <!-- Status badge for non-uploaded files (download mode) -->
      <span v-if="mode === 'download' && isDeleted"
            class="text-xs text-danger-500 bg-danger-500/10 px-2 py-0.5 rounded-full shrink-0">
        {{ $t('fileRow.removed') }}
      </span>
      <span v-else-if="mode === 'download' && file.status === 'missing'"
            class="text-xs text-warning-500 bg-warning-500/10 px-2 py-0.5 rounded-full shrink-0">
        {{ $t('fileRow.waitingForUpload') }}
      </span>
      <span v-else-if="mode === 'download' && file.status === 'uploading'"
            class="text-xs text-accent-400 bg-accent-500/10 px-2 py-0.5 rounded-full shrink-0 inline-flex items-center gap-1">
        <div class="animate-spin rounded-full h-3 w-3 border border-accent-400 border-t-transparent" />
        {{ $t('fileRow.uploading') }}
      </span>

      <!-- Actions -->
      <div v-if="!isDeleted" class="flex items-center gap-1 shrink-0">

        <!-- QR Code button (download mode) -->
        <button v-if="mode === 'download' && isDownloadable"
                class="btn bg-surface-700/50 text-surface-400 hover:text-surface-100 px-2 py-1.5 text-xs"
                :title="$t('fileRow.showQrCode')"
                @click="emit('show-qr', file)">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM17 14v3h3M14 17h3v3" />
          </svg>
        </button>

        <!-- Copy link (download mode) -->
        <CopyButton v-if="mode === 'download' && isDownloadable"
                    :text="fileUrl()" />

        <!-- View button (download mode, viewable files) -->
        <button v-if="mode === 'download' && file.status === 'uploaded' && isViewable && !isOneShot && !isStream"
                class="btn bg-accent-500/10 text-accent-400 hover:bg-accent-500/20 px-2 py-1.5 text-xs"
                :title="$t('fileRow.viewFileContent')"
                @click="emit('view', file)">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
          </svg>
          <span class="hidden md:inline">{{ $t('common.view') }}</span>
        </button>

        <!-- Download button (download mode) -->
        <a v-if="mode === 'download' && isDownloadable && !isE2ee"
           :href="fileUrl() + '?dl=1'"
           class="btn bg-success-500/10 text-success-500 hover:bg-success-500/20 px-2 md:px-3 py-1.5 text-xs"
           target="_blank"
           rel="noopener noreferrer">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          <span class="hidden md:inline">{{ $t('common.download') }}</span>
        </a>

        <!-- Decrypt + Download button (E2EE mode) -->
        <button v-if="mode === 'download' && isDownloadable && isE2ee"
                class="btn bg-accent-500/10 text-accent-400 hover:bg-accent-500/20 px-2 md:px-3 py-1.5 text-xs"
                :title="$t('fileRow.decryptAndDownload')"
                @click="emit('decrypt-download', file)">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          <span class="hidden md:inline">{{ $t('common.decrypt') }}</span>
        </button>

        <!-- Cancel button (uploading mode — for in-progress or queued files) -->
        <button v-if="mode === 'uploading' && (file.status === 'uploading' || file.status === 'toUpload')"
                class="btn bg-danger-500/10 text-danger-500 hover:bg-danger-500/20 px-2 py-1.5 text-xs"
                :title="$t('fileRow.cancelUpload')"
                @click="emit('cancel', file)">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>

        <!-- Remove button (error state — dismiss failed upload) -->
        <button v-if="mode === 'uploading' && file.status === 'error'"
                class="btn bg-danger-500/10 text-danger-500 hover:bg-danger-500/20 px-2 py-1.5 text-xs"
                :title="$t('fileRow.dismiss')"
                @click="emit('remove', file)">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>

        <!-- Remove button -->
        <button v-if="(mode === 'upload' || canRemove) && file.status !== 'uploading'"
                class="btn bg-danger-500/10 text-danger-500 hover:bg-danger-500/20 px-2 py-1.5 text-xs"
                :title="$t('fileRow.removeFile')"
                @click="emit('remove', file)">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Expanded file details -->
    <div v-if="showDetails && mode === 'download'"
         class="w-full mt-2 pt-2 border-t border-surface-700/50 text-xs text-surface-400 space-y-1 animate-fade-in">
      <div v-if="file.fileType" class="flex gap-2">
        <span class="text-surface-500 w-14">{{ $t('fileRow.type') }}</span>
        <span class="text-surface-300">{{ file.fileType }}</span>
      </div>
      <div v-if="file.fileMd5" class="flex gap-2">
        <span class="text-surface-500 w-14">{{ $t('fileRow.md5') }}</span>
        <span class="text-surface-300 font-mono">{{ file.fileMd5 }}</span>
      </div>
      <div v-if="file.createdAt" class="flex gap-2">
        <span class="text-surface-500 w-14">{{ $t('fileRow.created') }}</span>
        <span class="text-surface-300">{{ new Date(file.createdAt).toLocaleString() }}</span>
      </div>
    </div>
  </div>
</template>
