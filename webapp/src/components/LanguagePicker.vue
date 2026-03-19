<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getAvailableLanguages, currentLanguage, setUserLanguage } from '../settings.js'

const props = defineProps({
    buttonClass: {
        type: String,
        default: 'btn-ghost text-sm',
    },
})

const open = ref(false)
const pickerRef = ref(null)

const languages = computed(() => getAvailableLanguages())
const showPicker = computed(() => languages.value.length > 1)

function toggle() {
    open.value = !open.value
}

async function select(name) {
    await setUserLanguage(name)
    open.value = false
}

function onClickOutside(e) {
    if (pickerRef.value && !pickerRef.value.contains(e.target)) {
        open.value = false
    }
}

onMounted(() => document.addEventListener('click', onClickOutside, true))
onUnmounted(() => document.removeEventListener('click', onClickOutside, true))
</script>

<template>
  <div v-if="showPicker" ref="pickerRef" class="relative w-full">
    <!-- Trigger button -->
    <button
        id="language-picker-toggle"
        :class="[buttonClass, { 'bg-surface-700/50 text-surface-100': open }]"
        @click.stop="toggle"
        :title="$t('languagePicker.switchLanguage')">
      <!-- Globe icon -->
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <slot />
    </button>

    <!-- Dropdown -->
    <Transition name="dropdown">
      <div v-if="open"
           class="absolute right-0 top-full mt-2 w-44
                  bg-surface-900 border border-surface-700/50
                  rounded-lg shadow-xl overflow-hidden z-50">
        <div class="py-1">
          <button
              v-for="lang in languages"
              :key="lang.name"
              :id="'lang-option-' + lang.name"
              class="w-full flex items-center gap-3 px-3 py-2 text-sm text-left
                     transition-colors hover:bg-surface-700/30"
              :class="currentLanguage === lang.name
                ? 'text-accent-400'
                : 'text-surface-300 hover:text-surface-100'"
              @click="select(lang.name)">

            <!-- Check mark for active language -->
            <svg v-if="currentLanguage === lang.name"
                 class="w-4 h-4 shrink-0 text-accent-400"
                 fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M5 13l4 4L19 7" />
            </svg>
            <span v-else class="w-4 h-4 shrink-0"></span>

            <img v-if="lang.flag" :src="lang.flag" class="w-5 h-3.5 shrink-0 rounded-[2px]" alt="" />
            <span class="truncate">{{ lang.label }}</span>
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
    transition: opacity 0.15s ease, transform 0.15s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
    opacity: 0;
    transform: translateY(-4px);
}
</style>
