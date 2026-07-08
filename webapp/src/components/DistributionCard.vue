<script setup>
import { formatCount, ratioPercent } from '../utils.js'

// A titled card listing {label, value} rows with proportional bars.
// Data-driven so the current/lifetime file-size, TTL and feature cards all
// share one implementation. The bar colour is a single prop/token default so
// re-theming stays a one-line change — no per-card hard-coded hues.
defineProps({
    title: { type: String, required: true },
    // Optional one-line explanation shown as a native title-attribute tooltip
    // on the heading — explains why the Current/Lifetime card in this pair
    // can look identical on a fresh/low-churn scope. Deliberately a
    // caption/tooltip only, not a collapsing/toggle affordance.
    caption: { type: String, default: '' },
    // [{ key, label, value }]
    items: { type: Array, required: true },
    // Denominator for the proportional bars (0 renders empty bars, no crash)
    total: { type: Number, default: 0 },
    barClass: { type: String, default: 'bg-accent-400' },
})
</script>

<template>
  <div class="glass-card p-5">
    <h3 class="text-sm text-surface-400 uppercase tracking-wider mb-4" :title="caption || undefined">{{ title }}</h3>
    <div class="space-y-3">
      <div v-for="item in items" :key="item.key">
        <div class="flex justify-between text-xs mb-1">
          <span class="text-surface-400">{{ item.label }}</span>
          <span class="text-surface-300 tabular-nums">{{ formatCount(item.value) }}</span>
        </div>
        <div class="h-1.5 rounded-full bg-surface-800 overflow-hidden">
          <div class="h-full rounded-full"
               :class="barClass"
               :style="{ width: ratioPercent(item.value, total) + '%' }" />
        </div>
      </div>
    </div>
  </div>
</template>
