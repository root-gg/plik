<script setup>
// A label + pipe-separated option buttons acting as a single-choice control.
// Two instances compose a "Sort: … Order: …" bar. Emits the chosen value via
// v-model so callers keep their existing change handlers.
defineProps({
    label: { type: String, required: true },
    // [{ value, label }]
    options: { type: Array, required: true },
    modelValue: { type: String, required: true },
})

const emit = defineEmits(['update:modelValue'])
</script>

<template>
  <div class="flex items-center gap-2 text-surface-400">
    <span>{{ label }}</span>
    <template v-for="(opt, i) in options" :key="opt.value">
      <span v-if="i > 0" class="text-surface-600">|</span>
      <button @click="emit('update:modelValue', opt.value)"
              :class="modelValue === opt.value ? 'text-accent-400' : 'text-surface-500 hover:text-surface-300'"
              class="transition-colors">{{ opt.label }}</button>
    </template>
  </div>
</template>
