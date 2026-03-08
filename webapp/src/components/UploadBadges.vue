<script setup>
import { computed } from 'vue'

const props = defineProps({
    upload: { type: Object, required: true },
    size: { type: String, default: 'md', validator: v => ['sm', 'md'].includes(v) },
})

const sizeClasses = computed(() =>
    props.size === 'sm' ? 'text-[10px] px-1.5 py-0.5' : 'text-xs px-2 py-0.5'
)
</script>

<template>
  <div v-if="upload.oneShot || upload.removable || upload.stream || upload.protectedByPassword || upload.e2ee || upload.extend_ttl"
       class="flex flex-wrap" :class="size === 'sm' ? 'gap-1' : 'gap-1.5'">
    <span v-if="upload.oneShot"
          class="rounded-full bg-warning-500/15 text-warning-500" :class="sizeClasses">
      {{ $t('badges.oneShot') }}
    </span>
    <span v-if="upload.removable"
          class="rounded-full bg-danger-500/15 text-danger-500" :class="sizeClasses">
      {{ $t('badges.removable') }}
    </span>
    <span v-if="upload.stream"
          class="rounded-full bg-accent-500/15 text-accent-400" :class="sizeClasses">
      {{ $t('badges.stream') }}
    </span>
    <span v-if="upload.extend_ttl"
          class="rounded-full bg-emerald-500/15 text-emerald-400" :class="sizeClasses">
      {{ $t('badges.extendTTL') }}
    </span>
    <span v-if="upload.protectedByPassword"
          class="rounded-full bg-surface-600/50 text-surface-300" :class="sizeClasses">
      🔒 {{ $t('badges.password') }}
    </span>
    <span v-if="upload.e2ee"
          class="rounded-full bg-accent-500/15 text-accent-400" :class="sizeClasses">
      🔐 {{ $t('badges.encrypted') }}
    </span>
  </div>
</template>
