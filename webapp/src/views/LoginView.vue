<script setup>
import { ref, computed, onMounted } from 'vue'
import { config, isFeatureEnabled } from '../config.js'
import { auth, login } from '../authStore.js'
import { oidcLogin as apiOidcLogin } from '../api.js'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t: $t } = useI18n()

// Redirect authenticated users to their intended destination.
// This handles OAuth callbacks (server redirects to /#/login after success)
// and direct navigation to /#/login while already logged in.
// The redirect destination is stored in sessionStorage by the router guard.
function consumeRedirect() {
  const redirect = sessionStorage.getItem('plik-auth-redirect') || '/'
  sessionStorage.removeItem('plik-auth-redirect')
  return redirect
}

onMounted(() => {
  if (auth.user) router.replace(consumeRedirect())
})

const loginName = ref('')
const password = ref('')
const error = ref(null)
const loading = ref(false)

const hasOAuthProviders = computed(() =>
  config.googleAuthentication || config.ovhAuthentication || config.oidcAuthentication
)

async function handleSubmit() {
  if (loading.value) return
  error.value = null
  if (!loginName.value || !password.value) {
    error.value = $t('loginView.pleaseEnterCredentials')
    return
  }
  loading.value = true
  try {
    const ok = await login(loginName.value, password.value)
    if (ok) {
      router.push(consumeRedirect())
    } else {
      error.value = $t('loginView.invalidCredentials')
    }
  } catch (err) {
    error.value = err.message || $t('loginView.loginFailed')
  } finally {
    loading.value = false
  }
}

async function googleLogin() {
  try {
    const resp = await fetch(window.location.origin + window.location.pathname.replace(/\/$/, '') + '/auth/google/login')
    if (!resp.ok) throw new Error($t('loginView.googleLoginFailed'))
    const url = await resp.text()
    window.location.href = url
  } catch (err) {
    error.value = err.message || $t('loginView.googleLoginFailed')
  }
}

async function ovhLogin() {
  try {
    const resp = await fetch(window.location.origin + window.location.pathname.replace(/\/$/, '') + '/auth/ovh/login')
    if (!resp.ok) throw new Error($t('loginView.ovhLoginFailed'))
    const url = await resp.text()
    window.location.href = url
  } catch (err) {
    error.value = err.message || $t('loginView.ovhLoginFailed')
  }
}

async function handleOidcLogin() {
  try {
    const url = await apiOidcLogin()
    window.location.href = url
  } catch (err) {
    error.value = err.message || $t('loginView.oidcLoginFailed')
  }
}
</script>

<template>
  <div class="w-full min-h-[calc(100vh-3.5rem)] flex items-center justify-center p-4">
    <div class="w-full max-w-sm">
      <p class="text-surface-400 text-sm text-center mb-4">{{ $t('loginView.signInToAccount') }}</p>

      <!-- Login Card -->
      <div class="glass-card p-6 space-y-5">
        <!-- Error -->
        <div v-if="error"
             class="bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-3 text-sm text-red-400 flex items-start gap-2">
          <svg class="w-4 h-4 mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
          {{ error }}
        </div>

        <!-- Local Login Form (hidden when local auth is disabled) -->
        <form v-if="isFeatureEnabled('local_login')" @submit.prevent="handleSubmit" class="space-y-4">
          <div>
            <label class="text-xs text-surface-400 block mb-1.5">{{ $t('loginView.loginLabel') }}</label>
            <input type="text"
                   v-model="loginName"
                   class="input-field w-full"
                   :placeholder="$t('loginView.enterLogin')"
                   autocomplete="username"
                   autocapitalize="off"
                   autofocus />
          </div>
          <div>
            <label class="text-xs text-surface-400 block mb-1.5">{{ $t('loginView.passwordLabel') }}</label>
            <input type="password"
                   v-model="password"
                   class="input-field w-full"
                   :placeholder="$t('loginView.enterPassword')"
                   autocomplete="current-password" />
          </div>
          <button type="submit"
                  class="btn-primary w-full py-2.5"
                  :disabled="loading">
            <svg v-if="loading" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ loading ? $t('loginView.signingIn') : $t('loginView.signIn') }}
          </button>
        </form>

        <!-- OAuth Divider -->
        <div v-if="isFeatureEnabled('local_login') && hasOAuthProviders"
             class="flex items-center gap-3">
          <div class="flex-1 border-t border-surface-700/50"></div>
          <span class="text-xs text-surface-500">{{ $t('loginView.orContinueWith') }}</span>
          <div class="flex-1 border-t border-surface-700/50"></div>
        </div>

        <!-- OAuth Buttons -->
        <div v-if="hasOAuthProviders"
             class="space-y-2">
          <button v-if="config.googleAuthentication"
                  @click="googleLogin"
                  class="w-full flex items-center justify-center gap-2 py-2.5 rounded-xl border border-surface-600
                         bg-surface-700/50 text-surface-200 hover:bg-surface-600/50 hover:text-surface-100
                         transition-all text-sm font-medium">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
              <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
              <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
              <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
              <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
            </svg>
            {{ $t('loginView.signInWithGoogle') }}
          </button>

          <button v-if="config.ovhAuthentication"
                  @click="ovhLogin"
                  class="w-full flex items-center justify-center gap-2 py-2.5 rounded-xl border border-surface-600
                         bg-surface-700/50 text-surface-200 hover:bg-surface-600/50 hover:text-surface-100
                         transition-all text-sm font-medium">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
              <path d="M0 12l5-8h14l-5 8 5 8H5z" fill="#0050D4"/>
            </svg>
            {{ $t('loginView.signInWithOvh') }}
          </button>

          <button v-if="config.oidcAuthentication"
                  @click="handleOidcLogin"
                  class="w-full flex items-center justify-center gap-2 py-2.5 rounded-xl border border-surface-600
                         bg-surface-700/50 text-surface-200 hover:bg-surface-600/50 hover:text-surface-100
                         transition-all text-sm font-medium">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
            </svg>
            {{ $t('loginView.signInWithProvider', { provider: config.oidcProviderName }) }}
          </button>
        </div>
      </div>

      <!-- Back link (hidden when auth is forced — there's nowhere to go) -->
      <div v-if="config.feature_authentication !== 'forced'" class="text-center mt-6">
        <router-link to="/" class="text-sm text-surface-400 hover:text-accent-400 transition-colors">
          {{ $t('loginView.backToUpload') }}
        </router-link>
      </div>
    </div>
  </div>
</template>
