<script setup>
import { getQrCodeURL } from '../api.js'

const props = defineProps({
  title: { type: String, default: '' },
  url: { type: String, required: true },
})

const emit = defineEmits(['close'])

const qrSrc = getQrCodeURL(props.url, 400)
</script>

<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <!-- Backdrop -->
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="emit('close')" />

      <!-- Dialog -->
      <div class="relative glass-card p-6 max-w-sm w-full animate-fade-in space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-surface-100 truncate">{{ title || $t('common.qrCode') }}</h2>
          <button class="text-surface-400 hover:text-surface-100 transition-colors" @click="emit('close')">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="flex justify-center">
          <img :src="qrSrc" :alt="'QR code for ' + title"
               class="w-64 h-64 rounded-lg bg-white p-2" />
        </div>

        <a :href="url" target="_blank" rel="noopener"
           class="text-xs text-accent-400 hover:text-accent-300 transition-colors text-center break-all block">
          {{ url }}
        </a>
      </div>
    </div>
  </Teleport>
</template>
