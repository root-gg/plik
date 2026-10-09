<script setup>
// A single-choice pill-button group: the Activity metric selector
// (StatsUsagePanel), the Trending window chips, and the Trending metric
// toggle (both TrendingPanel) are all the same control — one active option
// highlighted among a small horizontal set, click-to-select, no keyboard
// handling beyond what native <button> elements already give for free
// (Tab between options, Space/Enter to activate).
defineProps({
    // [{ value, label }]
    options: { type: Array, required: true },
    modelValue: { type: String, required: true },
    // Accessible name for the group (role="group" aria-label)
    ariaLabel: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue'])
</script>

<template>
  <div class="flex items-center gap-1 rounded-lg bg-surface-800/60 p-1 flex-wrap"
       role="group" :aria-label="ariaLabel">
    <button v-for="option in options" :key="option.value" type="button"
            @click="emit('update:modelValue', option.value)"
            :aria-pressed="modelValue === option.value"
            :class="modelValue === option.value ? 'bg-accent-500/15 text-accent-300' : 'text-surface-500 hover:text-surface-300'"
            class="px-2.5 py-1 rounded-md text-xs transition-colors">
      {{ option.label }}
    </button>
  </div>
</template>
