<script setup>
import { ref, computed } from 'vue'
import { formatDate } from '../utils.js'
import { getArchiveURL, getAdminURL } from '../api.js'
import { showError } from '../notification.js'
import CopyButton from './CopyButton.vue'
import UploadBadges from './UploadBadges.vue'

const passphrase = defineModel('passphrase', { type: String, default: null })

const props = defineProps({
  upload: { type: Object, required: true },
})

const emit = defineEmits(['delete-upload', 'add-files', 'show-qr', 'edit-passphrase', 'show-events'])

const expirationText = computed(() => {
  if (!props.upload.expireAt) return 'Never expires'
  const d = new Date(props.upload.expireAt)
  const now = new Date()
  if (d <= now) return 'Expired'
  const diffMs = d - now
  const diffDays = Math.floor(diffMs / 86400000)
  const diffHours = Math.floor((diffMs % 86400000) / 3600000)
  if (diffDays > 0) return `Expires in ${diffDays}d ${diffHours}h`
  if (diffHours > 0) return `Expires in ${diffHours}h`
  const diffMins = Math.max(1, Math.ceil((diffMs % 3600000) / 60000))
  return `Expires in ${diffMins}m`
})

const archiveUrl = computed(() => getArchiveURL(props.upload.id))

const adminUrl = computed(() => {
  if (!props.upload.admin || !props.upload.uploadToken) return null
  return getAdminURL(props.upload.id, props.upload.uploadToken)
})

// Share URL (download page without upload token)
const includePassphrase = ref(false)
const shareUrl = computed(() => {
  let url = `${window.location.origin}${window.location.pathname}#/?id=${props.upload.id}`
  if (includePassphrase.value && passphrase.value) {
    url += `&key=${encodeURIComponent(passphrase.value)}`
  }
  return url
})

// Native share support (mobile + Chrome/Edge desktop)
const canNativeShare = typeof navigator !== 'undefined' && !!navigator.share

const shareSuccess = ref(false)
let shareTimer = null

async function nativeShare() {
  try {
    await navigator.share({ title: 'Plik Upload', url: shareUrl.value })
    shareSuccess.value = true
    clearTimeout(shareTimer)
    shareTimer = setTimeout(() => { shareSuccess.value = false }, 2000)
  } catch (err) {
    // User cancelled or share failed — ignore
    if (err.name !== 'AbortError') {
      showError('Share failed')
    }
  }
}

// Admins can delete upload, or if upload is marked as removable
const canDeleteUpload = computed(() => props.upload.admin || props.upload.removable)
const canAddFiles = computed(() => props.upload.admin && !props.upload.stream)
</script>

<template>
  <aside class="w-full md:w-80 md:shrink-0 p-4 space-y-3 animate-slide-in">
    <!-- Upload Info -->
    <div class="sidebar-section">
      <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider mb-2">Upload Info</h3>

      <div v-if="expirationText === 'Never expires'" class="text-sm text-surface-300">
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M18.178 8c5.096 0 5.096 8 0 8-5.095 0-7.133-8-12.739-8-4.585 0-4.585 8 0 8 5.606 0 7.644-8 12.74-8z" />
          </svg>
          <span class="text-emerald-400">Never expires</span>
        </div>
      </div>

      <div v-else-if="expirationText" class="text-sm text-surface-300">
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4 text-warning-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {{ expirationText }}
        </div>
        <p class="text-xs text-surface-500 mt-1">{{ formatDate(upload.expireAt) }}</p>
      </div>

      <!-- Upload options badges -->
      <UploadBadges :upload="upload" class="mt-2" />
    </div>

    <!-- Share -->
    <div class="sidebar-section">
      <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider mb-2">Share</h3>

      <!-- Passphrase display (E2EE only) -->
      <div v-if="upload.e2ee" class="mb-3">
        <label class="text-xs text-surface-500 mb-1 block">Passphrase</label>
        <div class="flex items-center gap-2 p-2 rounded bg-surface-800/50 border border-surface-700 min-w-0 overflow-hidden">
          <span v-if="passphrase" class="text-xs text-accent-400 font-mono truncate flex-1">{{ passphrase }}</span>
          <span v-else class="text-xs text-surface-500 italic flex-1">Not set</span>
          <button class="text-surface-400 hover:text-accent-400 transition-colors shrink-0"
                  title="Edit passphrase"
                  @click="emit('edit-passphrase')">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
          <CopyButton v-if="passphrase" :text="passphrase" size="sm" />
        </div>
        <label v-if="passphrase" class="flex items-center justify-between py-1.5 mt-2 cursor-pointer group">
          <span class="text-xs text-surface-400 group-hover:text-surface-200 transition-colors">Include passphrase in link</span>
          <button type="button"
                  class="toggle-switch scale-75"
                  :data-active="includePassphrase"
                  @click="includePassphrase = !includePassphrase">
            <span class="toggle-dot" />
          </button>
        </label>
      </div>

      <button v-if="canNativeShare"
              class="btn-primary w-full"
              :class="shareSuccess ? 'bg-success-500/20 text-success-500' : ''"
              @click="nativeShare">
        <svg v-if="shareSuccess" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
        <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
        </svg>
        {{ shareSuccess ? 'Shared!' : 'Share' }}
      </button>
      <div v-else class="flex items-center gap-2 p-2 rounded bg-surface-800/50 border border-surface-700 min-w-0 overflow-hidden">
        <span class="text-xs text-surface-300 truncate flex-1">{{ shareUrl }}</span>
        <CopyButton :text="shareUrl" size="sm" />
      </div>
    </div>

    <!-- Admin URL (only for admins) -->
    <div v-if="adminUrl" class="sidebar-section">
      <div class="flex items-center gap-1 mb-2">
        <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider">Admin URL</h3>
        <div class="group relative">
          <svg class="w-3.5 h-3.5 text-surface-500 cursor-help" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div class="absolute left-0 -top-2 -translate-y-full hidden group-hover:block w-56 p-2 text-xs bg-surface-900 text-surface-200 rounded shadow-lg z-10">
            Share this URL with others to allow them to add files to this upload
          </div>
        </div>
      </div>
      <div class="flex items-center gap-2 p-2 rounded bg-surface-900/50 border border-surface-700 min-w-0 overflow-hidden">
        <span class="text-xs text-surface-300 truncate flex-1">{{ adminUrl }}</span>
        <CopyButton :text="adminUrl" size="sm" />
      </div>
    </div>

    <!-- Actions -->
    <div class="sidebar-section space-y-2">
      <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider mb-2">Actions</h3>

      <!-- Zip archive -->
      <a v-if="upload.files?.length && !upload.stream && !upload.e2ee"
         :href="archiveUrl"
         class="btn-primary w-full">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
        </svg>
        Zip Archive
      </a>

      <!-- QR Code -->
      <button class="btn-primary w-full" @click="emit('show-qr')">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM17 14v3h3M14 17h3v3" />
        </svg>
        QR Code
      </button>

      <!-- Add files -->
      <button v-if="canAddFiles"
              class="btn-primary w-full"
              @click="emit('add-files')">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Add Files
      </button>

      <!-- Events (admin only) -->
      <button v-if="upload.admin"
              class="btn-primary w-full"
              @click="emit('show-events')">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        Events
      </button>

      <!-- Delete upload -->
      <button v-if="canDeleteUpload"
              class="btn-danger w-full"
              @click="emit('delete-upload')">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
        Delete Upload
      </button>
    </div>
  </aside>
</template>
