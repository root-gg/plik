<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDate } from '../utils.js'
import { getArchiveURL, getAdminURL } from '../api.js'
import CopyButton from './CopyButton.vue'
import UploadBadges from './UploadBadges.vue'

const passphrase = defineModel('passphrase', { type: String, default: null })

const props = defineProps({
  upload: { type: Object, required: true },
})

const emit = defineEmits(['delete-upload', 'add-files', 'show-qr', 'edit-passphrase', 'error'])

const { t } = useI18n()

const expirationText = computed(() => {
  if (!props.upload.expireAt) return t('downloadSidebar.neverExpires')
  const d = new Date(props.upload.expireAt)
  const now = new Date()
  if (d <= now) return t('downloadSidebar.expired')
  const diffMs = d - now
  const diffDays = Math.floor(diffMs / 86400000)
  const diffHours = Math.floor((diffMs % 86400000) / 3600000)
  if (diffDays > 0) return t('downloadSidebar.expiresInDaysHours', { days: diffDays, hours: diffHours })
  if (diffHours > 0) return t('downloadSidebar.expiresInHours', { hours: diffHours })
  const diffMins = Math.max(1, Math.ceil((diffMs % 3600000) / 60000))
  return t('downloadSidebar.expiresInMinutes', { minutes: diffMins })
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
      emit('error', t('downloadSidebar.shareFailed'))
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
      <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider mb-2">{{ $t('downloadSidebar.uploadInfo') }}</h3>

      <div v-if="!upload.expireAt" class="text-sm text-surface-300">
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M18.178 8c5.096 0 5.096 8 0 8-5.095 0-7.133-8-12.739-8-4.585 0-4.585 8 0 8 5.606 0 7.644-8 12.74-8z" />
          </svg>
          <span class="text-emerald-400">{{ $t('downloadSidebar.neverExpires') }}</span>
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
      <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider mb-2">{{ $t('downloadSidebar.share') }}</h3>

      <!-- Passphrase display (E2EE only) -->
      <div v-if="upload.e2ee" class="mb-3">
        <label class="text-xs text-surface-500 mb-1 block">{{ $t('downloadSidebar.passphrase') }}</label>
        <div class="flex items-center gap-2 p-2 rounded bg-surface-800/50 border border-surface-700 min-w-0 overflow-hidden">
          <span v-if="passphrase" class="text-xs text-accent-400 font-mono truncate flex-1">{{ passphrase }}</span>
          <span v-else class="text-xs text-surface-500 italic flex-1">{{ $t('downloadSidebar.notSet') }}</span>
          <button class="text-surface-400 hover:text-accent-400 transition-colors shrink-0"
                  :title="$t('downloadSidebar.editPassphrase')"
                  @click="emit('edit-passphrase')">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
          <CopyButton v-if="passphrase" :text="passphrase" size="sm" />
        </div>
        <label v-if="passphrase" class="flex items-center justify-between py-1.5 mt-2 cursor-pointer group">
          <span class="text-xs text-surface-400 group-hover:text-surface-200 transition-colors">{{ $t('downloadSidebar.includePassphraseInLink') }}</span>
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
        {{ shareSuccess ? $t('downloadSidebar.shared') : $t('downloadSidebar.share') }}
      </button>
      <div v-else class="flex items-center gap-2 p-2 rounded bg-surface-800/50 border border-surface-700 min-w-0 overflow-hidden">
        <span class="text-xs text-surface-300 truncate flex-1">{{ shareUrl }}</span>
        <CopyButton :text="shareUrl" size="sm" />
      </div>
    </div>

    <!-- Admin URL (only for admins) -->
    <div v-if="adminUrl" class="sidebar-section">
      <div class="flex items-center gap-1 mb-2">
        <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider">{{ $t('downloadSidebar.adminUrl') }}</h3>
        <div class="group relative">
          <svg class="w-3.5 h-3.5 text-surface-500 cursor-help" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div class="absolute left-0 -top-2 -translate-y-full hidden group-hover:block w-56 p-2 text-xs bg-surface-900 text-surface-200 rounded shadow-lg z-10">
            {{ $t('downloadSidebar.adminUrlHelp') }}
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
      <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider mb-2">{{ $t('downloadSidebar.actions') }}</h3>

      <!-- Zip archive -->
      <a v-if="upload.files?.length && !upload.stream && !upload.e2ee"
         :href="archiveUrl"
         class="btn-primary w-full">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
        </svg>
        {{ $t('downloadSidebar.zipArchive') }}
      </a>

      <!-- QR Code -->
      <button class="btn-primary w-full" @click="emit('show-qr')">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM17 14v3h3M14 17h3v3" />
        </svg>
        {{ $t('common.qrCode') }}
      </button>

      <!-- Add files -->
      <button v-if="canAddFiles"
              class="btn-primary w-full"
              @click="emit('add-files')">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        {{ $t('downloadSidebar.addFiles') }}
      </button>

      <!-- Delete upload -->
      <button v-if="canDeleteUpload"
              class="btn-danger w-full"
              @click="emit('delete-upload')">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
        {{ $t('downloadSidebar.deleteUpload') }}
      </button>
    </div>
  </aside>
</template>
