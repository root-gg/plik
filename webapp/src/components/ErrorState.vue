<script setup>
defineProps({
  message: { type: String, required: true },
  retryLabel: { type: String, default: '' },
})
defineEmits(['retry'])
</script>

<template>
  <div class="glass-card border-danger-500/50 p-6 text-center animate-fade-in">
    <svg class="w-12 h-12 text-danger-500 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
    <p class="text-danger-500 font-medium">{{ message }}</p>
    <button class="btn-ghost mt-4" @click="$emit('retry')">{{ retryLabel || $t('common.tryAgain') }}</button>
  </div>
</template>
