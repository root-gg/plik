<script setup>
defineProps({
    sortBy:      { type: String, required: true },
    sortOrder:   { type: String, required: true },
    badgeFilters:{ type: Object, required: true },
    showExtendTTL: { type: Boolean, default: true },
})

const emit = defineEmits(['update:sort-by', 'update:sort-order', 'toggle-filter'])
</script>

<template>
  <div class="glass-card p-3 mb-4 space-y-2 text-sm">
    <div class="flex flex-wrap items-center gap-4">
      <!-- Sort by -->
      <div class="flex items-center gap-2 text-surface-400">
        <span>{{ $t('uploadControls.sort') }}</span>
        <button @click="emit('update:sort-by', 'date')"
                :class="sortBy === 'date' ? 'text-accent-400' : 'text-surface-500 hover:text-surface-300'"
                class="transition-colors">{{ $t('uploadControls.date') }}</button>
        <span class="text-surface-600">|</span>
        <button @click="emit('update:sort-by', 'size')"
                :class="sortBy === 'size' ? 'text-accent-400' : 'text-surface-500 hover:text-surface-300'"
                class="transition-colors">{{ $t('uploadControls.size') }}</button>
      </div>
      <!-- Order -->
      <div class="flex items-center gap-2 text-surface-400">
        <span>{{ $t('uploadControls.order') }}</span>
        <button @click="emit('update:sort-order', 'desc')"
                :class="sortOrder === 'desc' ? 'text-accent-400' : 'text-surface-500 hover:text-surface-300'"
                class="transition-colors">{{ $t('uploadControls.desc') }}</button>
        <span class="text-surface-600">|</span>
        <button @click="emit('update:sort-order', 'asc')"
                :class="sortOrder === 'asc' ? 'text-accent-400' : 'text-surface-500 hover:text-surface-300'"
                class="transition-colors">{{ $t('uploadControls.asc') }}</button>
      </div>
    </div>
    <!-- Badge filters -->
    <div class="flex flex-wrap items-center gap-2 text-surface-400">
        <span>{{ $t('uploadControls.filter') }}</span>
        <button @click="emit('toggle-filter', 'oneShot')"
                :class="badgeFilters.oneShot ? 'bg-amber-500/20 text-amber-400 ring-1 ring-amber-500/50' : 'text-surface-500 hover:text-surface-300'"
                class="px-2 py-0.5 rounded text-xs transition-all">{{ $t('uploadControls.oneShot') }}</button>
        <button @click="emit('toggle-filter', 'removable')"
                :class="badgeFilters.removable ? 'bg-sky-500/20 text-sky-400 ring-1 ring-sky-500/50' : 'text-surface-500 hover:text-surface-300'"
                class="px-2 py-0.5 rounded text-xs transition-all">{{ $t('uploadControls.removable') }}</button>
        <button @click="emit('toggle-filter', 'stream')"
                :class="badgeFilters.stream ? 'bg-violet-500/20 text-violet-400 ring-1 ring-violet-500/50' : 'text-surface-500 hover:text-surface-300'"
                class="px-2 py-0.5 rounded text-xs transition-all">{{ $t('uploadControls.stream') }}</button>
        <button v-if="showExtendTTL"
                @click="emit('toggle-filter', 'extendTTL')"
                :class="badgeFilters.extendTTL ? 'bg-emerald-500/20 text-emerald-400 ring-1 ring-emerald-500/50' : 'text-surface-500 hover:text-surface-300'"
                class="px-2 py-0.5 rounded text-xs transition-all">{{ $t('uploadControls.extendTTL') }}</button>
        <button @click="emit('toggle-filter', 'password')"
                :class="badgeFilters.password ? 'bg-rose-500/20 text-rose-400 ring-1 ring-rose-500/50' : 'text-surface-500 hover:text-surface-300'"
                class="px-2 py-0.5 rounded text-xs transition-all">{{ $t('uploadControls.password') }}</button>
        <button @click="emit('toggle-filter', 'e2ee')"
                :class="badgeFilters.e2ee ? 'bg-fuchsia-500/20 text-fuchsia-400 ring-1 ring-fuchsia-500/50' : 'text-surface-500 hover:text-surface-300'"
                class="px-2 py-0.5 rounded text-xs transition-all">{{ $t('uploadControls.encrypted') }}</button>
    </div>
    <!-- Active filter chips (token, user, etc.) -->
    <slot name="active-filters" />
  </div>
</template>
