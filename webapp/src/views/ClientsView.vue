<script setup>
import { ref, computed, onMounted } from 'vue'
import { getVersion } from '../api.js'

const clients = ref([])
const loading = ref(true)
const error = ref(null)

// OS icon mapping
const osIcons = {
    linux: '🐧',
    darwin: '🍎',
    windows: '🪟',
    freebsd: '😈',
    openbsd: '🐡',
}

// Group clients by OS
const grouped = computed(() => {
    const groups = {}
    for (const c of clients.value) {
        const os = c.os || 'other'
        if (!groups[os]) groups[os] = []
        groups[os].push(c)
    }
    // Sort: linux first, then darwin, windows, rest alphabetically
    const order = ['linux', 'darwin', 'windows']
    const sorted = {}
    for (const os of order) {
        if (groups[os]) sorted[os] = groups[os]
    }
    for (const os of Object.keys(groups).sort()) {
        if (!sorted[os]) sorted[os] = groups[os]
    }
    return sorted
})

function clientDownloadURL(client) {
    return `/${client.path}`
}

function osLabel(os) {
    return os.charAt(0).toUpperCase() + os.slice(1)
}

onMounted(async () => {
    try {
        const info = await getVersion()
        clients.value = info.clients || []
    } catch (err) {
        error.value = err.message || $t('clientsView.failedToLoadClients')
    } finally {
        loading.value = false
    }
})
</script>

<template>
  <div class="w-full max-w-screen-md mx-auto px-4 sm:px-6 py-10">

    <!-- Header -->
    <div class="text-center mb-8">
      <h1 class="text-2xl font-bold text-surface-100">{{ $t('clientsView.title') }}</h1>
      <p class="text-sm text-surface-400 mt-2">
        {{ $t('clientsView.subtitle') }}
      </p>
    </div>

    <!-- Quick Start -->
    <div class="mb-8 glass-card p-5 text-sm text-surface-400 space-y-2">
      <p class="font-medium text-surface-300">{{ $t('clientsView.quickStart') }}</p>
      <p>{{ $t('clientsView.quickStartInstructions') }}</p>
      <pre class="bg-surface-900/80 rounded-lg px-4 py-3 text-xs font-mono text-surface-300 overflow-x-auto">chmod +x plik
./plik file1 file2 ...</pre>
      <i18n-t keypath="clientsView.quickStartConfig" tag="p">
        <template #config><code class="text-accent-400">~/.plikrc</code></template>
      </i18n-t>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-16 text-surface-500">
      {{ $t('clientsView.loadingClients') }}
    </div>

    <!-- Error -->
    <div v-else-if="error" class="glass-card p-6 text-center">
      <p class="text-red-400 text-sm">{{ error }}</p>
    </div>

    <!-- Empty -->
    <div v-else-if="clients.length === 0" class="glass-card p-6 text-center">
      <p class="text-surface-400 text-sm">{{ $t('clientsView.noClientsAvailable') }}</p>
    </div>

    <!-- Client list grouped by OS -->
    <div v-else class="space-y-6">
      <div v-for="(osClients, os) in grouped" :key="os" class="glass-card overflow-hidden">

        <!-- OS header -->
        <div class="px-5 py-3 border-b border-surface-700/50 flex items-center gap-2">
          <span class="text-lg">{{ osIcons[os] || '💻' }}</span>
          <h2 class="text-sm font-semibold text-surface-200 uppercase tracking-wider">
            {{ osLabel(os) }}
          </h2>
        </div>

        <!-- Clients for this OS -->
        <div class="divide-y divide-surface-700/30">
          <div v-for="client in osClients" :key="client.name"
               class="flex items-center justify-between px-5 py-3 hover:bg-surface-800/40 transition-colors">

            <!-- Info -->
            <div class="min-w-0">
              <p class="text-sm text-surface-200 font-medium truncate">{{ client.name }}</p>
              <p class="text-xs text-surface-500 font-mono mt-0.5 truncate">
                {{ client.arch }}
                <span v-if="client.md5" class="ml-3 hidden sm:inline">md5: {{ client.md5 }}</span>
              </p>
            </div>

            <!-- Download button -->
            <a :href="clientDownloadURL(client)"
               class="shrink-0 ml-4 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium
                      bg-accent-500/15 text-accent-400 border border-accent-500/30
                      hover:bg-accent-500/25 hover:text-accent-300 transition-colors">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              {{ $t('common.download') }}
            </a>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
