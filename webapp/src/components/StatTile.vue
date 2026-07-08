<script setup>
import { computed } from 'vue'
import { formatSizeOrZero, formatCount } from '../utils.js'

// A single labelled statistic: a big humanized number over a small label.
// Counts are grouped via toLocaleString(); sizes via humanReadableSize().
const props = defineProps({
    label: { type: String, required: true },
    // null/undefined → "—" (unavailable, e.g. a byte window whose daily
    // series fetch failed): distinct from an honest 0.
    value: { type: [Number, String], default: 0 },
    // 'count' → thousands-grouped integer, 'size' → human-readable bytes
    format: { type: String, default: 'count' },
    subLabel: { type: String, default: '' },
    // Wrap the tile in a bordered card (used by the downloads window row)
    bordered: { type: Boolean, default: false },
    // Colour treatment for the value (a single token — keeps re-theming easy)
    valueClass: { type: String, default: 'text-surface-200' },
})

const display = computed(() => {
    if (props.value === null || props.value === undefined) return '—'
    if (props.format === 'size') return formatSizeOrZero(props.value)
    return formatCount(props.value)
})
</script>

<template>
  <div :class="bordered ? 'rounded-lg border border-surface-700/50 p-4 text-center' : 'text-center'">
    <p class="text-2xl font-bold tabular-nums" :class="valueClass">{{ display }}</p>
    <p class="text-xs text-surface-500">{{ label }}</p>
    <p v-if="subLabel" class="text-xs text-surface-500">{{ subLabel }}</p>
  </div>
</template>
