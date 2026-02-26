<script setup>
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { config, isFeatureEnabled, isFeatureForced, isFeatureDefaultOn } from '../config.js'
import { createUpload } from '../api.js'
import { setToken } from '../tokenStore.js'
import { setPendingFiles } from '../pendingUploadStore.js'
import { generateRef, ttlToSeconds, secondsToTTL, encodeBasicAuth, humanReadableSize } from '../utils.js'
import { encryptFile } from '../crypto.js'
import { auth } from '../authStore.js'
import { renderMarkdown } from '../markdown.js'
import UploadSidebar from '../components/UploadSidebar.vue'
import FileRow from '../components/FileRow.vue'
import { defineAsyncComponent } from 'vue'
const CodeEditor = defineAsyncComponent(() => import('../components/CodeEditor.vue'))

const router = useRouter()
const fileInput = ref(null)

// Effective limits (user-specific overrides server config)
const effectiveMaxFileSize = computed(() => {
  const user = auth.user
  if (user && user.maxFileSize !== 0 && user.maxFileSize !== undefined) return user.maxFileSize
  return config.maxFileSize
})

const effectiveMaxTTL = computed(() => {
  const user = auth.user
  if (user && user.maxTTL !== 0 && user.maxTTL !== undefined) return user.maxTTL
  return config.maxTTL
})

// Upload settings
const defaultTTL = computed(() => {
  let ttl = config.defaultTTL
  // Clamp to user maxTTL if the user has a stricter limit
  const user = auth.user
  if (user && user.maxTTL > 0 && ttl > user.maxTTL) {
    ttl = user.maxTTL
  }
  return secondsToTTL(ttl)
})

const settings = reactive({
  oneShot: isFeatureDefaultOn('one_shot'),
  stream: isFeatureDefaultOn('stream'),
  removable: isFeatureDefaultOn('removable'),
  passwordEnabled: isFeatureDefaultOn('password'),
  login: '',
  password: '',
  commentEnabled: isFeatureDefaultOn('comments'),
  extendTTL: isFeatureDefaultOn('extend_ttl'),
  e2eeEnabled: isFeatureDefaultOn('e2ee'),
  e2eePassphrase: '',
  notifyCreator: false,
  receivers: [],
  // When both defaultTTL and maxTTL are 0 (no limit), default to "never expires" ON (opt-out)
  neverExpires: config.defaultTTL <= 0 && effectiveMaxTTL.value <= 0,
  ttlValue: defaultTTL.value.value || 15,
  ttlUnit: defaultTTL.value.unit || 'days',
})

// Comment
const commentText = ref('')
const commentTab = ref('write') // 'write' | 'preview'
const renderedComment = computed(() => {
  if (!commentText.value) return ''
  return renderMarkdown(commentText.value)
})

// File list
const files = ref([])
const isUploading = ref(false)
const uploadError = ref(null)

// Drag-and-drop state
const isDragging = ref(false)
let dragCounter = 0

// Whether the user has files selected and ready to upload
const hasFiles = computed(() => files.value.length > 0)

// Text paste mode
const textMode = ref(false)
const textContent = ref('')
const textFilename = ref('paste.txt')

function onLanguageDetected({ extension }) {
  if (!extension) return

  const current = textFilename.value
  // Only auto-set extension when using the default name
  if (!current || current.startsWith('paste.')) {
    textFilename.value = `paste.${extension}`
  }
  // Don't overwrite user-chosen filenames
}

function addTextAsFile() {
  if (!textContent.value.trim()) return
  const blob = new Blob([textContent.value], { type: 'text/plain' })
  const file = new File([blob], textFilename.value || 'paste.txt', { type: 'text/plain' })
  addFiles([file])
  textContent.value = ''
  textMode.value = false
}

function triggerFileSelect() {
  fileInput.value?.click()
}

function addFiles(rawFiles) {
  const existingNames = new Set(files.value.map(f => f.fileName))
  for (const file of rawFiles) {
    // Check max file size (user-specific or server default)
    const maxSize = effectiveMaxFileSize.value
    if (maxSize > 0 && file.size > maxSize) {
      uploadError.value = `File "${file.name}" is too big (${humanReadableSize(file.size)}). Max: ${humanReadableSize(maxSize)}`
      continue
    }
    // Skip duplicates
    if (existingNames.has(file.name)) continue
    existingNames.add(file.name)

    files.value.push({
      reference: generateRef(),
      fileName: file.name.slice(0, 1024),
      size: file.size,
      file: file,
      status: 'toUpload',
      progress: 0,
    })
  }
}

function onFilesSelected(event) {
  addFiles(Array.from(event.target.files))
  event.target.value = ''
}

// Drag-and-drop handlers
function onDragEnter(e) {
  e.preventDefault()
  dragCounter++
  isDragging.value = true
}

function onDragOver(e) {
  e.preventDefault()
}

function onDragLeave(e) {
  e.preventDefault()
  dragCounter--
  if (dragCounter <= 0) {
    dragCounter = 0
    isDragging.value = false
  }
}

function onDrop(e) {
  e.preventDefault()
  dragCounter = 0
  isDragging.value = false
  if (e.dataTransfer?.files?.length) {
    addFiles(Array.from(e.dataTransfer.files))
  }
}

// Clipboard paste handler
function onPaste(e) {
  if (isUploading.value) return
  // If comment textarea is focused, don't intercept
  if (e.target?.tagName === 'TEXTAREA') return
  const clipFiles = Array.from(e.clipboardData?.files || [])
  if (clipFiles.length) {
    e.preventDefault()
    addFiles(clipFiles)
  } else {
    // Plain text paste — open text mode with clipboard content
    const text = e.clipboardData?.getData('text/plain')
    if (text) {
      e.preventDefault()
      textContent.value = text
      textFilename.value = 'paste.txt'
      textMode.value = true
    }
  }
}

function removeLocalFile(file) {
  files.value = files.value.filter(f => f.reference !== file.reference)
}

function updateFileName(file, newName) {
  const f = files.value.find(fl => fl.reference === file.reference)
  if (f && newName) f.fileName = newName
}

function buildUploadParams() {
  const params = {
    oneShot: settings.oneShot,
    stream: settings.stream,
    removable: settings.removable,
    extend_ttl: settings.extendTTL,
    ttl: settings.neverExpires ? -1 : ttlToSeconds(settings.ttlValue, settings.ttlUnit),
  }

  if (settings.passwordEnabled && settings.login && settings.password) {
    params.login = settings.login
    params.password = settings.password
  }

  if (settings.commentEnabled && commentText.value.trim()) {
    params.comments = commentText.value.trim()
  }

  if (settings.e2eeEnabled) {
    params.e2ee = 'age'
  }

  // Notification params
  if (isFeatureEnabled('notification')) {
    if (settings.notifyCreator) {
      params.notifyCreator = true
    }
    if (settings.receivers && settings.receivers.length > 0) {
      params.receivers = settings.receivers
    }
  }

  // Pre-populate files so the server assigns IDs (matched back via reference)
  params.files = files.value.map(f => ({
    fileName: f.fileName,
    fileSize: f.size,
    fileType: f.file?.type || '',
    reference: f.reference,
  }))

  return params
}

const commentsRequired = computed(() => isFeatureForced('comments') && !commentText.value.trim())

async function createEmptyUpload() {
  if (isUploading.value) return
  if (commentsRequired.value) {
    uploadError.value = 'A comment is required'
    return
  }

  isUploading.value = true
  uploadError.value = null

  try {
    const upload = await createUpload(buildUploadParams())
    setToken(upload.id, upload.uploadToken)
    router.push({ path: '/', query: { id: upload.id } })
  } catch (err) {
    uploadError.value = err.status
      ? `${err.message} (HTTP ${err.status})`
      : (err.message || 'Failed to create upload')
  } finally {
    isUploading.value = false
  }
}

async function doUpload() {
  if (!hasFiles.value || isUploading.value) return
  if (commentsRequired.value) {
    uploadError.value = 'A comment is required'
    return
  }

  isUploading.value = true
  uploadError.value = null

  try {
    const params = buildUploadParams()
    const upload = await createUpload(params)

    // Prepare basic auth if password was set
    const basicAuth = (settings.passwordEnabled && settings.login && settings.password)
      ? encodeBasicAuth(settings.login, settings.password)
      : null

    // Stash files for DownloadView to pick up and upload
    // If E2EE is enabled, encrypt files before stashing
    let pendingFiles
    if (settings.e2eeEnabled && settings.e2eePassphrase) {
      pendingFiles = await Promise.all(files.value.map(async (f) => {
        const encryptedBlob = await encryptFile(f.file, settings.e2eePassphrase)
        return {
          ...f,
          file: encryptedBlob,
          id: upload.files?.find(sf => sf.reference === f.reference)?.id,
        }
      }))
    } else {
      pendingFiles = files.value.map(f => ({
        ...f,
        id: upload.files?.find(sf => sf.reference === f.reference)?.id,
      }))
    }

    const passphrase = settings.e2eeEnabled ? settings.e2eePassphrase : null
    setPendingFiles(upload.id, pendingFiles, basicAuth, passphrase)

    // Navigate to download view — passphrase is carried via pendingUploadStore
    setToken(upload.id, upload.uploadToken)
    router.push({ path: '/', query: { id: upload.id } })
  } catch (err) {
    uploadError.value = err.status
      ? `${err.message} (HTTP ${err.status})`
      : (err.message || 'Failed to create upload')
  } finally {
    isUploading.value = false
  }
}
</script>

<template>
  <div class="flex justify-center flex-1 min-h-0 overflow-x-hidden">
    <div class="flex flex-col md:flex-row flex-1 max-w-screen-2xl px-4 sm:px-6 min-h-0 overflow-hidden" @paste="onPaste">
      <!-- Sidebar -->
      <UploadSidebar
        :settings="settings"
        :effectiveMaxTTL="effectiveMaxTTL"
        :isAuthenticated="!!auth.user"
        @update:settings="Object.assign(settings, $event)" />

      <!-- Main Content -->
      <main class="flex-1 py-4 md:pl-4 md:pr-0 overflow-y-auto"
            @dragenter="onDragEnter"
            @dragover="onDragOver"
            @dragleave="onDragLeave"
            @drop="onDrop">
      <div class="space-y-4">
        <!-- Error Banner -->
        <div v-if="uploadError"
             class="glass-card border-danger-500/50 p-4 flex items-center gap-3 animate-fade-in">
          <svg class="w-5 h-5 text-danger-500 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span class="text-sm text-danger-500">{{ uploadError }}</span>
          <button class="ml-auto text-surface-400 hover:text-white" @click="uploadError = null">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Markdown Editor -->
        <div v-if="settings.commentEnabled && !isUploading" class="glass-card overflow-hidden animate-fade-in"
             :class="{ 'ring-1 ring-danger-500/50': commentsRequired }">
          <div class="flex border-b border-surface-700/50">
            <button class="px-4 py-2 text-sm font-medium transition-colors"
                    :class="commentTab === 'write'
                      ? 'text-accent-400 border-b-2 border-accent-400 bg-surface-700/30'
                      : 'text-surface-400 hover:text-surface-200'"
                    @click="commentTab = 'write'">
              <svg class="w-4 h-4 inline-block mr-1 -mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
              Write
              <span v-if="isFeatureForced('comments')" class="ml-1 text-[10px] text-danger-400 font-semibold uppercase">required</span>
            </button>
            <button class="px-4 py-2 text-sm font-medium transition-colors"
                    :class="commentTab === 'preview'
                      ? 'text-accent-400 border-b-2 border-accent-400 bg-surface-700/30'
                      : 'text-surface-400 hover:text-surface-200'"
                    @click="commentTab = 'preview'">
              <svg class="w-4 h-4 inline-block mr-1 -mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
              Preview
            </button>
          </div>
          <div v-if="commentTab === 'write'" class="p-3">
            <textarea
              v-model="commentText"
              class="w-full bg-transparent border border-surface-700 rounded-lg p-3 text-sm text-surface-200
                     placeholder-surface-500 font-mono resize-y focus:outline-none focus:border-accent-500/50
                     transition-colors min-h-[120px]"
              :placeholder="isFeatureForced('comments') ? 'Write a comment... (required, Markdown supported)' : 'Write a comment... (Markdown supported)'"
            />
          </div>
          <div v-else class="p-4 min-h-[120px]">
            <div v-if="renderedComment"
                 class="prose prose-invert prose-sm max-w-none"
                 v-html="renderedComment" />
            <p v-else class="text-sm text-surface-500 italic">Nothing to preview</p>
          </div>
        </div>

        <!-- Text Paste Mode -->
        <div v-if="textMode && !isUploading" class="glass-card overflow-hidden animate-fade-in">
          <div class="flex items-center justify-between border-b border-surface-700/50 px-4 py-2">
            <div class="flex items-center gap-2">
              <svg class="w-4 h-4 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <span class="text-sm font-medium text-surface-200">Text Upload</span>
            </div>
            <button class="text-surface-400 hover:text-white transition-colors"
                    @click="textMode = false; textContent = ''">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div class="p-3 space-y-3">
            <div class="flex items-center gap-2">
              <label class="text-xs text-surface-400 shrink-0">Filename</label>
              <input type="text"
                     v-model="textFilename"
                     class="input-field text-sm flex-1 font-mono"
                     placeholder="paste.txt" />
            </div>
            <CodeEditor
              v-model="textContent"
              :filename="textFilename"
              placeholder="Paste or type text here..."
              @language-detected="onLanguageDetected"
            />
            <div class="flex justify-end gap-2">
              <button class="btn border border-surface-600 bg-surface-700/50 text-surface-300 hover:bg-surface-600/50
                             hover:text-white px-4 py-1.5 text-sm transition-all"
                      @click="textMode = false; textContent = ''">
                Cancel
              </button>
              <button class="btn-primary px-4 py-1.5 text-sm"
                      :disabled="!textContent.trim()"
                      @click="addTextAsFile">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                Add as File
              </button>
            </div>
          </div>
        </div>

        <!-- Add Files Zone (with drag-and-drop) -->
        <div v-if="!isUploading && !textMode"
             class="glass-card p-8 flex flex-col items-center justify-center gap-4 cursor-pointer
                    transition-all duration-200 group"
             :class="isDragging
               ? 'border-accent-500 bg-accent-500/10 scale-[1.01]'
               : 'hover:bg-surface-700/30'"
             @click="triggerFileSelect">
          <div class="w-16 h-16 rounded-full flex items-center justify-center transition-colors duration-200"
               :class="isDragging ? 'bg-accent-500/30' : 'bg-accent-500/10 group-hover:bg-accent-500/20'">
            <svg class="w-8 h-8 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
            </svg>
          </div>
          <div class="text-center">
            <p class="text-surface-200 font-medium">
              {{ isDragging ? 'Drop files here' : 'Drop, paste or click to select files' }}
            </p>
            <p v-if="effectiveMaxFileSize > 0" class="text-sm text-surface-500 mt-1">
              Max {{ humanReadableSize(effectiveMaxFileSize) }} per file
            </p>
            <p v-else-if="effectiveMaxFileSize === -1" class="text-sm text-surface-500 mt-1">
              No file size limit
            </p>
            <div class="flex items-center justify-center gap-3 mt-3">
              <button class="text-xs text-surface-400 hover:text-accent-400 transition-colors"
                      title="Allow adding files to an upload after its creation"
                      @click.stop="createEmptyUpload">
                Create empty upload
              </button>
              <span class="text-surface-600 text-xs">·</span>
              <button v-if="isFeatureEnabled('text')" class="text-xs text-surface-400 hover:text-accent-400 transition-colors"
                      @click.stop="textMode = true">
                Paste text
              </button>
            </div>
          </div>
        </div>

        <!-- Hidden file input -->
        <input ref="fileInput"
               type="file"
               multiple
               class="hidden"
               @change="onFilesSelected" />

        <!-- File List -->
        <div v-if="files.length" class="space-y-2">
          <div class="flex items-center justify-between px-1">
            <h3 class="text-sm font-medium text-surface-400">
              {{ files.length }} file{{ files.length > 1 ? 's' : '' }} selected
            </h3>
          </div>

          <FileRow v-for="file in files"
                   :key="file.reference"
                   :file="file"
                   mode="upload"
                   @remove="removeLocalFile"
                   @update-name="(name) => updateFileName(file, name)" />
        </div>

        <!-- Upload Button -->
        <div v-if="hasFiles && !isUploading" class="flex justify-end">
          <button class="btn-success px-8 py-3 text-base" @click="doUpload">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
            </svg>
            Upload
          </button>
        </div>

        <!-- Uploading Spinner (while createUpload is in progress) -->
        <div v-if="isUploading" class="flex items-center justify-center py-4">
          <div class="animate-spin rounded-full h-6 w-6 border-2 border-accent-500 border-t-transparent" />
          <span class="ml-3 text-sm text-surface-400">Creating upload...</span>
        </div>
      </div>
    </main>
    </div>
  </div>
</template>
