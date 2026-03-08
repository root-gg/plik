<script setup>
import { ref } from 'vue'

const props = defineProps({
  text: { type: String, required: true },
  label: { type: String, default: '' },
  size: { type: String, default: 'sm' }, // 'sm' | 'md'
})

const copied = ref(false)
let timer = null

async function copy() {
  try {
    await navigator.clipboard.writeText(props.text)
    copied.value = true
    clearTimeout(timer)
    timer = setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // Fallback for non-HTTPS
    const ta = document.createElement('textarea')
    ta.value = props.text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
    copied.value = true
    clearTimeout(timer)
    timer = setTimeout(() => { copied.value = false }, 2000)
  }
}
</script>

<template>
  <button
    class="btn transition-colors"
    :class="[
      copied
        ? 'bg-success-500/20 text-success-500'
        : 'bg-surface-700/50 text-surface-300 hover:bg-surface-600/50 hover:text-surface-100',
      size === 'sm' ? 'px-2 py-1.5 text-xs' : 'px-3 py-2 text-sm',
    ]"
    :title="copied ? $t('common.copied') : $t('common.copyToClipboard')"
    @click.stop.prevent="copy">
    <!-- Copied check icon -->
    <svg v-if="copied" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
    </svg>
    <!-- Copy icon -->
    <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
    </svg>
    <span v-if="label">{{ copied ? $t('common.copied') : label }}</span>
  </button>
</template>
