<script setup>
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { auth, logout } from '../authStore.js'
import { config, isFeatureEnabled } from '../config.js'
import ErrorBanner from '../components/ErrorBanner.vue'
import UploadControls from '../components/UploadControls.vue'
import {
    getUserUploads, deleteUserUploads, removeUpload,
    getUserTokens, createToken, revokeToken,
    deleteAccount, updateUser, getUserStatistics
} from '../api.js'
import {
    humanReadableSize, quotaLabel, ttlLabel,
    formatDate, buildEditForm, buildEditPayload,
} from '../utils.js'
import CopyButton from '../components/CopyButton.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import EditUserModal from '../components/EditUserModal.vue'
import UploadCard from '../components/UploadCard.vue'

const router = useRouter()
const route = useRoute()
const { t: $t } = useI18n()

// ── Display mode ──
const tokenFilter = ref(null)

// ── Uploads ──
const uploads = ref([])
const uploadsCursor = ref(null)
const uploadsLoading = ref(false)
const uploadsTotal = ref(null)
const uploadsSortBy = ref('date')
const uploadsSortOrder = ref('desc')

// Badge filters — false = no filter, true = only matching
const badgeFilters = ref({
    oneShot: false,
    removable: false,
    stream: false,
    extendTTL: false,
    password: false,
    e2ee: false,
})
const BADGE_FILTER_KEYS = ['oneShot', 'removable', 'stream', 'extendTTL', 'password', 'e2ee']

// ── Tokens ──
const tokens = ref([])
const tokensCursor = ref(null)
const tokensLoading = ref(false)
const newTokenComment = ref('')

// ── Confirmation state ──
const confirm = ref(null) // { message, action }

// ── Edit account ──
const showEditAccount = ref(false)
const editForm = ref({})
const editSaving = ref(false)
const editError = ref('')
const error = ref(null)
const successMessage = ref('')

// ── Helpers ──

const editTTLUnit = ref(60)

// ── Stats ──
const userStats = ref(null)
const statsLoading = ref(false)

async function loadUserStats() {
    statsLoading.value = true
    try {
        userStats.value = await getUserStatistics()
    } catch (e) {
        error.value = 'Failed to load user stats'
    } finally {
        statsLoading.value = false
    }
}

// Effective default TTL = min(config.defaultTTL, user.maxTTL) when user has a limit
const effectiveDefaultTTL = computed(() => {
    const cfgTTL = config.defaultTTL || 0
    const userMaxTTL = auth.user?.maxTTL || 0
    if (userMaxTTL > 0 && cfgTTL > 0) return Math.min(cfgTTL, userMaxTTL)
    if (userMaxTTL > 0) return userMaxTTL
    return cfgTTL
})

// ── Token lookup map (token UUID → comment) ──
const tokenMap = computed(() => {
    const map = {}
    for (const t of tokens.value) {
        map[t.token] = t.comment || ''
    }
    return map
})

function tokenLabel(tokenStr) {
    if (!tokenStr) return ''
    const comment = tokenMap.value[tokenStr]
    if (comment) return comment
    return tokenStr.substring(0, 8) + '...'
}

// ── Uploads API ──
async function loadUploads(more = false) {
    uploadsLoading.value = true
    try {
        const opts = { limit: 50 }
        if (tokenFilter.value) opts.token = tokenFilter.value
        if (more && uploadsCursor.value) opts.after = uploadsCursor.value
        if (uploadsSortBy.value !== 'date') opts.sort = uploadsSortBy.value
        if (uploadsSortOrder.value !== 'desc') opts.order = uploadsSortOrder.value
        // Badge filters
        for (const key of BADGE_FILTER_KEYS) {
            if (badgeFilters.value[key]) opts[key] = true
        }
        const data = await getUserUploads(opts)
        if (more) {
            uploads.value = [...uploads.value, ...data.results]
        } else {
            uploads.value = data.results || []
        }
        uploadsCursor.value = data.after || null
        uploadsTotal.value = data.total ?? null
    } catch (err) {
        error.value = 'Could not load uploads'
    } finally {
        uploadsLoading.value = false
    }
}

async function handleDeleteUpload(upload) {
    confirm.value = {
        message: `Delete upload ${upload.id}?`,
        action: async () => {
            try {
                await removeUpload(upload.id, upload.uploadToken)
                uploads.value = uploads.value.filter(u => u.id !== upload.id)
            } catch (err) {
                error.value = 'Could not delete upload'
            }
            confirm.value = null
        }
    }
}

async function handleDeleteAllUploads() {
    const label = tokenFilter.value ? `all uploads for token ${tokenFilter.value}` : 'ALL your uploads'
    confirm.value = {
        message: `Delete ${label}? This cannot be undone.`,
        action: async () => {
            try {
                await deleteUserUploads(tokenFilter.value)
                uploads.value = []
                uploadsCursor.value = null
            } catch (err) {
                error.value = 'Could not delete uploads'
            }
            confirm.value = null
        }
    }
}

function filterByToken(token) {
    tokenFilter.value = token
    uploads.value = []
    uploadsCursor.value = null
    if (display.value === 'uploads') {
        loadUploads()
    } else {
        router.push('/home/uploads')
    }
}

function clearTokenFilter() {
    tokenFilter.value = null
    uploads.value = []
    uploadsCursor.value = null
    loadUploads()
}

function toggleBadgeFilter(key) {
    badgeFilters.value[key] = !badgeFilters.value[key]
    uploads.value = []
    uploadsCursor.value = null
    internalNav = true
    router.push({ query: currentUploadsQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUploads() // Intentional: reads from refs directly, not URL, so safe before push resolves
}

function changeSortBy(val) {
    if (val === uploadsSortBy.value) return
    uploadsSortBy.value = val
    uploads.value = []
    uploadsCursor.value = null
    internalNav = true
    router.push({ query: currentUploadsQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUploads()
}

function changeSortOrder(val) {
    if (val === uploadsSortOrder.value) return
    uploadsSortOrder.value = val
    uploads.value = []
    uploadsCursor.value = null
    internalNav = true
    router.push({ query: currentUploadsQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUploads()
}

// ── Tokens API ──
async function loadTokens(more = false) {
    tokensLoading.value = true
    try {
        const opts = { limit: 50 }
        if (more && tokensCursor.value) opts.after = tokensCursor.value
        const data = await getUserTokens(opts)
        if (more) {
            tokens.value = [...tokens.value, ...data.results]
        } else {
            tokens.value = data.results || []
        }
        tokensCursor.value = data.after || null
    } catch (err) {
        error.value = 'Could not load tokens'
    } finally {
        tokensLoading.value = false
    }
}

async function handleCreateToken() {
    try {
        const token = await createToken(newTokenComment.value.trim() || undefined)
        tokens.value = [token, ...tokens.value]
        newTokenComment.value = ''
    } catch (err) {
        error.value = 'Could not create token'
    }
}

async function handleDeleteTokenUploads(token) {
    const label = token.comment || token.token.substring(0, 8) + '...'
    confirm.value = {
        message: `Delete all uploads for token "${label}"? The token itself will not be revoked.`,
        action: async () => {
            try {
                const result = await deleteUserUploads(token.token)
                confirm.value = null
                // Show feedback (backend returns "X uploads removed")
                successMessage.value = typeof result === 'string' ? result : 'Uploads removed'
                setTimeout(() => { successMessage.value = '' }, 3000)
            } catch (err) {
                confirm.value = null
                error.value = 'Could not delete token uploads'
            }
        }
    }
}

async function handleRevokeToken(token) {
    confirm.value = {
        message: `Revoke token ${token.token.substring(0, 8)}...? Uploads created with this token will remain.`,
        action: async () => {
            try {
                await revokeToken(token.token)
                tokens.value = tokens.value.filter(t => t.token !== token.token)
            } catch (err) {
                error.value = $t('homeView.failedToRevokeToken')
            }
            confirm.value = null
        }
    }
}

// ── Account ──
async function handleLogout() {
    await logout()
    router.push('/')
}

async function handleDeleteAccount() {
    confirm.value = {
        message: $t('homeView.deleteAccountConfirm'),
        action: async () => {
            try {
                await deleteAccount()
                auth.user = null
                router.push('/')
            } catch (err) {
                error.value = $t('homeView.failedToDeleteAccount')
            }
            confirm.value = null
        }
    }
}

function openEditAccount() {
    const { form, ttlUnit } = buildEditForm(auth.user)
    editForm.value = form
    editTTLUnit.value = ttlUnit
    editError.value = ''
    showEditAccount.value = true
}

async function saveEditAccount() {
    editSaving.value = true
    editError.value = ''
    try {
        const payload = buildEditPayload(editForm.value, editTTLUnit.value)
        const updated = await updateUser(payload)
        Object.assign(auth.user, updated)
        showEditAccount.value = false
    } catch (err) {
        editError.value = err.message || 'Failed to save'
    } finally {
        editSaving.value = false
    }
}

// ── Display switching (via route path) ──
const display = computed(() => route.params.tab || 'stats')

function showStats() {
    router.push('/home/stats')
}

function showUploads() {
    tokenFilter.value = null
    router.push('/home/uploads')
}

function showTokens() {
    router.push('/home/tokens')
}

// Build query params from current uploads tab filter state (omits defaults, excludes token)
function currentUploadsQuery() {
    const q = {
        sort: uploadsSortBy.value !== 'date' ? uploadsSortBy.value : undefined,
        order: uploadsSortOrder.value !== 'desc' ? uploadsSortOrder.value : undefined,
    }
    for (const key of BADGE_FILTER_KEYS) {
        if (badgeFilters.value[key]) q[key] = 'true'
    }
    return q
}

// ── Route → state sync ──

// Guard flag: suppresses watchers during programmatic navigation so they
// don't double-load or overwrite state the caller already set up.
let internalNav = false
watch(display, (tab, prevTab) => {
    if (tab === prevTab || internalNav) return
    if (tab === 'stats') {
        loadUserStats()
    } else if (tab === 'uploads') {
        uploadsSortBy.value = route.query.sort || 'date'
        uploadsSortOrder.value = route.query.order || 'desc'
        for (const key of BADGE_FILTER_KEYS) {
            badgeFilters.value[key] = route.query[key] === 'true'
        }
        uploads.value = []
        loadUploads()
    } else if (tab === 'tokens') {
        loadTokens()
    }
})

// Filter changes within the uploads tab (query-based) — triggers on back/forward
watch(() => route.query, (query, oldQuery) => {
    if (internalNav || route.params.tab !== 'uploads') return
    const changed = query.sort !== oldQuery?.sort ||
                    query.order !== oldQuery?.order ||
                    BADGE_FILTER_KEYS.some(k => query[k] !== oldQuery?.[k])
    if (changed) {
        uploadsSortBy.value = query.sort || 'date'
        uploadsSortOrder.value = query.order || 'desc'
        for (const key of BADGE_FILTER_KEYS) {
            badgeFilters.value[key] = query[key] === 'true'
        }
        uploads.value = []
        uploadsCursor.value = null
        loadUploads()
    }
})

// ── Init ──
onMounted(() => {
    if (!auth.user) {
        router.push('/login')
        return
    }
    const tab = display.value

    loadTokens()  // needed for token comment lookup map

    if (tab === 'uploads') {
        // Restore badge filters from URL
        uploadsSortBy.value = route.query.sort || 'date'
        uploadsSortOrder.value = route.query.order || 'desc'
        for (const key of BADGE_FILTER_KEYS) {
            badgeFilters.value[key] = route.query[key] === 'true'
        }
        loadUploads()
    } else if (tab !== 'tokens') {
        // tokens already loaded above via loadTokens()
        loadUserStats()
    }
})
</script>

<template>
  <div class="w-full max-w-screen-2xl mx-auto px-4 sm:px-6 py-6">
    <div class="flex flex-col md:flex-row gap-6">

      <!-- ═══════ Sidebar ═══════ -->
      <aside class="w-full md:w-72 shrink-0 space-y-4">

        <!-- User Info Card -->
        <div class="glass-card p-5 text-center space-y-3">
          <div class="w-14 h-14 rounded-full bg-accent-500/20 flex items-center justify-center mx-auto overflow-hidden">
            <img v-if="auth.user?.profilePicture"
                 :src="auth.user.profilePicture"
                 alt="Profile"
                 class="w-full h-full object-cover"
                 referrerpolicy="no-referrer" />
            <svg v-else class="w-7 h-7 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
            </svg>
          </div>
          <div>
            <p class="text-surface-200 font-medium">{{ auth.user?.login || auth.user?.name }}</p>
            <p class="text-xs text-surface-500">{{ auth.user?.provider }}</p>
            <p v-if="auth.user?.name" class="text-xs text-surface-400 mt-1">{{ auth.user.name }}</p>
            <p v-if="auth.user?.email" class="text-xs text-surface-400">{{ auth.user.email }}</p>
            <span v-if="auth.user?.admin"
                  class="inline-block mt-1 text-xs bg-emerald-500/20 text-emerald-400 px-2 py-0.5 rounded-full">
              admin
            </span>
          </div>
        </div>

        <!-- Nav Buttons -->
        <div class="glass-card p-2 space-y-1">
          <button @click="showStats"
                  :class="display === 'stats'
                    ? 'bg-accent-500/10 text-accent-400 border-l-2 border-accent-400'
                    : 'text-surface-300 hover:text-surface-100 hover:bg-surface-700/50 border-l-2 border-transparent'"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            Stats
          </button>

          <button @click="showUploads"
                  :class="display === 'uploads'
                    ? 'bg-accent-500/10 text-accent-400 border-l-2 border-accent-400'
                    : 'text-surface-300 hover:text-surface-100 hover:bg-surface-700/50 border-l-2 border-transparent'"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
            Uploads
          </button>

          <button @click="showTokens"
                  :class="display === 'tokens'
                    ? 'bg-accent-500/10 text-accent-400 border-l-2 border-accent-400'
                    : 'text-surface-300 hover:text-surface-100 hover:bg-surface-700/50 border-l-2 border-transparent'"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
            </svg>
            Tokens
          </button>
        </div>

        <!-- Account Actions -->
        <div class="glass-card p-2 space-y-1">
          <button @click="handleLogout"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm
                         text-surface-300 hover:text-surface-100 hover:bg-surface-700/50 transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
            </svg>
            {{ $t('homeView.signOut') }}
          </button>

          <button v-if="auth.user?.provider === 'local'"
                  @click="openEditAccount"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm
                         text-surface-300 hover:text-surface-100 hover:bg-surface-700/50 transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
            {{ $t('homeView.editAccount') }}
          </button>

          <button v-if="display === 'uploads'"
                  @click="handleDeleteAllUploads"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm
                         text-red-400/70 hover:text-red-400 hover:bg-red-500/10 transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            {{ $t('homeView.deleteUploads') }}
          </button>

          <button v-if="isFeatureEnabled('delete_account')"
                  @click="handleDeleteAccount"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm
                         text-red-400/70 hover:text-red-400 hover:bg-red-500/10 transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
            {{ $t('homeView.deleteAccount') }}
          </button>
        </div>
      </aside>

      <!-- ═══════ Main Content ═══════ -->
      <main class="flex-1 min-w-0">

        <!-- Error Banner -->
        <ErrorBanner v-if="error" :message="error" @dismiss="error = null" class="mb-4" />
        <div v-if="successMessage"
             class="mb-4 px-4 py-2.5 rounded-xl bg-green-500/15 border border-green-500/30
                    text-emerald-500 text-sm flex items-center gap-2">
          <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          {{ successMessage }}
        </div>

        <!-- ─── Stats View ─── -->
        <template v-if="display === 'stats'">
          <div v-if="statsLoading" class="text-center py-12 text-surface-500">{{ $t('homeView.loadingStats') }}</div>

          <div v-else class="space-y-4">
            <!-- User Configuration -->
            <div class="glass-card p-5">
              <h3 class="text-sm text-surface-400 uppercase tracking-wider mb-4">{{ $t('homeView.userConfiguration') }}</h3>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-center">
                <div>
                  <p class="text-xs text-surface-500">{{ $t('homeView.maxFileSize') }}</p>
                  <p class="text-surface-200 font-medium">{{ quotaLabel(auth.user?.maxFileSize, $t) }}</p>
                  <p v-if="!auth.user?.maxFileSize && config.maxFileSize" class="text-xs text-surface-500">({{ quotaLabel(config.maxFileSize, $t) }})</p>
                </div>
                <div>
                  <p class="text-xs text-surface-500">{{ $t('homeView.maxUserSize') }}</p>
                  <p class="text-surface-200 font-medium">{{ quotaLabel(auth.user?.maxUserSize, $t) }}</p>
                  <p v-if="!auth.user?.maxUserSize && config.maxUserSize" class="text-xs text-surface-500">({{ quotaLabel(config.maxUserSize, $t) }})</p>
                </div>
                <div>
                  <p class="text-xs text-surface-500">{{ $t('homeView.defaultTTL') }}</p>
                  <p class="text-surface-200 font-medium">{{ ttlLabel(effectiveDefaultTTL, $t) }}</p>
                </div>
                <div>
                  <p class="text-xs text-surface-500">{{ $t('homeView.maxTTL') }}</p>
                  <p class="text-surface-200 font-medium">{{ ttlLabel(auth.user?.maxTTL, $t) }}</p>
                  <p v-if="!auth.user?.maxTTL && config.maxTTL" class="text-xs text-surface-500">({{ ttlLabel(config.maxTTL, $t) }})</p>
                </div>
              </div>
            </div>

            <!-- User Statistics -->
            <div class="glass-card p-5">
              <h3 class="text-sm text-surface-400 uppercase tracking-wider mb-4">{{ $t('homeView.userStatistics') }}</h3>
              <div v-if="userStats" class="grid grid-cols-3 gap-6 text-center">
                <div>
                  <p class="text-2xl font-bold text-surface-200">{{ userStats.uploads }}</p>
                  <p class="text-xs text-surface-500">{{ $t('homeView.uploads') }}</p>
                </div>
                <div>
                  <p class="text-2xl font-bold text-surface-200">{{ userStats.files }}</p>
                  <p class="text-xs text-surface-500">{{ $t('homeView.files') }}</p>
                </div>
                <div>
                  <p class="text-2xl font-bold text-surface-200">{{ humanReadableSize(userStats.totalSize) }}</p>
                  <p class="text-xs text-surface-500">{{ $t('homeView.totalSize') }}</p>
                </div>
              </div>
              <p v-else class="text-sm text-surface-500 text-center py-2">{{ $t('homeView.noStatsAvailable') }}</p>
            </div>
          </div>
        </template>

        <!-- ─── Uploads View ─── -->
        <template v-if="display === 'uploads'">

          <UploadControls
            :sort-by="uploadsSortBy"
            :sort-order="uploadsSortOrder"
            :badge-filters="badgeFilters"
            :show-extend-t-t-l="isFeatureEnabled('extendTTL')"
            @update:sort-by="changeSortBy"
            @update:sort-order="changeSortOrder"
            @toggle-filter="toggleBadgeFilter"
          >
            <template #active-filters v-if="tokenFilter">
              <div class="flex flex-wrap items-center gap-3">
                <div class="flex items-center gap-1.5 text-surface-300">
                  <svg class="w-3.5 h-3.5 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                          d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
                  </svg>
                  token: <span class="font-mono text-accent-400">{{ tokenFilter.substring(0, 12) }}...</span>
                  <button @click="clearTokenFilter" class="text-surface-500 hover:text-surface-100">×</button>
                </div>
              </div>
            </template>
          </UploadControls>

          <!-- Loading -->
          <div v-if="uploadsLoading && uploads.length === 0"
               class="text-center py-12 text-surface-500">
            {{ $t('homeView.loadingUploads') }}
          </div>

          <!-- Empty state -->
          <div v-else-if="uploads.length === 0"
               class="text-center py-12 text-surface-500">
            {{ $t('homeView.noUploadsYet') }}
          </div>

          <!-- Upload cards -->
          <div class="space-y-3">
            <UploadCard v-for="upload in uploads" :key="upload.id"
                        :upload="upload"
                        :token-label="tokenLabel(upload.token)"
                        @delete="handleDeleteUpload"
                        @filter-token="filterByToken" />
          </div>

          <!-- Load more -->
          <div v-if="uploadsCursor" class="mt-4">
            <button @click="loadUploads(true)"
                    class="w-full glass-card p-3 text-sm text-surface-400 hover:text-surface-100
                           hover:bg-surface-700/30 transition-colors text-center"
                    :disabled="uploadsLoading">
              {{ uploadsLoading ? $t('common.loading') : $t('homeView.loadMoreUploads') }}
            </button>
          </div>
        </template>

        <!-- ─── Tokens View ─── -->
        <template v-if="display === 'tokens'">

          <!-- Create token -->
          <div class="glass-card p-4 mb-4 space-y-3">
            <p class="text-sm text-surface-400 text-center">
              {{ $t('homeView.tokenDescription', { config: '' }) }}
              <span class="font-mono text-surface-300">~/.plikrc</span>
              file.
            </p>
            <div class="flex gap-2">
              <input type="text"
                     v-model="newTokenComment"
                     class="input-field flex-1"
                     :placeholder="$t('homeView.commentOptional')"
                     @keyup.enter="handleCreateToken" />
              <button @click="handleCreateToken"
                      class="btn-primary px-4 text-sm whitespace-nowrap">
                {{ $t('homeView.createToken') }}
              </button>
            </div>
          </div>

          <!-- Loading -->
          <div v-if="tokensLoading && tokens.length === 0"
               class="text-center py-12 text-surface-500">
            {{ $t('homeView.loadingTokens') }}
          </div>

          <!-- Empty state -->
          <div v-else-if="tokens.length === 0"
               class="text-center py-8 text-surface-500">
            {{ $t('homeView.noTokensYet') }}
          </div>

          <!-- Token list -->
          <div class="space-y-2">
            <div v-for="token in tokens" :key="token.token"
                 class="glass-card p-4 flex flex-col sm:flex-row items-start sm:items-center gap-3">
              <!-- Token value -->
              <div class="flex-1 min-w-0 space-y-1">
                <p v-if="token.comment" class="text-sm text-surface-200 truncate">{{ token.comment }}</p>
                <div class="flex items-center gap-2">
                  <button @click="filterByToken(token.token)"
                          class="font-mono text-xs text-accent-400/70 hover:text-accent-300 transition-colors
                                 truncate text-left"
                          :title="$t('homeView.showUploadsForToken')">
                    {{ token.token }}
                  </button>
                  <CopyButton :text="token.token" size="sm" />
                </div>
              </div>
              <!-- Created date -->
              <span class="text-xs text-surface-500 shrink-0">{{ formatDate(token.createdAt) }}</span>
              <!-- Revoke -->
              <button @click="handleDeleteTokenUploads(token)"
                      class="text-xs text-orange-400 hover:text-orange-300
                             border border-orange-500/30
                             rounded-lg px-3 py-1.5 hover:bg-amber-500/10 transition-colors shrink-0"
                      :title="$t('homeView.deleteTokenUploads')">
                {{ $t('homeView.deleteTokenUploads') }}
              </button>
              <button @click="handleRevokeToken(token)"
                      class="text-xs text-red-400 hover:text-red-300 border border-red-500/30
                             rounded-lg px-3 py-1.5 hover:bg-red-500/10 transition-colors shrink-0">
                {{ $t('homeView.revokeToken') }}
              </button>
            </div>
          </div>

          <!-- Load more -->
          <div v-if="tokensCursor" class="mt-4">
            <button @click="loadTokens(true)"
                    class="w-full glass-card p-3 text-sm text-surface-400 hover:text-surface-100
                           hover:bg-surface-700/30 transition-colors text-center"
                    :disabled="tokensLoading">
              {{ tokensLoading ? $t('common.loading') : $t('homeView.loadMoreTokens') }}
            </button>
          </div>
        </template>
      </main>
    </div>

    <!-- ═══════ Confirm Dialog ═══════ -->
    <ConfirmDialog v-if="confirm"
                   :message="confirm.message"
                   @confirm="confirm.action()"
                   @cancel="confirm = null" />
    <!-- ═══════ Edit Account Modal ═══════ -->
    <EditUserModal v-model="showEditAccount"
                   v-model:form="editForm"
                   v-model:ttl-unit="editTTLUnit"
                   :error="editError"
                   :saving="editSaving"
                   :title="$t('homeView.editAccountTitle')"
                   :quota-header="$t('homeView.adminSettings')"
                   :show-quotas="auth.user?.admin"
                   @save="saveEditAccount" />
  </div>
</template>
