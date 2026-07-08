<script setup>
import { ref, onMounted, onUnmounted, computed, nextTick, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getUpload, removeUpload, removeFile as apiRemoveFile, uploadFile, getFileURL } from '../api.js'
import { generateRef, isMarkdownFile, isImageFile, isVideoFile, isAudioFile, isViewableFile, charsetFromContentType } from '../utils.js'
import { fetchAndDecrypt } from '../crypto.js'
import { getToken, setToken } from '../tokenStore.js'
import { config } from '../config.js'
import { consumePendingFiles } from '../pendingUploadStore.js'
import { renderMarkdown, initMermaidInElement } from '../markdown.js'
import DownloadSidebar from '../components/DownloadSidebar.vue'
import MarkdownTabs from '../components/MarkdownTabs.vue'
import FileRow from '../components/FileRow.vue'
import CopyButton from '../components/CopyButton.vue'
import QrCodeDialog from '../components/QrCodeDialog.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import ErrorState from '../components/ErrorState.vue'
import ErrorBanner from '../components/ErrorBanner.vue'
import { defineAsyncComponent } from 'vue'
const CodeEditor = defineAsyncComponent(() => import('../components/CodeEditor.vue'))

const props = defineProps({
  id: { type: String, required: true },
})

const router = useRouter()
const route = useRoute()
const { t: $t } = useI18n()

const upload = ref(null)
const loading = ref(true)
const error = ref(null)
const uploadError = ref(null)
const fileInput = ref(null)
const commentsRef = ref(null)

// Render mermaid diagrams in upload comments after the comment block mounts.
// commentsRef is inside v-if="upload.comments" — watching the ref directly
// ensures we run after Vue has mounted the block and injected v-html content.
watch(commentsRef, async (el) => {
  if (!el) return
  await nextTick()
  initMermaidInElement(el)
})

// Staged files pending upload
const pendingFiles = ref([])
const isAddingFiles = ref(false)

// BasicAuth for password-protected uploads (passed from UploadView via pending store)
let pendingBasicAuth = null

// Track whether uploads were cancelled
let uploadsCancelled = false

// Basic auth credentials (transient — only available right after upload via pending store)
const basicAuthLogin = ref(null)
const basicAuthPassword = ref(null)

// E2EE passphrase (extracted from URL fragment or pending store, or prompted)
const e2eePassphrase = ref(null)
const showPassphraseModal = ref(false)
const passphraseInput = ref('')
const isDecrypting = ref(false)

// QR code dialog
const showQr = ref(false)
const qrTitle = ref('')
const qrUrl = ref('')

// Confirmation dialog state
const confirmDialog = ref(null)

// File viewer state
const viewingFile = ref(null)
const viewingContent = ref('')
const viewingLoading = ref(false)
const viewingError = ref(null)
const lastAutoViewedId = ref(null)
const viewerTab = ref('code') // 'code' | 'preview'

// Media time tracking
const initialMediaTime = ref(null)
const mediaCurrentTime = ref(0) // reactive tracker for share URL
const isDeepLinked = ref(false) // true when viewer was opened from URL params (for autoplay)
const isViewingMarkdown = computed(() => viewingFile.value && isMarkdownFile(viewingFile.value))
const isViewingImage = computed(() => viewingFile.value && isImageFile(viewingFile.value))
const viewingImageUrl = computed(() => {
  if (!isViewingImage.value) return ''
  return getFileURL(props.id, viewingFile.value.id, viewingFile.value.fileName, upload.value?.stream)
})
const isViewingVideo = computed(() => viewingFile.value && isVideoFile(viewingFile.value))
const viewingVideoUrl = computed(() => {
  if (!isViewingVideo.value) return ''
  return getFileURL(props.id, viewingFile.value.id, viewingFile.value.fileName, upload.value?.stream)
})
const isViewingAudio = computed(() => viewingFile.value && isAudioFile(viewingFile.value))
const viewingAudioUrl = computed(() => {
  if (!isViewingAudio.value) return ''
  return getFileURL(props.id, viewingFile.value.id, viewingFile.value.fileName, upload.value?.stream)
})
const renderedFileContent = computed(() => {
  if (!isViewingMarkdown.value || viewerTab.value !== 'preview') return ''
  if (!viewingContent.value) return ''
  return renderMarkdown(viewingContent.value)
})

async function viewFile(file) {
  // If already viewing this file, close it
  if (viewingFile.value?.id === file.id) {
    closeViewer()
    return
  }
  viewingFile.value = file
  viewerTab.value = isMarkdownFile(file) ? 'preview' : 'code'
  viewingContent.value = ''
  viewingLoading.value = false
  viewingError.value = null
  mediaCurrentTime.value = 0 // reset so shareAtTimeUrl doesn't show stale time
  mediaPreviewCounted = false // new view — allow one media-load stats refresh

  // Sync file ID to URL
  syncViewerToUrl()

  // Media files render directly from the server URL — no content fetch needed;
  // the browser's own GET is the counted download. The counter refresh
  // happens when the element reports the content arrived (onMediaPreviewLoaded).
  if (isImageFile(file) || isVideoFile(file) || isAudioFile(file)) {
    nextTick(() => {
      document.getElementById('file-viewer-panel')?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    })
    return
  }

  viewingLoading.value = true
  try {
    const url = getFileURL(props.id, file.id, file.fileName, upload.value?.stream)
    const resp = await fetch(url, { credentials: 'same-origin' })
    if (!resp.ok) {
      const text = await resp.text().catch(() => '')
      throw new Error(text || `Failed to load file (HTTP ${resp.status})`)
    }
    // Decode using the charset from the Content-Type header (e.g. "text/plain; charset=utf-16be").
    // resp.text() always assumes UTF-8, which garbles non-UTF-8 files.
    const encoding = charsetFromContentType(resp.headers.get('Content-Type'))
    const buf = await resp.arrayBuffer()
    const text = new TextDecoder(encoding).decode(buf)
    viewingContent.value = text
    // This request just counted as a download — refresh the counters
    // shown in the sidebar/file rows. Fire-and-forget: it must not delay or
    // block the viewer from rendering.
    refreshDownloadStats()
  } catch (err) {
    viewingError.value = err.message || $t('downloadView.failedToLoadUpload')
  } finally {
    viewingLoading.value = false
    nextTick(() => {
      document.getElementById('file-viewer-panel')?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    })
  }
}

function closeViewer() {
  viewingFile.value = null
  viewingContent.value = ''
  viewingError.value = null
  viewerTab.value = 'code'
  initialMediaTime.value = null
  mediaCurrentTime.value = 0
  isDeepLinked.value = false
  syncViewerToUrl()
}

// --- URL ↔ Viewer sync ---

// Update URL query params to reflect current viewer state
function syncViewerToUrl() {
  const query = { ...route.query }
  // Strip t= when not viewing media (keep it for video/audio deep links)
  if (!viewingFile.value || (!isVideoFile(viewingFile.value) && !isAudioFile(viewingFile.value))) {
    delete query.t
  }
  if (viewingFile.value) {
    query.file = viewingFile.value.id
  } else {
    delete query.file
  }
  router.replace({ query })
}

// Handle timeupdate events from video/audio
function onMediaTimeUpdate(event) {
  mediaCurrentTime.value = event.target.currentTime
}

// Counter refresh for media previews (img/video/audio). Unlike the text
// path there is no fetch promise to await — the browser issues the GET itself
// — so hook the element's load event instead (@load on <img>, loadedmetadata
// on <video>/<audio>, which is guaranteed to fire with preload="metadata").
// Media elements can emit further load-ish events within one view (seeks
// trigger range requests, metadata can re-fire), so refresh at most once per
// view; viewFile() resets the guard for each new view.
let mediaPreviewCounted = false
function onMediaPreviewLoaded() {
  if (mediaPreviewCounted) return
  mediaPreviewCounted = true
  refreshDownloadStats()
}

// Seek media to initial time once metadata is loaded
function onMediaLoadedMetadata(event) {
  if (initialMediaTime.value != null && initialMediaTime.value > 0) {
    event.target.currentTime = initialMediaTime.value
    initialMediaTime.value = null
  }
  // Autoplay when arriving from a deep link
  if (isDeepLinked.value) {
    isDeepLinked.value = false
    event.target.muted = true
    event.target.play().then(() => {
      event.target.muted = false
    }).catch(() => {})
  }
  // The metadata GET that fired this event was counted server-side
  onMediaPreviewLoaded()
}

// Build a shareable URL with file= and t= for current media position
const shareAtTimeUrl = computed(() => {
  if (!viewingFile.value) return ''
  const t = Math.floor(mediaCurrentTime.value)
  const query = { id: props.id, file: viewingFile.value.id }
  if (t > 0) query.t = String(t)
  const resolved = router.resolve({ query })
  return window.location.origin + resolved.href
})

// Viewer navigation — prev/next through viewable files
const viewableFiles = computed(() => {
  if (upload.value?.oneShot || upload.value?.stream) return []
  return activeFiles.value.filter(f => f.status === 'uploaded' && isViewableFile(f))
})
const viewerIndex = computed(() => {
  if (!viewingFile.value) return -1
  return viewableFiles.value.findIndex(f => f.id === viewingFile.value.id)
})
const hasPrev = computed(() => viewerIndex.value > 0)
const hasNext = computed(() => viewerIndex.value >= 0 && viewerIndex.value < viewableFiles.value.length - 1)

function viewPrev() {
  if (hasPrev.value) viewFile(viewableFiles.value[viewerIndex.value - 1])
}
function viewNext() {
  if (hasNext.value) viewFile(viewableFiles.value[viewerIndex.value + 1])
}

function onViewerKeydown(e) {
  if (!viewingFile.value) return
  // Don't intercept when user is typing in an input/textarea
  const tag = e.target.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || e.target.isContentEditable) return
  if (e.key === 'ArrowLeft') { e.preventDefault(); viewPrev() }
  else if (e.key === 'ArrowRight') { e.preventDefault(); viewNext() }
  else if (e.key === 'Escape') { e.preventDefault(); closeViewer() }
}

// Active files for the top panel
// During uploads, only show files the user can interact with:
//  - Non-streaming: only 'uploaded' (file complete on server)
//  - Streaming: 'uploading' + 'uploaded' (download works via live stream)
// When not uploading (e.g. friend viewing), show all files including removed/deleted
const activeFiles = computed(() => {
  if (!upload.value?.files) return []
  return upload.value.files.filter(f => {
    if (f.status === 'removed' || f.status === 'deleted') return true
    if (isAddingFiles.value) {
      if (upload.value.stream) {
        return f.status === 'uploading' || f.status === 'uploaded'
      } else {
        return f.status === 'uploaded'
      }
    }
    return true
  })
})

// Total non-removed files (for "X/Y files uploaded" display during uploads)
const totalFiles = computed(() => {
  if (!upload.value?.files) return 0
  return upload.value.files.filter(f => f.status !== 'removed' && f.status !== 'deleted').length
})

// Count of fully uploaded files (progress numerator during active uploads)
const uploadedCount = computed(() => {
  if (!upload.value?.files) return 0
  return upload.value.files.filter(f => f.status === 'uploaded').length
})

// Upload token from in-memory store (set after upload or from admin URL)
const uploadToken = computed(() => getToken(props.id))

// Check if user is upload admin
const isAdmin = computed(() => upload.value?.admin || false)
const canRemoveFiles = computed(() =>
  upload.value?.removable || upload.value?.admin
)

const streamTimeoutLabel = computed(() => {
  const s = config.streamTimeout
  if (s <= 0) return ''
  if (s >= 86400) { const n = Math.round(s / 86400); return $t('common.timeDay', { n }, n) }
  if (s >= 3600) { const n = Math.round(s / 3600); return $t('common.timeHour', { n }, n) }
  if (s >= 60) { const n = Math.round(s / 60); return $t('common.timeMinute', { n }, n) }
  return $t('common.timeSecond', { n: s }, s)
})

async function fetchUpload() {
  loading.value = true
  error.value = null
  try {
    upload.value = await getUpload(props.id, uploadToken.value)
  } catch (err) {
    error.value = err.status
      ? `${err.message} (HTTP ${err.status})`
      : (err.message || $t('downloadView.failedToLoadUpload'))
  } finally {
    loading.value = false
  }
}

// Previews count as downloads. The initial fetchUpload() (on mount, or at
// the end of the upload pool) can resolve before the server has recorded a
// just-triggered preview's download, so the sidebar/file-row counters go
// stale until *something* refetches. Call this right after a preview's
// content request resolves — server-side recording happens synchronously
// within that request, so by the time it resolves the count is already
// incremented and this refetch reads the up-to-date value.
// A full fetchUpload() would flash the page-level loading spinner and is
// overkill for a counter refresh, so this only patches the download fields
// in place, leaving viewer/pending-upload state untouched.
async function refreshDownloadStats() {
  if (!upload.value) return
  try {
    const fresh = await getUpload(props.id, uploadToken.value)
    if (!upload.value || fresh.id !== upload.value.id) return
    upload.value.downloadCount = fresh.downloadCount
    upload.value.lastDownloadedAt = fresh.lastDownloadedAt
    for (const freshFile of fresh.files || []) {
      const localFile = upload.value.files?.find(f => f.id === freshFile.id)
      if (localFile) {
        localFile.downloadCount = freshFile.downloadCount
        localFile.lastDownloadedAt = freshFile.lastDownloadedAt
      }
    }
  } catch {
    // Best-effort — leave the counters as they were rather than surface an error.
  }
}

async function deleteUpload() {
  confirmDialog.value = {
    title: $t('downloadView.deleteUploadTitle'),
    message: $t('downloadView.deleteUploadMessage'),
    confirmText: $t('common.delete'),
    onConfirm: async () => {
      try {
        await removeUpload(props.id, uploadToken.value)
        // Redirect to home page
        router.push({ path: '/' })
      } catch (err) {
        error.value = err.message || $t('downloadView.failedToDeleteUpload')
        confirmDialog.value = null
      } finally {
        confirmDialog.value = null
      }
    }
  }
}

async function deleteFile(file) {
  confirmDialog.value = {
    title: $t('downloadView.deleteFileTitle'),
    message: $t('downloadView.deleteFileMessage', { name: file.fileName }),
    confirmText: $t('common.delete'),
    onConfirm: async () => {
      try {
        await apiRemoveFile(
          { id: props.id, stream: upload.value.stream, uploadToken: uploadToken.value },
          file,
        )
        // Close viewer if the deleted file was being viewed
        if (viewingFile.value?.id === file.id) {
          closeViewer()
        }
        await fetchUpload()
      } catch (err) {
        error.value = err.message || $t('downloadView.failedToDeleteFile')
        confirmDialog.value = null
      } finally {
        confirmDialog.value = null
      }
    }
  }
}

function triggerAddFiles() {
  fileInput.value?.click()
}

function onFilesSelected(event) {
  const selectedFiles = Array.from(event.target.files)
  event.target.value = ''

  const existingNames = new Set(pendingFiles.value.map(f => f.fileName))
  for (const file of selectedFiles) {
    if (existingNames.has(file.name)) continue
    existingNames.add(file.name)
    pendingFiles.value.push({
      reference: generateRef(),
      fileName: file.name,
      size: file.size,
      file: file,
      status: 'toUpload',
      progress: 0,
    })
  }
}

function removePendingFile(file) {
  pendingFiles.value = pendingFiles.value.filter(f => f.reference !== file.reference)
}

async function cancelFileUpload(file) {
  if (file.abort) {
    file.abort()
  }
  pendingFiles.value = pendingFiles.value.filter(f => f.reference !== file.reference)

  // If no more active/error files, exit upload mode
  if (!pendingFiles.value.some(f => f.status === 'uploading' || f.status === 'toUpload' || f.status === 'error')) {
    isAddingFiles.value = false
  }

  // For streaming uploads the server goroutine stays blocked waiting for a
  // downloader even after the XHR is aborted. Explicitly delete the file so
  // it transitions to 'removed'/'deleted' and disappears from the file list.
  if (upload.value.stream && file.id) {
    try {
      await apiRemoveFile(
        { id: props.id, stream: true, uploadToken: uploadToken.value },
        file,
      )
    } catch (err) { console.warn('Failed to remove streaming file:', err) }
  }

  await fetchUpload()
}

async function cancelAllUploads() {
  uploadsCancelled = true
  const filesToClean = [...pendingFiles.value]
  for (const file of filesToClean) {
    if (file.abort) {
      file.abort()
    }
  }
  pendingFiles.value = []
  isAddingFiles.value = false

  // For streaming uploads, explicitly remove cancelled files from the server
  if (upload.value.stream) {
    await Promise.allSettled(
      filesToClean
        .filter(f => f.id)
        .map(f => apiRemoveFile(
          { id: props.id, stream: true, uploadToken: uploadToken.value },
          f,
        ))
    )
  }

  await fetchUpload()
}

// --- Shared upload helpers ---

const MAX_CONCURRENT = 5

// BasicAuth stored at component level so retries preserve credentials
let activeBasicAuth = null

// Whether the upload pool is currently running (re-entry guard)
let isUploading = false

// Locally update a server file's status (reactive, no full refresh needed)
function setServerFileStatus(fileId, status) {
  const serverFile = upload.value?.files?.find(f => f.id === fileId)
  if (serverFile) serverFile.status = status
}

// Upload a single file entry (shared by pool and individual retry)
function uploadFileEntry(fileEntry) {
  fileEntry.status = 'uploading'
  fileEntry.error = null
  fileEntry.progress = 0

  const isStream = upload.value.stream

  const { promise, abort } = uploadFile(
    { id: props.id, stream: isStream, uploadToken: uploadToken.value },
    { id: fileEntry.id, fileName: fileEntry.fileName, file: fileEntry.file },
    (progress) => { fileEntry.progress = progress },
    activeBasicAuth,
    isStream ? () => setServerFileStatus(fileEntry.id, 'uploading') : undefined,
  )

  fileEntry.abort = abort

  return promise.then((result) => {
    fileEntry.status = 'uploaded'
    fileEntry.id = result.id
    // Merge all server-detected metadata (fileType, fileSize, fileMd5)
    const serverFile = upload.value?.files?.find(f => f.id === result.id)
    if (serverFile) Object.assign(serverFile, result)
    // Remove from pending panel immediately
    pendingFiles.value = pendingFiles.value.filter(f => f.reference !== fileEntry.reference)
  }).catch(async (err) => {
    if (!err.cancelled) {
      fileEntry.status = 'error'
      fileEntry.error = err.message || $t('api.uploadFailed', { status: '' })
    }
    // Refresh server state so we know which files are still retryable
    await fetchUpload()
    // Remove pending files whose server status is no longer retryable
    // (e.g. cancelled by someone else, already downloaded, etc.)
    pendingFiles.value = pendingFiles.value.filter(f => {
      if (f.status !== 'error' || !f.id) return true
      const serverFile = upload.value?.files?.find(sf => sf.id === f.id)
      // Keep if server says missing (retryable) or file not found (keep error visible)
      return !serverFile || serverFile.status === 'missing'
    })
  })
}

// Check if we should exit upload mode
function checkUploadModeExit() {
  const hasErrors = pendingFiles.value.some(f => f.status === 'error')
  const hasActive = pendingFiles.value.some(f => f.status === 'uploading' || f.status === 'toUpload')
  if (!hasErrors && !hasActive) {
    isAddingFiles.value = false
  }
}

// --- Upload pool ---

async function uploadPendingFiles() {
  if (!pendingFiles.value.length || isUploading) return
  isUploading = true
  isAddingFiles.value = true
  uploadsCancelled = false

  activeBasicAuth = pendingBasicAuth || activeBasicAuth
  pendingBasicAuth = null

  // Re-check loop: after each batch, pick up files that were retried mid-batch
  while (!uploadsCancelled) {
    const filesToUpload = pendingFiles.value.filter(f => f.status === 'toUpload')
    if (!filesToUpload.length) break

    const queue = [...filesToUpload]
    const workers = Array.from({ length: Math.min(MAX_CONCURRENT, queue.length) }, async () => {
      while (queue.length > 0 && !uploadsCancelled) {
        const fileEntry = queue.shift()
        await uploadFileEntry(fileEntry)
      }
    })

    await Promise.allSettled(workers)
  }

  isUploading = false
  checkUploadModeExit()

  // Final refresh to sync with server truth
  if (!uploadsCancelled) {
    await fetchUpload()
  }
}

// --- Retry (funnel through standard upload pool) ---

function retryFile(file) {
  file.status = 'toUpload'
  file.error = null
  file.progress = 0
  file.abort = null
  if (!isUploading) {
    uploadPendingFiles()
  }
  // If pool is running, the re-check loop picks it up after the current batch
}

function retryAllFailed() {
  for (const file of pendingFiles.value) {
    if (file.status === 'error') {
      file.status = 'toUpload'
      file.error = null
      file.progress = 0
      file.abort = null
    }
  }
  if (!isUploading) {
    uploadPendingFiles()
  }
}

// File download links
function fileLinks() {
  if (!upload.value?.files) return []
  return upload.value.files
    .filter(f => f.status === 'uploaded')
    .map(f => ({
      ...f,
      url: getFileURL(props.id, f.id, f.fileName, upload.value?.stream),
    }))
}

// QR code helpers
function openQrUpload() {
  qrTitle.value = $t('downloadView.uploadLink')
  qrUrl.value = window.location.href
  showQr.value = true
}

function openQrFile(file) {
  qrTitle.value = file.fileName
  qrUrl.value = getFileURL(props.id, file.id, file.fileName, upload.value?.stream)
  showQr.value = true
}

// E2EE decrypt-and-download handler
async function decryptAndDownload(file) {
  if (!e2eePassphrase.value) {
    openPassphraseModal()
    return
  }

  isDecrypting.value = true
  try {
    const url = getFileURL(props.id, file.id, file.fileName, upload.value?.stream)
    const blob = await fetchAndDecrypt(url, e2eePassphrase.value)
    // Trigger browser download
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = file.fileName
    a.click()
    URL.revokeObjectURL(a.href)
  } catch (err) {
    uploadError.value = $t('downloadView.decryptionFailed', { error: err.message || $t('downloadView.wrongPassphrase') })
  } finally {
    isDecrypting.value = false
  }
}

function openPassphraseModal() {
  passphraseInput.value = e2eePassphrase.value || ''
  showPassphraseModal.value = true
}

function submitPassphrase() {
  if (!passphraseInput.value.trim()) return
  e2eePassphrase.value = passphraseInput.value.trim()
  passphraseInput.value = ''
  showPassphraseModal.value = false
}

// Whether this upload uses E2EE
const isE2EE = computed(() => !!upload.value?.e2ee)

onMounted(() => document.addEventListener('keydown', onViewerKeydown))
onUnmounted(() => document.removeEventListener('keydown', onViewerKeydown))

onMounted(async () => {
  // Extract E2EE passphrase from URL query param (?key=... inside the hash route)
  const queryKey = router.currentRoute.value.query.key
  if (queryKey) {
    e2eePassphrase.value = queryKey
    // Strip the key from the URL without reloading
    router.replace({ query: { id: props.id } })
  }

  // If uploadToken is in the URL (from admin URL), save it to memory and strip from URL
  const queryToken = router.currentRoute.value.query.uploadToken
  if (queryToken) {
    setToken(props.id, queryToken)
    router.replace({ query: { id: props.id } })
  }

  // Extract file= and t= from URL BEFORE fetchUpload, because the
  // activeFiles watcher ({ immediate: true }) fires when upload.value changes
  // and could race with our deep-link handling.
  const queryFileId = router.currentRoute.value.query.file
  const queryTime = router.currentRoute.value.query.t

  // Pre-set lastAutoViewedId so the auto-view watcher won't open a conflicting file
  if (queryFileId) {
    lastAutoViewedId.value = queryFileId
  }

  await fetchUpload()

  // Consume pending files from UploadView (if any)
  const pending = consumePendingFiles(props.id)
  if (pending) {
    pendingBasicAuth = pending.basicAuth
    pendingFiles.value = pending.files
    // Carry passphrase from pending store
    if (pending.passphrase && !e2eePassphrase.value) {
      e2eePassphrase.value = pending.passphrase
    }
    // Carry basic auth credentials for display in share card
    if (pending.login) basicAuthLogin.value = pending.login
    if (pending.password) basicAuthPassword.value = pending.password
    // Auto-start uploading
    uploadPendingFiles()
  }

  // Open file from URL (file= query param)
  if (queryFileId && upload.value?.files) {
    const targetFile = upload.value.files.find(f => f.id === queryFileId && f.status !== 'removed' && f.status !== 'deleted')
    if (targetFile) {
      // Store the time param for media seek
      if (queryTime) {
        const t = parseInt(queryTime, 10)
        if (!isNaN(t) && t > 0) initialMediaTime.value = t
      }

      isDeepLinked.value = true
      lastAutoViewedId.value = targetFile.id // confirm match (pre-set was by ID string)
      viewFile(targetFile)
    } else {
      // File not found — clear the pre-set guard so auto-view can work normally
      lastAutoViewedId.value = null
    }
  }

  // If this is an E2EE upload and we don't have the passphrase, prompt the user
  if (upload.value?.e2ee && !e2eePassphrase.value) {
    openPassphraseModal()
  }
})

// When the upload ID changes (e.g. user pastes a different URL), reset and re-fetch
watch(() => props.id, async (newId, oldId) => {
  if (newId === oldId) return

  // Reset all state
  upload.value = null
  error.value = null
  uploadError.value = null
  pendingFiles.value = []
  isAddingFiles.value = false
  closeViewer()
  lastAutoViewedId.value = null

  // Handle uploadToken in query
  const queryToken = router.currentRoute.value.query.uploadToken
  if (queryToken) {
    setToken(newId, queryToken)
    router.replace({ path: '/', query: { id: newId } })
  }

  await fetchUpload()
})

// Auto-show view panel if the upload contains exactly one file and it's a text file
watch(activeFiles, (files) => {
  // Only auto-view when the entire upload has exactly one file
  const totalUploadFiles = upload.value?.files?.filter(f => f.status !== 'removed' && f.status !== 'deleted')
  if (totalUploadFiles?.length !== 1) return
  // Don't auto-open for one-shot (viewing consumes the download) or streaming uploads
  if (upload.value?.oneShot || upload.value?.stream) return

  const file = files[0]
  if (file?.status === 'uploaded' && isViewableFile(file) && lastAutoViewedId.value !== file.id) {
    lastAutoViewedId.value = file.id
    viewFile(file)
  }
}, { immediate: true })
</script>

<template>
  <div class="flex justify-center flex-1 min-h-0 overflow-x-hidden">
    <div class="flex flex-col md:flex-row flex-1 max-w-screen-2xl px-4 sm:px-6 min-h-0 overflow-hidden">
      <!-- Sidebar -->
      <DownloadSidebar
        v-if="upload"
        :upload="{ ...upload, admin: isAdmin }"
        v-model:passphrase="e2eePassphrase"
        :login="basicAuthLogin"
        :password="basicAuthPassword"
        @edit-passphrase="openPassphraseModal"
        @delete-upload="deleteUpload"
        @add-files="triggerAddFiles"
        @show-qr="openQrUpload"
        @error="uploadError = $event" />

      <!-- Loading placeholder sidebar -->
      <aside v-else class="w-full md:w-80 md:shrink-0 p-4">
        <div class="sidebar-section animate-pulse">
          <div class="h-4 bg-surface-700 rounded w-1/2 mb-3" />
          <div class="h-8 bg-surface-700 rounded mb-2" />
          <div class="h-8 bg-surface-700 rounded" />
        </div>
      </aside>

      <!-- Main Content -->
      <main class="flex-1 py-4 md:pl-4 md:pr-0 overflow-y-auto">
      <div class="space-y-4">
        <!-- Loading -->
        <div v-if="loading" class="flex flex-col items-center justify-center py-16">
          <div class="animate-spin rounded-full h-8 w-8 border-2 border-accent-400 border-t-transparent" />
          <span class="mt-4 text-sm text-surface-400">{{ $t('downloadView.loadingUpload') }}</span>
        </div>

        <!-- Error -->
        <ErrorState v-else-if="error" :message="error" @retry="fetchUpload" />

        <!-- Upload Content -->
        <template v-else-if="upload">
          <!-- Inline Error Banner (for errors during file upload) -->
          <ErrorBanner v-if="uploadError" :message="uploadError" @dismiss="uploadError = null" />
          <!-- Comment -->
          <div v-if="upload.comments" class="glass-card p-4 animate-fade-in">
            <div class="flex items-center gap-2 mb-2">
              <svg class="w-4 h-4 text-surface-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
              </svg>
              <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider">{{ $t('downloadView.comment') }}</h3>
            </div>
            <div ref="commentsRef" class="prose prose-sm max-w-none" v-html="renderMarkdown(upload.comments)" />
          </div>

          <!-- E2EE Indicator -->
          <div v-if="isE2EE" class="glass-card p-3 flex items-center gap-3 animate-fade-in border-accent-400/30">
            <svg class="w-5 h-5 text-accent-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
            <div>
              <span class="text-sm text-accent-400 font-medium">{{ $t('downloadView.e2eeIndicator', { link: '' }) }}<a href="https://age-encryption.org" target="_blank" rel="noopener noreferrer" class="underline hover:text-accent-300 transition-colors">Age</a></span>
              <p class="text-xs text-surface-400 mt-0.5">{{ $t('downloadView.e2eeDecryptInBrowser') }}</p>
            </div>
          </div>

          <!-- Streaming Indicator -->
          <div v-if="upload.stream" class="glass-card p-3 flex items-center gap-3 animate-fade-in border-accent-400/30">
            <svg class="w-5 h-5 text-accent-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            <div>
              <span class="text-sm text-accent-400 font-medium">{{ $t('downloadView.streamingUpload') }}</span>
              <p class="text-xs text-surface-400 mt-0.5">
                {{ $t('downloadView.streamingDescription') }}
              </p>
              <p v-if="config.streamTimeout > 0" class="text-xs text-surface-400 mt-0.5">
                {{ $t('downloadView.streamTimeout', { timeout: streamTimeoutLabel }) }}
              </p>
            </div>
          </div>

          <!-- Decrypting Spinner -->
          <div v-if="isDecrypting" class="flex items-center justify-center py-4">
            <div class="animate-spin rounded-full h-6 w-6 border-2 border-accent-400 border-t-transparent" />
            <span class="ml-3 text-sm text-surface-400">{{ $t('downloadView.decrypting') }}</span>
          </div>

          <!-- File Viewer -->
          <div v-if="viewingFile" id="file-viewer-panel" class="glass-card overflow-hidden animate-fade-in">
            <div class="flex items-center justify-between border-b border-surface-700/50 px-4 py-2">
              <div class="flex items-center gap-2">
                <!-- Image icon for image files -->
                <svg v-if="isViewingImage" class="w-4 h-4 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <!-- Film icon for video files -->
                <svg v-else-if="isViewingVideo" class="w-4 h-4 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
                <!-- Music icon for audio files -->
                <svg v-else-if="isViewingAudio" class="w-4 h-4 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
                </svg>
                <!-- Code icon for text files -->
                <svg v-else class="w-4 h-4 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                </svg>
                <span class="text-sm font-medium text-surface-200">{{ viewingFile.fileName }}</span>
              </div>
              <div class="flex items-center gap-1">
                <CopyButton v-if="viewingContent && !isViewingVideo && !isViewingAudio" :text="viewingContent" :label="$t('common.copy')" />
                <!-- Copy link at current time (video/audio only) -->
                <CopyButton v-if="isViewingVideo || isViewingAudio"
                            :text="shareAtTimeUrl"
                            :label="$t('downloadView.copyLinkAtTime')"
                            size="sm" />
                <!-- Prev/Next navigation (only when multiple viewable files) -->
                <template v-if="viewableFiles.length > 1">
                  <button class="p-1 transition-colors"
                          :class="hasPrev ? 'text-surface-400 hover:text-surface-100' : 'text-surface-700 cursor-default'"
                          :disabled="!hasPrev"
                          :title="$t('downloadView.previousFile')"
                          @click="viewPrev">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
                    </svg>
                  </button>
                  <span class="text-xs text-surface-500 tabular-nums min-w-8 text-center">{{ viewerIndex + 1 }}/{{ viewableFiles.length }}</span>
                  <button class="p-1 transition-colors"
                          :class="hasNext ? 'text-surface-400 hover:text-surface-100' : 'text-surface-700 cursor-default'"
                          :disabled="!hasNext"
                          :title="$t('downloadView.nextFile')"
                          @click="viewNext">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                    </svg>
                  </button>
                </template>
                <button class="p-1 text-surface-400 hover:text-surface-100 transition-colors"
                        :title="$t('downloadView.closeViewer')"
                        @click="closeViewer">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
            <!-- Markdown Code/Preview tabs -->
            <MarkdownTabs v-if="isViewingMarkdown && !viewingLoading && !viewingError"
                          :modelValue="viewerTab"
                          @update:modelValue="viewerTab = $event"
                          :renderedHtml="renderedFileContent">
              <div class="p-2">
                <CodeEditor
                  v-model="viewingContent"
                  :filename="viewingFile.fileName"
                  :readonly="true"
                />
              </div>
            </MarkdownTabs>
            <div v-if="viewingLoading" class="flex items-center justify-center py-8">
              <div class="animate-spin rounded-full h-6 w-6 border-2 border-accent-400 border-t-transparent" />
              <span class="ml-3 text-sm text-surface-400">{{ $t('downloadView.loadingFileContent') }}</span>
            </div>
            <div v-else-if="viewingError" class="p-4 text-sm text-danger-500">{{ viewingError }}</div>
            <div v-else-if="isViewingImage" class="p-4 flex items-center justify-center bg-surface-900/50">
              <img :src="viewingImageUrl"
                   :alt="viewingFile.fileName"
                   class="max-w-full max-h-[70vh] object-contain rounded"
                   @load="onMediaPreviewLoaded" />
            </div>
            <div v-else-if="isViewingVideo" class="p-4 flex items-center justify-center bg-surface-900/50">
              <video
                     :key="viewingFile.id"
                     :src="viewingVideoUrl"
                     controls
                     preload="metadata"
                     class="max-w-full max-h-[70vh] rounded"
                     @timeupdate="onMediaTimeUpdate"
                     @loadedmetadata="onMediaLoadedMetadata" />
            </div>
            <div v-else-if="isViewingAudio" class="p-4 flex items-center justify-center bg-surface-900/50">
              <audio
                     :key="viewingFile.id"
                     :src="viewingAudioUrl"
                     controls
                     preload="metadata"
                     class="w-full max-w-lg"
                     @timeupdate="onMediaTimeUpdate"
                     @loadedmetadata="onMediaLoadedMetadata" />
            </div>
            <div v-else-if="!isViewingMarkdown" class="p-2">
              <CodeEditor
                v-model="viewingContent"
                :filename="viewingFile.fileName"
                :readonly="true"
              />
            </div>
          </div>

          <!-- File List -->
          <div v-if="activeFiles.length" class="space-y-2">
            <div class="flex items-center justify-between px-1">
              <h3 class="text-sm font-medium text-surface-400">
                <template v-if="isAddingFiles && !upload.stream">
                  {{ uploadedCount }}/{{ totalFiles }} {{ $t('homeView.files').toLowerCase() }} {{ $t('fileRow.uploaded').replace(':','') }}
                </template>
                <template v-else>
                  {{ totalFiles }} {{ $t('homeView.files').toLowerCase() }}
                </template>
              </h3>
              <CopyButton
                v-if="fileLinks().length > 1"
                :text="fileLinks().map(f => f.url).join('\n')"
                :label="$t('common.copyAllLinks')"
                size="sm" />
            </div>

            <FileRow v-for="file in activeFiles"
                     :key="file.id"
                     :file="file"
                     :upload-id="id"
                     mode="download"
                     :can-remove="canRemoveFiles"
                     :is-stream="upload.stream"
                     :is-one-shot="upload.oneShot"
                     :is-e2ee="isE2EE"
                     :show-download-stats="isAdmin"
                     @remove="deleteFile"
                     @show-qr="openQrFile"
                     @view="viewFile"
                     @decrypt-download="decryptAndDownload" />
          </div>

          <!-- Pending Files (staged for upload / uploading) -->
          <div v-if="pendingFiles.length" class="space-y-2">
            <div class="flex items-center justify-between px-1">
              <h3 class="text-sm font-medium text-surface-400">
                <template v-if="isAddingFiles && pendingFiles.some(f => f.status === 'error') && !pendingFiles.some(f => f.status === 'uploading' || f.status === 'toUpload')">
                  {{ $t('downloadView.filesFailed', { count: pendingFiles.filter(f => f.status === 'error').length }, pendingFiles.filter(f => f.status === 'error').length) }}
                </template>
                <template v-else-if="isAddingFiles">
                  {{ $t('downloadView.filesLeftToUpload', { count: pendingFiles.filter(f => f.status !== 'uploaded').length }, pendingFiles.filter(f => f.status !== 'uploaded').length) }}
                </template>
                <template v-else>
                  {{ pendingFiles.length }} {{ $t('homeView.files').toLowerCase() }}
                </template>
              </h3>
              <div class="flex items-center gap-3">
                <button v-if="isAddingFiles && pendingFiles.some(f => f.status === 'error') && !pendingFiles.some(f => f.status === 'uploading' || f.status === 'toUpload')"
                        class="text-xs text-accent-400 hover:text-accent-300 transition-colors"
                        @click="retryAllFailed">
                  {{ $t('downloadView.retryFailed') }}
                </button>
                <button v-if="isAddingFiles"
                        class="text-xs text-danger-500 hover:text-danger-400 transition-colors"
                        @click="cancelAllUploads">
                  {{ $t('downloadView.cancelAll') }}
                </button>
              </div>
            </div>

            <FileRow v-for="file in pendingFiles"
                     :key="file.reference"
                     :file="file"
                     :mode="isAddingFiles ? 'uploading' : 'upload'"
                     @remove="isAddingFiles ? cancelFileUpload(file) : removePendingFile(file)"
                     @cancel="cancelFileUpload"
                     @retry="retryFile" />
          </div>

          <!-- Upload Pending Files Button (only shown when files are staged but not yet uploading) -->
          <div v-if="pendingFiles.length && !isAddingFiles" class="flex justify-end">
            <button class="btn-success px-8 py-3 text-base" @click="uploadPendingFiles">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
              </svg>
              {{ $t('common.upload') }}
            </button>
          </div>

          <!-- Upload progress indicator -->
          <div v-if="isAddingFiles && pendingFiles.some(f => f.status === 'uploading' || f.status === 'toUpload')" class="flex items-center justify-center py-2">
            <div class="animate-spin rounded-full h-4 w-4 border-2 border-accent-400 border-t-transparent" />
            <span class="ml-2 text-xs text-surface-400">{{ $t('downloadView.uploadingFiles') }}</span>
          </div>

          <!-- No files -->
          <div v-if="!activeFiles.length && !pendingFiles.length" class="glass-card p-8 text-center">
            <p class="text-surface-400">{{ $t('downloadView.noFilesInUpload') }}</p>
          </div>


        </template>
      </div>
    </main>

    <!-- Hidden file input for adding files -->
    <input ref="fileInput"
           type="file"
           multiple
           class="hidden"
           @change="onFilesSelected" />

    <!-- QR Code Dialog -->
    <QrCodeDialog v-if="showQr"
                  :title="qrTitle"
                  :url="qrUrl"
                  @close="showQr = false" />

    <!-- Confirm Dialog -->
    <ConfirmDialog v-if="confirmDialog"
                   :title="confirmDialog.title"
                   :message="confirmDialog.message"
                   :confirm-text="confirmDialog.confirmText"
                   @confirm="confirmDialog.onConfirm"
                   @cancel="confirmDialog = null" />

    <!-- Passphrase Modal -->
    <div v-if="showPassphraseModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
         @mousedown.self="e2eePassphrase ? (showPassphraseModal = false) : null">
      <div class="glass-card p-6 w-full max-w-sm mx-4 space-y-4 animate-fade-in">
        <div class="flex items-center gap-3">
          <svg class="w-6 h-6 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          <h3 class="text-lg font-medium text-surface-100">{{ $t('downloadView.enterPassphrase') }}</h3>
        </div>
        <p class="text-sm text-surface-400">{{ $t('downloadView.e2eePassphrasePrompt') }}</p>
        <input type="text"
               v-model="passphraseInput"
               class="input-field font-mono text-sm"
               :placeholder="$t('downloadView.passphrasePlaceholder')"
               @keydown.enter="submitPassphrase" />
        <div class="flex justify-end">
          <button class="btn-primary px-4 py-1.5 text-sm"
                  :disabled="!passphraseInput.trim()"
                  @click="submitPassphrase">{{ $t('common.decrypt') }}</button>
        </div>
      </div>
    </div>
    </div>
  </div>
</template>
