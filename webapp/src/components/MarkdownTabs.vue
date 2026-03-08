<script setup>
/**
 * MarkdownTabs — reusable Code/Preview (or Write/Preview) tab bar
 * with rendered-markdown panel.
 *
 * Props:
 *   modelValue  — active tab ('code'|'write' | 'preview')
 *   leftLabel   — label for the left tab (default: 'Code')
 *   leftIcon    — 'code' (angle-brackets) or 'write' (pencil)
 *   renderedHtml — pre-rendered, sanitised HTML for the Preview tab
 *
 * Emits:
 *   update:modelValue
 *
 * Slots:
 *   default — editor content (CodeEditor, textarea, …) shown when left tab is active
 */
const props = defineProps({
  modelValue: { type: String, required: true },
  leftLabel:  { type: String, default: 'Code' },
  leftIcon:   { type: String, default: 'code', validator: v => ['code', 'write'].includes(v) },
  renderedHtml: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue'])

const leftValue = props.leftIcon === 'write' ? 'write' : 'code'
</script>

<template>
  <!-- Tab bar -->
  <div class="flex border-b border-surface-700/50">
    <button class="px-4 py-2 text-sm font-medium transition-colors"
            :class="modelValue === leftValue
              ? 'text-accent-400 border-b-2 border-accent-400 bg-surface-700/30'
              : 'text-surface-400 hover:text-surface-200'"
            @click="emit('update:modelValue', leftValue)">
      <!-- Code icon (angle-brackets) -->
      <svg v-if="leftIcon === 'code'" class="w-4 h-4 inline-block mr-1 -mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
      </svg>
      <!-- Write icon (pencil) -->
      <svg v-else class="w-4 h-4 inline-block mr-1 -mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
      </svg>
      {{ leftLabel }}
      <slot name="left-badge" />
    </button>
    <button class="px-4 py-2 text-sm font-medium transition-colors"
            :class="modelValue === 'preview'
              ? 'text-accent-400 border-b-2 border-accent-400 bg-surface-700/30'
              : 'text-surface-400 hover:text-surface-200'"
            @click="emit('update:modelValue', 'preview')">
      <svg class="w-4 h-4 inline-block mr-1 -mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
      </svg>
      {{ $t('common.preview') }}
    </button>
  </div>

  <!-- Preview panel -->
  <div v-if="modelValue === 'preview'" class="p-4">
    <div v-if="renderedHtml"
         class="prose prose-invert prose-sm max-w-none"
         v-html="renderedHtml" />
    <p v-else class="text-sm text-surface-500 italic">{{ $t('common.nothingToPreview') }}</p>
  </div>

  <!-- Editor slot (Code / Write tab) -->
  <slot v-if="modelValue !== 'preview'" />
</template>
