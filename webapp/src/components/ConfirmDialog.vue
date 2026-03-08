<script setup>
const props = defineProps({
  title: { type: String, default: '' },
  message: { type: String, required: true },
  confirmText: { type: String, default: '' },
  cancelText: { type: String, default: '' },
  variant: { type: String, default: 'danger' }, // 'danger' | 'primary'
})

const emit = defineEmits(['confirm', 'cancel'])
</script>

<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <!-- Backdrop -->
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="emit('cancel')" />

      <!-- Dialog -->
      <div class="relative glass-card p-6 max-w-md w-full animate-fade-in space-y-4">
        <div class="flex items-start gap-3">
          <!-- Icon -->
          <div v-if="variant === 'danger'"
               class="w-10 h-10 rounded-full bg-danger-500/10 flex items-center justify-center shrink-0">
            <svg class="w-5 h-5 text-danger-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div v-else
               class="w-10 h-10 rounded-full bg-accent-500/10 flex items-center justify-center shrink-0">
            <svg class="w-5 h-5 text-accent-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>

          <!-- Content -->
          <div class="flex-1 min-w-0">
            <h2 class="text-lg font-semibold text-surface-100 mb-2">{{ title || $t('common.confirmAction') }}</h2>
            <p class="text-sm text-surface-300">{{ message }}</p>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex gap-3 justify-end">
          <button class="btn-ghost px-4 py-2" @click="emit('cancel')">
            {{ cancelText || $t('common.cancel') }}
          </button>
          <button :class="variant === 'danger' ? 'btn-danger' : 'btn-primary'"
                  class="px-4 py-2"
                  @click="emit('confirm')">
            {{ confirmText || $t('common.confirm') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
