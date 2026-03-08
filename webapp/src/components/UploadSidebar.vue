<script setup>
import { computed, ref } from 'vue'
import { config, isFeatureEnabled, isFeatureForced } from '../config.js'
import { secondsToTTL } from '../utils.js'
import { generatePassphrase } from '../crypto.js'

const props = defineProps({
  settings: {
    type: Object,
    required: true,
  },
  effectiveMaxTTL: {
    type: Number,
    default: 0,
  },
})

const emit = defineEmits(['update:settings'])

function updateSetting(key, value) {
  emit('update:settings', { ...props.settings, [key]: value })
}

// Password generation
const copied = ref(false)
const e2eeCopied = ref(false)
const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%&*'

function generatePassword() {
  const array = new Uint32Array(20)
  crypto.getRandomValues(array)
  return Array.from(array, b => chars[b % chars.length]).join('')
}

function togglePassword() {
  if (isFeatureForced('password')) return
  const enabling = !props.settings.passwordEnabled
  if (enabling) {
    emit('update:settings', {
      ...props.settings,
      passwordEnabled: true,
      login: 'user',
      password: generatePassword(),
    })
  } else {
    updateSetting('passwordEnabled', false)
  }
}

function copyPassword() {
  if (!props.settings.password) return
  navigator.clipboard.writeText(props.settings.password)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1500)
}

// E2EE toggle
function toggleE2EE() {
  const enabling = !props.settings.e2eeEnabled
  if (enabling) {
    emit('update:settings', {
      ...props.settings,
      e2eeEnabled: true,
      e2eePassphrase: generatePassphrase(),
    })
  } else {
    emit('update:settings', {
      ...props.settings,
      e2eeEnabled: false,
      e2eePassphrase: '',
    })
  }
}

function copyE2EEPassphrase() {
  if (!props.settings.e2eePassphrase) return
  navigator.clipboard.writeText(props.settings.e2eePassphrase)
  e2eeCopied.value = true
  setTimeout(() => { e2eeCopied.value = false }, 1500)
}

// TTL handling
const defaultTTL = computed(() => secondsToTTL(config.defaultTTL))
const ttlValue = computed({
  get: () => props.settings.ttlValue,
  set: (v) => updateSetting('ttlValue', Number(v)),
})
const ttlUnit = computed({
  get: () => props.settings.ttlUnit,
  set: (v) => updateSetting('ttlUnit', v),
})
const maxTTL = computed(() => {
  const val = props.effectiveMaxTTL || config.maxTTL
  return val && val > 0 ? secondsToTTL(val) : null
})

// "Never expires" is allowed when there is no maxTTL limit
const canNeverExpire = computed(() => {
  const val = props.effectiveMaxTTL || config.maxTTL
  return !val || val <= 0
})

function toggleNeverExpires() {
  updateSetting('neverExpires', !props.settings.neverExpires)
}

const hasAnySettings = computed(() =>
  isFeatureEnabled('one_shot') ||
  isFeatureEnabled('stream') ||
  isFeatureEnabled('removable') ||
  isFeatureEnabled('password') ||
  isFeatureEnabled('comments') ||
  isFeatureEnabled('extend_ttl') ||
  isFeatureEnabled('set_ttl') ||
  isFeatureEnabled('e2ee')
)
</script>

<template>
  <aside v-if="hasAnySettings" class="w-full md:w-80 md:shrink-0 p-4 space-y-3 animate-slide-in">
    <!-- Upload Settings -->
    <div class="sidebar-section relative z-10">
      <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider mb-2">{{ $t('uploadSidebar.uploadSettings') }}</h3>

      <!-- One Shot -->
      <label v-if="isFeatureEnabled('one_shot')"
             class="flex items-center justify-between py-1 cursor-pointer group">
        <span class="text-sm text-surface-200 group-hover:text-surface-100 transition-colors flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <circle cx="11" cy="14" r="7" stroke-width="2" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.5 8.5L17 7m0 0l1.5-1.5M17 7l1.5 1.5M17 7l-1.5-1.5" />
            <path stroke-linecap="round" stroke-width="2" d="M14 9l2-2" />
          </svg>
          {{ $t('uploadSidebar.destructAfterDownload') }}
          <span class="setting-help-wrap relative" @click.prevent.stop>
            <span class="setting-help" tabindex="0" role="button" aria-label="Help">?</span>
            <span class="setting-tooltip">{{ $t('uploadSidebar.destructHelp') }}</span>
          </span>
        </span>
        <button type="button"
                class="toggle-switch"
                :data-active="settings.oneShot"
                :disabled="isFeatureForced('one_shot')"
                @click="!isFeatureForced('one_shot') && updateSetting('oneShot', !settings.oneShot)">
          <span class="toggle-dot" />
        </button>
      </label>

      <!-- Streaming -->
      <label v-if="isFeatureEnabled('stream')"
             class="flex items-center justify-between py-1 cursor-pointer group">
        <span class="text-sm text-surface-200 group-hover:text-surface-100 transition-colors flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5.636 18.364a9 9 0 010-12.728m12.728 0a9 9 0 010 12.728M9.172 15.828a5 5 0 010-7.072m5.656 0a5 5 0 010 7.072M12 12h.01" />
          </svg>
          {{ $t('uploadSidebar.streaming') }}
          <span class="setting-help-wrap relative" @click.prevent.stop>
            <span class="setting-help" tabindex="0" role="button" aria-label="Help">?</span>
            <span class="setting-tooltip">{{ $t('uploadSidebar.streamingHelp') }}</span>
          </span>
        </span>
        <button type="button"
                class="toggle-switch"
                :data-active="settings.stream"
                :disabled="isFeatureForced('stream')"
                @click="!isFeatureForced('stream') && updateSetting('stream', !settings.stream)">
          <span class="toggle-dot" />
        </button>
      </label>

      <!-- Removable -->
      <label v-if="isFeatureEnabled('removable')"
             class="flex items-center justify-between py-1 cursor-pointer group">
        <span class="text-sm text-surface-200 group-hover:text-surface-100 transition-colors flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5 text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
          {{ $t('uploadSidebar.removable') }}
          <span class="setting-help-wrap relative" @click.prevent.stop>
            <span class="setting-help" tabindex="0" role="button" aria-label="Help">?</span>
            <span class="setting-tooltip">{{ $t('uploadSidebar.removableHelp') }}</span>
          </span>
        </span>
        <button type="button"
                class="toggle-switch"
                :data-active="settings.removable"
                :disabled="isFeatureForced('removable')"
                @click="!isFeatureForced('removable') && updateSetting('removable', !settings.removable)">
          <span class="toggle-dot" />
        </button>
      </label>

      <!-- E2EE -->
      <div v-if="isFeatureEnabled('e2ee')">
        <label class="flex items-center justify-between py-1 cursor-pointer group">
          <span class="text-sm text-surface-200 group-hover:text-surface-100 transition-colors flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
            {{ $t('uploadSidebar.e2ee') }}
            <span class="setting-help-wrap relative" @click.prevent.stop>
              <span class="setting-help" tabindex="0" role="button" aria-label="Help">?</span>
              <span class="setting-tooltip">{{ $t('uploadSidebar.e2eeHelp') }}</span>
            </span>
          </span>
          <button type="button"
                  class="toggle-switch"
                  :class="{ 'opacity-50 cursor-not-allowed': isFeatureForced('e2ee') }"
                  :data-active="settings.e2eeEnabled"
                  :disabled="isFeatureForced('e2ee')"
                  @click="toggleE2EE">
            <span class="toggle-dot" />
          </button>
        </label>
        <div v-if="settings.e2eeEnabled" class="mt-2">
          <label class="text-xs text-surface-500 mb-1 block">{{ $t('uploadSidebar.passphrase') }}</label>
          <div class="relative">
            <input type="text"
                   class="input-field pr-9 font-mono text-xs"
                   :class="{ 'ring-1 ring-danger-500/50 border-danger-500/50': !settings.e2eePassphrase.trim() }"
                   :value="settings.e2eePassphrase"
                   @input="updateSetting('e2eePassphrase', $event.target.value)" />
            <button type="button"
                    class="absolute right-2 top-1/2 -translate-y-1/2 text-surface-400 hover:text-surface-100 transition-colors"
                    :title="$t('uploadSidebar.copyPassphrase')"
                    @click="copyE2EEPassphrase">
              <svg v-if="!e2eeCopied" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <rect x="9" y="9" width="13" height="13" rx="2" stroke-width="2" />
                <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" stroke-width="2" />
              </svg>
              <svg v-else class="w-4 h-4 text-success-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
            </button>
          </div>
          <p v-if="!settings.e2eePassphrase.trim()" class="text-xs text-danger-400 mt-1">{{ $t('uploadSidebar.passphraseEmpty') }}</p>
        </div>
      </div>

      <!-- Password -->
      <div v-if="isFeatureEnabled('password')">
        <label class="flex items-center justify-between py-1 cursor-pointer group">
          <span class="text-sm text-surface-200 group-hover:text-surface-100 transition-colors flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
            </svg>
            {{ $t('uploadSidebar.password') }}
            <span class="setting-help-wrap relative" @click.prevent.stop>
              <span class="setting-help" tabindex="0" role="button" aria-label="Help">?</span>
              <span class="setting-tooltip">{{ $t('uploadSidebar.passwordHelp') }}</span>
            </span>
          </span>
          <button type="button"
                  class="toggle-switch"
                  :data-active="settings.passwordEnabled"
                  :disabled="isFeatureForced('password')"
                  @click="togglePassword">
            <span class="toggle-dot" />
          </button>
        </label>
        <div v-if="settings.passwordEnabled" class="mt-2 space-y-2">
          <input type="text"
                 class="input-field"
                 :placeholder="$t('uploadSidebar.loginPlaceholder')"
                 :value="settings.login"
                 @input="updateSetting('login', $event.target.value)" />
          <div class="relative">
            <input type="text"
                   class="input-field pr-9 font-mono text-xs"
                   :placeholder="$t('uploadSidebar.passwordPlaceholder')"
                   :value="settings.password"
                   @input="updateSetting('password', $event.target.value)" />
            <button type="button"
                    class="absolute right-2 top-1/2 -translate-y-1/2 text-surface-400 hover:text-surface-100 transition-colors"
                    :title="$t('uploadSidebar.copyPassword')"
                    @click="copyPassword">
              <svg v-if="!copied" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <rect x="9" y="9" width="13" height="13" rx="2" stroke-width="2" />
                <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" stroke-width="2" />
              </svg>
              <svg v-else class="w-4 h-4 text-success-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      <!-- Comment -->
      <label v-if="isFeatureEnabled('comments')"
             class="flex items-center justify-between py-1 cursor-pointer group">
        <span class="text-sm text-surface-200 group-hover:text-surface-100 transition-colors flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
          </svg>
          {{ $t('uploadSidebar.comment') }}
          <span class="setting-help-wrap relative" @click.prevent.stop>
            <span class="setting-help" tabindex="0" role="button" aria-label="Help">?</span>
            <span class="setting-tooltip">{{ $t('uploadSidebar.commentHelp') }}</span>
          </span>
        </span>
        <button type="button"
                class="toggle-switch"
                :data-active="settings.commentEnabled"
                :disabled="isFeatureForced('comments')"
                @click="!isFeatureForced('comments') && updateSetting('commentEnabled', !settings.commentEnabled)">
          <span class="toggle-dot" />
        </button>
      </label>

      <!-- Extend TTL -->
      <label v-if="isFeatureEnabled('extend_ttl')"
             class="flex items-center justify-between py-1 cursor-pointer group">
        <span class="text-sm text-surface-200 group-hover:text-surface-100 transition-colors flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5 text-teal-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {{ $t('uploadSidebar.extendTTL') }}
          <span class="setting-help-wrap relative" @click.prevent.stop>
            <span class="setting-help" tabindex="0" role="button" aria-label="Help">?</span>
            <span class="setting-tooltip">{{ $t('uploadSidebar.extendTTLHelp') }}</span>
          </span>
        </span>
        <button type="button"
                class="toggle-switch"
                :data-active="settings.extendTTL"
                :disabled="isFeatureForced('extend_ttl')"
                @click="!isFeatureForced('extend_ttl') && updateSetting('extendTTL', !settings.extendTTL)">
          <span class="toggle-dot" />
        </button>
      </label>
    </div>

    <!-- TTL Section -->
    <div v-if="isFeatureEnabled('set_ttl')" class="sidebar-section">
      <h3 class="text-xs font-semibold text-surface-400 uppercase tracking-wider mb-2">{{ $t('uploadSidebar.expiration') }}</h3>

      <!-- Never expires toggle -->
      <label v-if="canNeverExpire"
             class="flex items-center justify-between py-1 mb-2 cursor-pointer group">
        <span class="text-sm text-surface-200 group-hover:text-surface-100 transition-colors flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.178 8c5.096 0 5.096 8 0 8-5.095 0-7.133-8-12.739-8-4.585 0-4.585 8 0 8 5.606 0 7.644-8 12.74-8z" />
          </svg>
          {{ $t('uploadSidebar.neverExpires') }}
          <span class="setting-help-wrap relative" @click.prevent.stop>
            <span class="setting-help" tabindex="0" role="button" aria-label="Help">?</span>
            <span class="setting-tooltip">{{ $t('uploadSidebar.neverExpiresHelp') }}</span>
          </span>
        </span>
        <button type="button"
                class="toggle-switch"
                :data-active="settings.neverExpires"
                @click="toggleNeverExpires">
          <span class="toggle-dot" />
        </button>
      </label>

      <div v-if="!settings.neverExpires" class="flex items-center gap-2">
        <input type="number"
               class="input-field w-20"
               min="1"
               :value="ttlValue"
               :disabled="isFeatureForced('set_ttl')"
               @input="ttlValue = $event.target.value" />
        <select class="input-field flex-1"
                :value="ttlUnit"
                :disabled="isFeatureForced('set_ttl')"
                @change="ttlUnit = $event.target.value">
          <option value="minutes">{{ $t('uploadSidebar.minutes') }}</option>
          <option value="hours">{{ $t('uploadSidebar.hours') }}</option>
          <option value="days">{{ $t('uploadSidebar.days') }}</option>
        </select>
      </div>
      <p v-if="maxTTL && !settings.neverExpires" class="text-xs text-surface-500 mt-1">
        {{ $t('uploadSidebar.maxTTL', { value: maxTTL.value, unit: $t('uploadSidebar.' + maxTTL.unit) }) }}
      </p>
    </div>
  </aside>
</template>
