<script setup>
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { auth, impersonate as doImpersonate, clearImpersonate } from '../authStore.js'
import { config } from '../config.js'
import ErrorBanner from '../components/ErrorBanner.vue'
import UploadControls from '../components/UploadControls.vue'
import {
    getServerStats, getAdminUsers, getAdminUploads, searchUsers,
    createUser as apiCreateUser, deleteUser as apiDeleteUser,
    updateUser, removeUpload, getVersion
} from '../api.js'
import {
    humanReadableSize, quotaLabel, ttlLabel,
    buildEditForm, buildEditPayload,
    clampQuota, filterQuotaInput, defaultSizeHint, defaultTTLHint, TTL_UNITS,
} from '../utils.js'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import EditUserModal from '../components/EditUserModal.vue'
import UploadCard from '../components/UploadCard.vue'

const router = useRouter()
const route = useRoute()
const { t: $t } = useI18n()

// ── Display mode ──
const display = computed(() => route.params.tab || 'stats')

// ── Version info ──
const version = ref(null)

// ── Stats ──
const stats = ref(null)
const statsLoading = ref(false)

// ── Users ──
const users = ref([])
const usersCursor = ref(null)
const usersTotal = ref(null)
const usersLoading = ref(false)
const usersProviderFilter = ref('')
const usersAdminFilter = ref('')   // '' | 'true' | 'false'
const usersSortBy = ref('date')    // 'date' | 'size'
const usersSortOrder = ref('desc') // 'desc' | 'asc'

// ── User search ──
const usersSearchQuery = ref('')
const usersSearchResults = ref([])
const usersSearchOpen = ref(false)
let usersSearchTimer = null

// ── Uploads ──
const uploads = ref([])
const uploadsCursor = ref(null)
const uploadsTotal = ref(null)
const uploadsLoading = ref(false)
const uploadsUserFilter = ref('')
const uploadsTokenFilter = ref('')
const uploadsSortBy = ref('date') // 'date' | 'size'
const uploadsSortOrder = ref('desc') // 'desc' | 'asc'

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

// ── Create user modal ──
const showCreateUser = ref(false)
const createForm = ref({ provider: 'local', login: '', password: '', name: '', email: '', admin: false, maxFileSize: 0, maxUserSize: 0, maxTTL: 0 })
const createTTLUnit = ref(60)
const createError = ref('')
const createSaving = ref(false)

// ── Edit user modal ──
const showEditUser = ref(false)
const editForm = ref({})
const editError = ref('')
const editSaving = ref(false)

// ── Confirm dialog ──
const confirm = ref(null)
const error = ref(null)

// ── Helpers ──

// Edit form display helpers
const editTTLUnit = ref(60)

// ── Stats API ──
async function loadStats() {
    statsLoading.value = true
    try {
        stats.value = await getServerStats()
    } catch (err) {
        error.value = $t('adminView.failedToLoadStats')
    } finally {
        statsLoading.value = false
    }
}

// ── Users API ──
async function loadUsers(more = false) {
    usersLoading.value = true
    try {
        const opts = {
            limit: 50,
            sort: usersSortBy.value,
            order: usersSortOrder.value,
        }
        if (usersProviderFilter.value) opts.provider = usersProviderFilter.value
        if (usersAdminFilter.value) opts.admin = usersAdminFilter.value
        if (more && usersCursor.value) opts.after = usersCursor.value
        const data = await getAdminUsers(opts)
        if (more) {
            users.value = [...users.value, ...data.results]
        } else {
            users.value = data.results || []
        }
        usersCursor.value = data.after || null
        usersTotal.value = data.total ?? null
    } catch (err) {
        error.value = $t('adminView.failedToLoadUsers')
    } finally {
        usersLoading.value = false
    }
}

function changeUsersProviderFilter(val) {
    usersProviderFilter.value = val
    users.value = []
    usersCursor.value = null
    internalNav = true
    router.replace({ query: currentUsersQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUsers()
}

function changeUsersAdminFilter(val) {
    usersAdminFilter.value = val
    users.value = []
    usersCursor.value = null
    internalNav = true
    router.replace({ query: currentUsersQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUsers()
}

function changeUsersSortBy(val) {
    usersSortBy.value = val
    users.value = []
    usersCursor.value = null
    internalNav = true
    router.replace({ query: currentUsersQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUsers()
}

function changeUsersSortOrder(val) {
    usersSortOrder.value = val
    users.value = []
    usersCursor.value = null
    internalNav = true
    router.replace({ query: currentUsersQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUsers()
}

// ── User search ──
function onUsersSearchInput() {
    clearTimeout(usersSearchTimer)
    const q = usersSearchQuery.value.trim()
    if (q.length < 2) {
        usersSearchResults.value = []
        usersSearchOpen.value = false
        return
    }
    usersSearchTimer = setTimeout(async () => {
        try {
            const opts = { q, limit: 5 }
            if (usersProviderFilter.value) opts.provider = usersProviderFilter.value
            if (usersAdminFilter.value) opts.admin = usersAdminFilter.value
            usersSearchResults.value = await searchUsers(opts)
            usersSearchOpen.value = usersSearchResults.value.length > 0
        } catch (e) {
            usersSearchResults.value = []
            usersSearchOpen.value = false
        }
    }, 300)
}

function selectSearchResult(user) {
    usersSearchQuery.value = ''
    usersSearchResults.value = []
    usersSearchOpen.value = false
    users.value = [user]
    usersCursor.value = null
}

function closeSearch() {
    usersSearchOpen.value = false
}

// ── Uploads API ──
async function loadUploads(more = false) {
    uploadsLoading.value = true
    try {
        const opts = {
            limit: 50,
            sort: uploadsSortBy.value,
            order: uploadsSortOrder.value,
        }
        if (uploadsUserFilter.value) opts.user = uploadsUserFilter.value
        if (uploadsTokenFilter.value) opts.token = uploadsTokenFilter.value
        if (more && uploadsCursor.value) opts.after = uploadsCursor.value
        // Badge filters
        for (const key of BADGE_FILTER_KEYS) {
            if (badgeFilters.value[key]) opts[key] = true
        }
        const data = await getAdminUploads(opts)
        if (more) {
            uploads.value = [...uploads.value, ...data.results]
        } else {
            uploads.value = data.results || []
        }
        uploadsCursor.value = data.after || null
        uploadsTotal.value = data.total ?? null
    } catch (err) {
        error.value = $t('adminView.failedToLoadUploads')
    } finally {
        uploadsLoading.value = false
    }
}

function filterUploadsByUser(userId) {
    uploadsUserFilter.value = userId
    uploadsTokenFilter.value = ''
    uploads.value = []
    uploadsCursor.value = null
    internalNav = true
    router.push({ path: '/admin/uploads', query: currentUploadsQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUploads()
}

async function viewUserInUsersTab(userId) {
    try {
        const results = await searchUsers({ q: userId, limit: 1 })
        if (results.length > 0) {
            users.value = [results[0]]
            usersCursor.value = null
            internalNav = true
            router.push('/admin/users')
                .finally(() => nextTick(() => { internalNav = false }))
        }
    } catch {
        error.value = $t('adminView.failedToFindUser')
    }
}

function filterUploadsByToken(token) {
    uploadsTokenFilter.value = token
    uploads.value = []
    uploadsCursor.value = null
    // Token NOT in URL for security — only update tab and keep user filter if set
    internalNav = true
    router.push({ path: '/admin/uploads', query: currentUploadsQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUploads()
}

function clearUserFilter() {
    uploadsUserFilter.value = ''
    uploads.value = []
    uploadsCursor.value = null
    internalNav = true
    router.replace({ query: currentUploadsQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUploads()
}

function clearTokenFilter() {
    uploadsTokenFilter.value = ''
    uploads.value = []
    uploadsCursor.value = null
    // Token was never in URL, so just reload
    loadUploads()
}

function changeSortBy(val) {
    uploadsSortBy.value = val
    uploads.value = []
    uploadsCursor.value = null
    internalNav = true
    router.replace({ query: currentUploadsQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUploads()
}

function changeSortOrder(val) {
    uploadsSortOrder.value = val
    uploads.value = []
    uploadsCursor.value = null
    internalNav = true
    router.replace({ query: currentUploadsQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUploads()
}

function toggleBadgeFilter(key) {
    badgeFilters.value[key] = !badgeFilters.value[key]
    uploads.value = []
    uploadsCursor.value = null
    internalNav = true
    router.push({ query: currentUploadsQuery() })
        .finally(() => nextTick(() => { internalNav = false }))
    loadUploads() // Intentional: reads from badgeFilters.value, not URL, so safe before push resolves
}

// ── User management ──
function openCreateUser() {
    createForm.value = { provider: 'local', login: '', password: '', name: '', email: '', admin: false, maxFileSize: 0, maxUserSize: 0, maxTTL: 0 }
    createTTLUnit.value = 60
    createError.value = ''
    showCreateUser.value = true
}

async function submitCreateUser() {
    createSaving.value = true
    createError.value = ''
    try {
        const payload = buildEditPayload(createForm.value, createTTLUnit.value)
        const u = await apiCreateUser(payload)
        users.value = [u, ...users.value]
        showCreateUser.value = false
    } catch (err) {
        createError.value = err.message || $t('adminView.failedToCreateUser')
    } finally {
        createSaving.value = false
    }
}

function openEditUser(user) {
    const { form, ttlUnit } = buildEditForm(user)
    editForm.value = form
    editTTLUnit.value = ttlUnit
    editError.value = ''
    showEditUser.value = true
}

async function submitEditUser() {
    editSaving.value = true
    editError.value = ''
    try {
        const payload = buildEditPayload(editForm.value, editTTLUnit.value)
        const updated = await updateUser(payload)
        const idx = users.value.findIndex(u => u.id === updated.id)
        if (idx >= 0) users.value[idx] = updated
        showEditUser.value = false
    } catch (err) {
        editError.value = err.message || $t('adminView.failedToUpdateUser')
    } finally {
        editSaving.value = false
    }
}

function handleDeleteUser(user) {
    confirm.value = {
        message: $t('adminView.deleteUserConfirm', { login: user.login, provider: user.provider }),
        action: async () => {
            try {
                await apiDeleteUser(user.id)
                users.value = users.value.filter(u => u.id !== user.id)
            } catch (err) {
                error.value = $t('adminView.failedToDeleteUser')
            }
            confirm.value = null
        }
    }
}

function handleDeleteUpload(upload) {
    confirm.value = {
        message: $t('adminView.deleteUploadConfirm', { id: upload.id }),
        action: async () => {
            try {
                await removeUpload(upload.id, upload.uploadToken)
                uploads.value = uploads.value.filter(u => u.id !== upload.id)
            } catch (err) {
                error.value = $t('adminView.failedToDeleteUpload')
            }
            confirm.value = null
        }
    }
}

// ── Display switching (via route path) ──
function showStatsView() {
    router.push('/admin/stats')
}

function showUsersView() {
    router.push('/admin/users')
}

function showUploadsView() {
    router.push('/admin/uploads')
}

// ── Route helpers ──

// Build query params from current users tab filter state (omits defaults)
function currentUsersQuery() {
    return {
        provider: usersProviderFilter.value || undefined,
        admin: usersAdminFilter.value || undefined,
        sort: usersSortBy.value !== 'date' ? usersSortBy.value : undefined,
        order: usersSortOrder.value !== 'desc' ? usersSortOrder.value : undefined,
    }
}

// Build query params from current uploads tab filter state (omits defaults, excludes token)
function currentUploadsQuery() {
    const q = {
        user: uploadsUserFilter.value || undefined,
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

// Tab changes (path-based) — triggers on back/forward between tabs
watch(display, (tab, prevTab) => {
    if (tab === prevTab || internalNav) return
    if (tab === 'stats') {
        loadStats()
    } else if (tab === 'users') {
        // Sync filters from query params (e.g. back/forward within users tab)
        usersProviderFilter.value = route.query.provider || ''
        usersAdminFilter.value = route.query.admin || ''
        usersSortBy.value = route.query.sort || 'date'
        usersSortOrder.value = route.query.order || 'desc'
        users.value = []
        usersCursor.value = null
        loadUsers()
    } else if (tab === 'uploads') {
        uploadsUserFilter.value = route.query.user || ''
        uploadsTokenFilter.value = ''  // never from URL
        uploadsSortBy.value = route.query.sort || 'date'
        uploadsSortOrder.value = route.query.order || 'desc'
        for (const key of BADGE_FILTER_KEYS) {
            badgeFilters.value[key] = route.query[key] === 'true'
        }
        uploads.value = []
        uploadsCursor.value = null
        loadUploads()
    }
})

// Filter changes within same tab (query-based) — triggers on back/forward within a tab
watch(() => route.query, (query, oldQuery) => {
    // Only react if still on the same tab (path didn't change)
    if (internalNav || !route.params.tab) return
    const tab = route.params.tab

    if (tab === 'users') {
        const changed = query.provider !== oldQuery?.provider ||
                        query.admin !== oldQuery?.admin ||
                        query.sort !== oldQuery?.sort ||
                        query.order !== oldQuery?.order
        if (changed) {
            usersProviderFilter.value = query.provider || ''
            usersAdminFilter.value = query.admin || ''
            usersSortBy.value = query.sort || 'date'
            usersSortOrder.value = query.order || 'desc'
            users.value = []
            usersCursor.value = null
            loadUsers()
        }
    } else if (tab === 'uploads') {
        const changed = query.user !== oldQuery?.user ||
                        query.sort !== oldQuery?.sort ||
                        query.order !== oldQuery?.order ||
                        BADGE_FILTER_KEYS.some(k => query[k] !== oldQuery?.[k])
        if (changed) {
            uploadsUserFilter.value = query.user || ''
            uploadsTokenFilter.value = ''  // never from URL
            uploadsSortBy.value = query.sort || 'date'
            uploadsSortOrder.value = query.order || 'desc'
            for (const key of BADGE_FILTER_KEYS) {
                badgeFilters.value[key] = query[key] === 'true'
            }
            uploads.value = []
            uploadsCursor.value = null
            loadUploads()
        }
    }
})

// ── Init ──
onMounted(async () => {
    if (!auth.user || !auth.user.admin) {
        router.push('/')
        return
    }
    try {
        version.value = await getVersion()
    } catch (err) {
        error.value = $t('adminView.failedToLoadVersion')
    }

    // Initialize from URL
    const tab = display.value

    if (tab === 'users') {
        usersProviderFilter.value = route.query.provider || ''
        usersAdminFilter.value = route.query.admin || ''
        usersSortBy.value = route.query.sort || 'date'
        usersSortOrder.value = route.query.order || 'desc'
        loadUsers()
    } else if (tab === 'uploads') {
        uploadsUserFilter.value = route.query.user || ''
        uploadsSortBy.value = route.query.sort || 'date'
        uploadsSortOrder.value = route.query.order || 'desc'
        // Restore badge filters from URL
        for (const key of BADGE_FILTER_KEYS) {
            badgeFilters.value[key] = route.query[key] === 'true'
        }
        loadUploads()
    } else {
        loadStats()
    }
})
</script>

<template>
  <div class="w-full max-w-screen-2xl mx-auto px-4 sm:px-6 py-6">
    <div class="flex flex-col md:flex-row gap-6">

      <!-- ═══════ Sidebar ═══════ -->
      <aside class="w-full md:w-72 shrink-0 space-y-4">

        <!-- Server Info Card -->
        <div class="glass-card p-5 text-center space-y-2">
          <p class="text-surface-200 font-medium">{{ $t('adminView.plikServer') }}</p>
          <p v-if="version" class="text-xs text-surface-500 font-mono">
            v{{ version.version }}
          </p>
          <p v-if="version" class="text-xs text-surface-500">
            {{ version.goVersion }}
          </p>
          <div v-if="version" class="flex items-center justify-center gap-2 pt-1">
            <span :class="version.isRelease
              ? 'bg-emerald-500/20 text-emerald-400'
              : 'bg-red-500/20 text-red-400'"
                  class="text-xs px-2 py-0.5 rounded-full">release</span>
            <span :class="version.isMint
              ? 'bg-emerald-500/20 text-emerald-400'
              : 'bg-red-500/20 text-red-400'"
                  class="text-xs px-2 py-0.5 rounded-full">mint</span>
          </div>
        </div>

        <!-- Nav Buttons -->
        <div class="glass-card p-2 space-y-1">
          <button @click="showStatsView"
                  :class="display === 'stats'
                    ? 'bg-accent-500/10 text-accent-400 border-l-2 border-accent-400'
                    : 'text-surface-300 hover:text-surface-100 hover:bg-surface-700/50 border-l-2 border-transparent'"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            {{ $t('adminView.stats') }}
          </button>

          <button @click="showUploadsView"
                  :class="display === 'uploads'
                    ? 'bg-accent-500/10 text-accent-400 border-l-2 border-accent-400'
                    : 'text-surface-300 hover:text-surface-100 hover:bg-surface-700/50 border-l-2 border-transparent'"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
            {{ $t('adminView.uploads') }}
          </button>

          <button @click="showUsersView"
                  :class="display === 'users'
                    ? 'bg-accent-500/10 text-accent-400 border-l-2 border-accent-400'
                    : 'text-surface-300 hover:text-surface-100 hover:bg-surface-700/50 border-l-2 border-transparent'"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
            </svg>
            {{ $t('adminView.users') }}
          </button>
        </div>

        <!-- Create User -->
        <div class="glass-card p-2">
          <button @click="openCreateUser"
                  class="w-full py-2.5 rounded-lg flex items-center gap-3 px-3 text-sm
                         text-surface-300 hover:text-surface-100 hover:bg-surface-700/50 transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
            </svg>
            {{ $t('adminView.createUser') }}
          </button>
        </div>
      </aside>

      <!-- ═══════ Main Content ═══════ -->
      <main class="flex-1 min-w-0">

        <!-- Error Banner -->
        <ErrorBanner v-if="error" :message="error" @dismiss="error = null" class="mb-4" />

        <!-- ─── Stats View ─── -->
        <template v-if="display === 'stats'">
          <div v-if="statsLoading" class="text-center py-12 text-surface-500">{{ $t('adminView.loadingStats') }}</div>

          <div v-else-if="stats" class="space-y-4">
            <!-- Server Config -->
            <div class="glass-card p-5">
              <h3 class="text-sm text-surface-400 uppercase tracking-wider mb-4">{{ $t('adminView.serverConfiguration') }}</h3>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-center">
                <div>
                  <p class="text-xs text-surface-500">{{ $t('adminView.maxFileSize') }}</p>
                  <p class="text-surface-200 font-medium">{{ quotaLabel(config.maxFileSize, $t) }}</p>
                </div>
                <div>
                  <p class="text-xs text-surface-500">{{ $t('adminView.maxUserSize') }}</p>
                  <p class="text-surface-200 font-medium">{{ quotaLabel(config.maxUserSize, $t) }}</p>
                </div>
                <div>
                  <p class="text-xs text-surface-500">{{ $t('adminView.defaultTTL') }}</p>
                  <p class="text-surface-200 font-medium">{{ ttlLabel(config.defaultTTL, $t) }}</p>
                </div>
                <div>
                  <p class="text-xs text-surface-500">{{ $t('adminView.maxTTL') }}</p>
                  <p class="text-surface-200 font-medium">{{ ttlLabel(config.maxTTL, $t) }}</p>
                </div>
              </div>
            </div>

            <!-- Server Stats -->
            <div class="glass-card p-5">
              <h3 class="text-sm text-surface-400 uppercase tracking-wider mb-4">{{ $t('adminView.serverStatistics') }}</h3>
              <div class="grid grid-cols-2 sm:grid-cols-3 gap-6 text-center">
                <div>
                  <p class="text-2xl font-bold text-surface-200">{{ stats.users }}</p>
                  <p class="text-xs text-surface-500">{{ $t('adminView.usersLabel') }}</p>
                </div>
                <div>
                  <p class="text-2xl font-bold text-surface-200">{{ stats.uploads }}</p>
                  <p class="text-xs text-surface-500">{{ $t('adminView.uploadsLabel') }}</p>
                </div>
                <div>
                  <p class="text-2xl font-bold text-surface-200">{{ stats.anonymousUploads }}</p>
                  <p class="text-xs text-surface-500">{{ $t('adminView.anonymousUploads') }}</p>
                </div>
                <div>
                  <p class="text-2xl font-bold text-surface-200">{{ stats.files }}</p>
                  <p class="text-xs text-surface-500">{{ $t('adminView.files') }}</p>
                </div>
                <div>
                  <p class="text-2xl font-bold text-surface-200">{{ humanReadableSize(stats.totalSize) }}</p>
                  <p class="text-xs text-surface-500">{{ $t('adminView.totalSize') }}</p>
                </div>
                <div>
                  <p class="text-2xl font-bold text-surface-200">{{ humanReadableSize(stats.anonymousTotalSize) }}</p>
                  <p class="text-xs text-surface-500">{{ $t('adminView.anonymousSize') }}</p>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- ─── Users View ─── -->
        <template v-if="display === 'users'">

          <!-- User search -->
          <div class="relative mb-4">
            <input v-model="usersSearchQuery"
                   @input="onUsersSearchInput"
                   @keydown.escape="closeSearch"
                   type="text"
                   :placeholder="$t('adminView.searchUsersPlaceholder')"
                   class="input-field rounded-xl!
                          px-4! py-2.5!" />
            <!-- Search dropdown -->
            <div v-if="usersSearchOpen"
                 class="absolute z-20 top-full mt-1 w-full glass-card
                        overflow-hidden">
              <button v-for="u in usersSearchResults" :key="u.id"
                      @click="selectSearchResult(u)"
                      class="w-full text-left px-4 py-2.5 text-sm hover:bg-accent-500/10
                             transition-colors flex items-center gap-3">
                <span class="text-accent-400 font-mono text-xs truncate max-w-[120px]"
                      :title="u.id">{{ u.login || u.id }}</span>
                <span v-if="u.name" class="text-surface-300 truncate">{{ u.name }}</span>
                <span v-if="u.email" class="text-surface-500 text-xs truncate ml-auto">{{ u.email }}</span>
                <span v-if="u.admin"
                      class="text-xs bg-emerald-500/20 text-emerald-400 px-1.5 py-0.5 rounded-full">admin</span>
              </button>
            </div>
          </div>

          <!-- Sort / filter controls -->
          <div class="glass-card p-3 mb-4 space-y-2 text-sm">
            <div class="flex flex-wrap items-center gap-4">
              <!-- Sort by -->
              <div class="flex items-center gap-2 text-surface-400">
                <span>{{ $t('adminView.sort') }}</span>
                <button @click="changeUsersSortBy('date')"
                        :class="usersSortBy === 'date' ? 'text-accent-400' : 'text-surface-500 hover:text-surface-300'"
                        class="transition-colors">{{ $t('adminView.date') }}</button>
              </div>
              <!-- Order -->
              <div class="flex items-center gap-2 text-surface-400">
                <span>{{ $t('adminView.order') }}</span>
                <button @click="changeUsersSortOrder('desc')"
                        :class="usersSortOrder === 'desc' ? 'text-accent-400' : 'text-surface-500 hover:text-surface-300'"
                        class="transition-colors">{{ $t('adminView.desc') }}</button>
                <span class="text-surface-600">|</span>
                <button @click="changeUsersSortOrder('asc')"
                        :class="usersSortOrder === 'asc' ? 'text-accent-400' : 'text-surface-500 hover:text-surface-300'"
                        class="transition-colors">{{ $t('adminView.asc') }}</button>
              </div>
            </div>

            <!-- Provider filter -->
            <div class="flex flex-wrap items-center gap-2 text-surface-400">
              <span>{{ $t('adminView.provider') }}</span>
              <button v-for="p in ['', 'local', 'google', 'ovh', 'oidc']" :key="p"
                      @click="changeUsersProviderFilter(p)"
                      :class="usersProviderFilter === p ? 'text-accent-400 bg-accent-500/10' : 'text-surface-500 hover:text-surface-300'"
                      class="px-2 py-0.5 rounded-full text-xs transition-colors">
                {{ p || $t('adminView.all') }}
              </button>
            </div>

            <!-- Admin filter -->
            <div class="flex flex-wrap items-center gap-2 text-surface-400">
              <span>{{ $t('adminView.role') }}</span>
              <button v-for="a in ['', 'true', 'false']" :key="a"
                      @click="changeUsersAdminFilter(a)"
                      :class="usersAdminFilter === a ? 'text-accent-400 bg-accent-500/10' : 'text-surface-500 hover:text-surface-300'"
                      class="px-2 py-0.5 rounded-full text-xs transition-colors">
                {{ a === '' ? $t('adminView.all') : a === 'true' ? $t('common.admin') : $t('adminView.nonAdmin') }}
              </button>
            </div>
          </div>

          <p v-if="usersTotal !== null" class="text-xs text-surface-500 mb-2">
            {{ $t('adminView.showingUsersOf', { shown: users.length, total: usersTotal }) }}
          </p>

          <div v-if="usersLoading && users.length === 0" class="text-center py-12 text-surface-500">
            {{ $t('adminView.loadingUsers') }}
          </div>

          <div v-else-if="users.length === 0" class="text-center py-12 text-surface-500">
            {{ $t('adminView.noUsers') }}
          </div>

          <div class="space-y-3">
            <div v-for="user in users" :key="user.id" class="glass-card p-4">
              <div class="flex flex-col sm:flex-row gap-4">
                <!-- User info -->
                <div class="sm:w-1/4 text-sm space-y-1">
                  <div class="flex items-center gap-2">
                    <img v-if="user.profilePicture"
                         :src="user.profilePicture"
                         alt=""
                         class="w-6 h-6 rounded-full object-cover shrink-0"
                         referrerpolicy="no-referrer" />
                    <p class="text-surface-200 font-medium">{{ user.login }}</p>
                  </div>
                  <p class="text-surface-500">({{ user.provider }})</p>
                  <span v-if="user.admin"
                        class="inline-block text-xs bg-emerald-500/20 text-emerald-400 px-2 py-0.5 rounded-full">
                    admin
                  </span>
                  <p v-if="user.createdAt" class="text-xs text-surface-600" :title="new Date(user.createdAt).toLocaleString()">
                    {{ new Date(user.createdAt).toLocaleDateString() }}
                  </p>
                </div>

                <!-- Name / Email -->
                <div class="sm:w-1/4 text-sm space-y-1">
                  <p v-if="user.name" class="text-surface-300">{{ user.name }}</p>
                  <p v-if="user.email" class="text-surface-400">{{ user.email }}</p>
                </div>

                <!-- Quotas -->
                <div class="sm:w-1/4 text-xs text-surface-500 space-y-1">
                  <p>{{ $t('adminView.maxFileSizeLabel', { value: quotaLabel(user.maxFileSize, $t) }) }}</p>
                  <p>{{ $t('adminView.maxUserSizeLabel', { value: quotaLabel(user.maxUserSize, $t) }) }}</p>
                  <p>{{ $t('adminView.maxTTLLabel', { value: ttlLabel(user.maxTTL, $t) }) }}</p>
                </div>

                <!-- Actions -->
                <div class="sm:w-1/4 flex flex-wrap items-center justify-end gap-2">
                  <button @click="filterUploadsByUser(user.id)"
                          class="text-xs text-surface-400 hover:text-surface-200 border border-surface-600/50
                                 rounded-lg px-3 py-1.5 hover:bg-surface-700/50 transition-colors"
                          :title="$t('adminView.viewUploads')">
                    📁
                  </button>
                  <button @click="doImpersonate(user)"
                          :disabled="user.id === auth.originalUser?.id"
                          :class="user.id === auth.originalUser?.id ? 'opacity-30 cursor-not-allowed' : 'hover:text-green-300 hover:bg-green-500/10'"
                          class="text-xs text-green-400 border border-green-500/30
                                 rounded-lg px-3 py-1.5 transition-colors"
                          :title="$t('adminView.impersonate')">
                    👤
                  </button>
                  <button @click="openEditUser(user)"
                          class="text-xs text-accent-400 hover:text-accent-300 border border-accent-500/30
                                 rounded-lg px-3 py-1.5 hover:bg-accent-500/10 transition-colors">
                    {{ $t('adminView.edit') }}
                  </button>
                  <button @click="handleDeleteUser(user)"
                          :disabled="user.id === auth.user?.id"
                          :class="user.id === auth.user?.id ? 'opacity-30 cursor-not-allowed' : 'hover:text-red-300 hover:bg-red-500/10'"
                          class="text-xs text-red-400 border border-red-500/30
                                 rounded-lg px-3 py-1.5 transition-colors">
                    {{ $t('common.delete') }}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Load more -->
          <div v-if="usersCursor" class="mt-4">
            <button @click="loadUsers(true)"
                    class="w-full glass-card p-3 text-sm text-surface-400 hover:text-surface-100
                           hover:bg-surface-700/30 transition-colors text-center"
                    :disabled="usersLoading">
              {{ usersLoading ? $t('common.loading') : $t('adminView.loadMoreUsers') }}
            </button>
          </div>
        </template>

        <!-- ─── Uploads View ─── -->
        <template v-if="display === 'uploads'">

          <UploadControls
            :sort-by="uploadsSortBy"
            :sort-order="uploadsSortOrder"
            :badge-filters="badgeFilters"
            :show-extend-t-t-l="true"
            @update:sort-by="changeSortBy"
            @update:sort-order="changeSortOrder"
            @toggle-filter="toggleBadgeFilter"
          >
            <template #active-filters v-if="uploadsUserFilter || uploadsTokenFilter">
              <div class="flex flex-wrap items-center gap-3">
                <div v-if="uploadsUserFilter" class="flex items-center gap-1.5 text-surface-300">
                  <svg class="w-3.5 h-3.5 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                          d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
                  </svg>
                  user: <span class="font-mono text-accent-400">{{ uploadsUserFilter }}</span>
                  <button @click="viewUserInUsersTab(uploadsUserFilter)"
                          class="text-surface-400 hover:text-accent-400 transition-colors"
                          :title="$t('adminView.viewUserInUsersTab')">🔍</button>
                  <button @click="clearUserFilter" class="text-surface-500 hover:text-surface-100">×</button>
                </div>
                <div v-if="uploadsTokenFilter" class="flex items-center gap-1.5 text-surface-300">
                  <svg class="w-3.5 h-3.5 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                          d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
                  </svg>
                  token: <span class="font-mono text-accent-400">{{ uploadsTokenFilter.substring(0, 12) }}...</span>
                  <button @click="clearTokenFilter" class="text-surface-500 hover:text-surface-100">×</button>
                </div>
              </div>
            </template>
          </UploadControls>

          <p v-if="uploadsTotal !== null" class="text-xs text-surface-500 mb-2">
            {{ $t('adminView.showingUploadsOf', { shown: uploads.length, total: uploadsTotal }) }}
          </p>

          <div v-if="uploadsLoading && uploads.length === 0" class="text-center py-12 text-surface-500">
            {{ $t('adminView.loadingUploads') }}
          </div>

          <div v-else-if="uploads.length === 0" class="text-center py-12 text-surface-500">
            {{ $t('adminView.noUploads') }}
          </div>

          <div class="space-y-3">
            <UploadCard v-for="upload in uploads" :key="upload.id"
                        :upload="upload"
                        :show-user="true"
                        @delete="handleDeleteUpload"
                        @filter-token="filterUploadsByToken"
                        @filter-user="filterUploadsByUser" />
          </div>

          <!-- Load more -->
          <div v-if="uploadsCursor" class="mt-4">
            <button @click="loadUploads(true)"
                    class="w-full glass-card p-3 text-sm text-surface-400 hover:text-surface-100
                           hover:bg-surface-700/30 transition-colors text-center"
                    :disabled="uploadsLoading">
              {{ uploadsLoading ? $t('common.loading') : $t('adminView.loadMoreUploads') }}
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

    <!-- ═══════ Create User Modal ═══════ -->
    <Teleport to="body">
      <div v-if="showCreateUser"
           class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
           @mousedown.self="showCreateUser = false">
        <div class="glass-card p-4 sm:p-6 max-w-md w-full space-y-5 animate-fade-in max-h-[90vh] overflow-y-auto">
          <h2 class="text-lg font-semibold text-surface-200">{{ $t('adminView.createUserTitle') }}</h2>

          <div v-if="createError" class="text-sm text-red-400 bg-red-500/10 rounded-lg px-3 py-2">
            {{ createError }}
          </div>

          <!-- Provider -->
          <div>
            <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.provider') }}</label>
            <select v-model="createForm.provider" class="input-field w-full">
              <option value="local">local</option>
              <option value="google">google</option>
              <option value="ovh">ovh</option>
              <option value="oidc">oidc</option>
            </select>
          </div>

          <!-- Login -->
          <div>
            <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.login') }}</label>
            <input type="text" v-model="createForm.login" class="input-field w-full" :placeholder="$t('adminView.loginPlaceholder')" />
          </div>

          <!-- Password (local only) -->
          <div v-if="createForm.provider === 'local'">
            <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.password') }}</label>
            <input type="password" v-model="createForm.password" class="input-field w-full" :placeholder="$t('adminView.passwordPlaceholder')" />
          </div>

          <!-- Name -->
          <div>
            <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.name') }}</label>
            <input type="text" v-model="createForm.name" class="input-field w-full" :placeholder="$t('adminView.optional')" />
          </div>

          <!-- Email -->
          <div>
            <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.email') }}</label>
            <input type="email" v-model="createForm.email" class="input-field w-full" :placeholder="$t('adminView.optional')" />
          </div>

          <!-- Quotas -->
          <div class="border-t border-surface-700/50 pt-4 space-y-4">
            <p class="text-xs text-surface-500 uppercase tracking-wider">{{ $t('adminView.quotas') }}</p>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.maxFileSizeGB') }}</label>
                <input type="text" inputmode="decimal" v-model="createForm.maxFileSize"
                       @input="createForm.maxFileSize = filterQuotaInput($event.target.value, true)"
                       @blur="createForm.maxFileSize = clampQuota(createForm.maxFileSize)"
                       class="input-field w-full" />
                <p class="text-xs text-surface-600 mt-0.5">{{ defaultSizeHint(config.maxFileSize, $t) }}</p>
              </div>
              <div>
                <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.maxUserSizeGB') }}</label>
                <input type="text" inputmode="decimal" v-model="createForm.maxUserSize"
                       @input="createForm.maxUserSize = filterQuotaInput($event.target.value, true)"
                       @blur="createForm.maxUserSize = clampQuota(createForm.maxUserSize)"
                       class="input-field w-full" />
                <p class="text-xs text-surface-600 mt-0.5">{{ defaultSizeHint(config.maxUserSize, $t) }}</p>
              </div>
            </div>
            <div>
              <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.maxTTL') }}</label>
              <div class="flex gap-2">
                <input type="text" inputmode="numeric" v-model="createForm.maxTTL"
                       @input="createForm.maxTTL = filterQuotaInput($event.target.value, false)"
                       @blur="createForm.maxTTL = clampQuota(createForm.maxTTL)"
                       class="input-field flex-1" />
                <select v-model.number="createTTLUnit" class="input-field w-28">
                  <option v-for="u in TTL_UNITS" :key="u.seconds" :value="u.seconds">{{ $t(u.i18nKey) }}</option>
                </select>
              </div>
              <p class="text-xs text-surface-600 mt-0.5">{{ defaultTTLHint(config.maxTTL, $t) }}</p>
            </div>

            <!-- Admin -->
            <label class="flex items-center gap-2 text-sm text-surface-300 cursor-pointer">
              <input type="checkbox" v-model="createForm.admin"
                     class="w-4 h-4 rounded border-surface-600 bg-surface-800
                            text-accent-500 focus:ring-accent-500/30" />
              {{ $t('common.admin') }}
            </label>
          </div>

          <div class="flex justify-end gap-2 pt-2">
            <button @click="showCreateUser = false" class="btn-ghost text-sm px-4 py-2">{{ $t('common.cancel') }}</button>
            <button @click="submitCreateUser" :disabled="createSaving"
                    class="btn-primary px-4 py-2 text-sm">
              {{ createSaving ? $t('adminView.creating') : $t('adminView.create') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- ═══════ Edit User Modal ═══════ -->
    <EditUserModal v-model="showEditUser"
                   v-model:form="editForm"
                   v-model:ttl-unit="editTTLUnit"
                   :error="editError"
                   :saving="editSaving"
                   :title="$t('adminView.editUserTitle')"
                   :quota-header="$t('adminView.quotas')"
                   :show-quotas="true"
                   @save="submitEditUser" />
  </div>
</template>
