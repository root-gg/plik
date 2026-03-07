<script setup>
import { ref, onMounted, computed } from 'vue'
import { getUploadEvents, getAdminEvents } from '../api.js'
import { formatDate } from '../utils.js'

const props = defineProps({
    uploadId: { type: String, default: '' },
    uploadToken: { type: String, default: '' },
    adminMode: { type: Boolean, default: false },
    typeFilter: { type: String, default: '' },
    uploadFilter: { type: String, default: '' },
})

const events = ref([])
const cursor = ref(null)
const total = ref(0)
const loading = ref(false)
const error = ref(null)

const eventTypeLabels = {
    upload_created: 'Upload Created',
    file_added: 'File Added',
    file_downloaded: 'File Downloaded',
}

const eventTypeColors = {
    upload_created: 'bg-emerald-500/20 text-emerald-400',
    file_added: 'bg-blue-500/20 text-blue-400',
    file_downloaded: 'bg-amber-500/20 text-amber-400',
}

function typeLabel(type) {
    return eventTypeLabels[type] || type
}

function typeColor(type) {
    return eventTypeColors[type] || 'bg-surface-500/20 text-surface-400'
}

const hasMore = computed(() => cursor.value?.after)

async function loadEvents(more = false) {
    loading.value = true
    error.value = null
    try {
        const opts = { limit: 20 }
        if (more && cursor.value?.after) opts.after = cursor.value.after

        let data
        if (props.adminMode) {
            const adminOpts = { ...opts }
            if (props.uploadFilter) adminOpts.upload = props.uploadFilter
            if (props.typeFilter) adminOpts.type = props.typeFilter
            data = await getAdminEvents(adminOpts)
        } else {
            data = await getUploadEvents(props.uploadId, props.uploadToken, opts)
        }

        if (more) {
            events.value = [...events.value, ...(data.results || [])]
        } else {
            events.value = data.results || []
        }
        cursor.value = { after: data.after, before: data.before }
        if (data.total !== undefined) total.value = data.total
    } catch (err) {
        error.value = err.message || 'Failed to load events'
    } finally {
        loading.value = false
    }
}

onMounted(() => loadEvents())

defineExpose({ reload: () => loadEvents() })
</script>

<template>
  <div class="space-y-3">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h3 class="text-sm font-medium text-surface-300">
        Events
        <span v-if="total" class="text-surface-500">({{ total }})</span>
      </h3>
    </div>

    <!-- Error -->
    <div v-if="error" class="glass-card p-3 text-sm text-red-400">
      {{ error }}
    </div>

    <!-- Loading initial -->
    <div v-else-if="loading && events.length === 0" class="glass-card p-4 text-center text-surface-500">
      Loading events…
    </div>

    <!-- Empty -->
    <div v-else-if="events.length === 0" class="glass-card p-4 text-center text-surface-500 text-sm">
      No events recorded
    </div>

    <!-- Event list -->
    <div v-else class="space-y-1.5">
      <div v-for="event in events" :key="event.id"
           class="glass-card px-3 py-2 flex items-center gap-2 text-sm">
        <!-- Type badge -->
        <span :class="typeColor(event.type)"
              class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full shrink-0">
          {{ typeLabel(event.type) }}
        </span>

        <!-- Message detail (file info) -->
        <span v-if="event.message" class="text-surface-200 truncate">
          {{ event.message }}
        </span>

        <!-- Metadata -->
        <div class="flex items-center gap-2 ml-auto shrink-0 text-xs text-surface-500">
          <span :title="new Date(event.createdAt).toLocaleString()">
            {{ formatDate(event.createdAt) }}
          </span>
          <a v-if="adminMode && event.uploadId"
             :href="`#/?id=${event.uploadId}`"
             class="font-mono text-accent-400/70 hover:text-accent-300 transition-colors"
             title="View upload">
            {{ event.uploadId }}
          </a>
          <span v-if="event.user">👤 {{ event.user }}</span>
          <span v-if="event.remoteIp" class="text-surface-600">{{ event.remoteIp }}</span>
        </div>
      </div>

      <!-- Load more -->
      <div v-if="hasMore" class="text-center pt-2">
        <button @click="loadEvents(true)"
                :disabled="loading"
                class="btn text-xs px-4 py-1.5">
          {{ loading ? 'Loading…' : 'Load more' }}
        </button>
      </div>
    </div>
  </div>
</template>
