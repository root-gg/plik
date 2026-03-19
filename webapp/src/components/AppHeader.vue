<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { config, isFeatureEnabled } from '../config.js'
import { auth, clearImpersonate } from '../authStore.js'
import { settings, getAvailableThemes } from '../settings.js'
import ThemePicker from './ThemePicker.vue'
import LanguagePicker from './LanguagePicker.vue'
import { SUPPORTED_LOCALES } from '../i18n.js'

const showThemePicker = computed(() => getAvailableThemes().length > 1)
const showLanguagePicker = computed(() => SUPPORTED_LOCALES.length > 1)

const route = useRoute()
const mobileOpen = ref(false)

// Close mobile menu on navigation
watch(() => route.fullPath, () => { mobileOpen.value = false })
</script>

<template>
  <header class="bg-surface-900/80 backdrop-blur-md border-b border-surface-700/50 sticky top-0 z-50">
    <div class="max-w-screen-2xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between">
      <!-- Logo (centered over sidebar width) -->
      <router-link to="/" class="hidden md:flex items-center justify-center w-72 shrink-0 group">
        <img v-if="settings.logo" :src="settings.logo" :alt="settings.name" class="plik-logo-img h-8" />
        <span v-else class="plik-logo-text text-2xl font-bold italic text-accent-400 tracking-tight
                     group-hover:text-accent-300 transition-colors duration-200">
          {{ settings.name }}
        </span>
      </router-link>
      <router-link to="/" class="md:hidden flex items-center gap-2 group"
                   @click="mobileOpen = false">
        <img v-if="settings.logo" :src="settings.logo" :alt="settings.name" class="plik-logo-img h-8" />
        <span v-else class="plik-logo-text text-2xl font-bold italic text-accent-400 tracking-tight
                     group-hover:text-accent-300 transition-colors duration-200">
          {{ settings.name }}
        </span>
      </router-link>

      <!-- Nav Links (desktop) -->
      <nav class="hidden sm:flex items-center gap-1">
        <router-link to="/" class="btn-ghost text-sm">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
          </svg>
          {{ $t('header.upload') }}
        </router-link>

        <router-link v-if="isFeatureEnabled('clients')"
           to="/clients"
           class="btn-ghost text-sm">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          {{ $t('header.cli') }}
        </router-link>

        <a v-if="isFeatureEnabled('github')"
           href="https://root-gg.github.io/plik/"
           target="_blank"
           rel="noopener noreferrer"
           class="btn-ghost text-sm">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
          </svg>
          {{ $t('header.documentation') }}
        </a>

        <a v-if="isFeatureEnabled('github')"
           href="https://github.com/root-gg/plik"
           target="_blank"
           rel="noopener noreferrer"
           class="btn-ghost text-sm">
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
          </svg>
          GitHub
        </a>

        <!-- Separator + Theme picker -->
        <template v-if="showThemePicker">
          <div class="w-px h-5 bg-surface-700/50 mx-1"></div>
          <ThemePicker>
            <span>{{ $t('header.theme') }}</span>
          </ThemePicker>
        </template>

        <template v-if="showLanguagePicker">
          <div class="w-px h-5 bg-surface-700/50 mx-1"></div>
          <LanguagePicker>
            <span>{{ $t('header.language') }}</span>
          </LanguagePicker>
        </template>

        <!-- Separator before auth -->
        <div v-if="isFeatureEnabled('authentication')"
             class="w-px h-5 bg-surface-700/50 mx-1"></div>

        <!-- Logged in: Username link -->
        <router-link v-if="auth.user"
                     to="/home"
                     class="btn-ghost text-sm">
          <img v-if="auth.user.profilePicture"
               :src="auth.user.profilePicture"
               alt="Profile"
               class="w-5 h-5 rounded-full object-cover"
               referrerpolicy="no-referrer" />
          <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
          </svg>
          {{ auth.user.login || auth.user.name || $t('header.account') }}
        </router-link>

        <router-link v-if="auth.user?.admin"
                     to="/admin"
                     class="btn-ghost text-sm">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z" />
          </svg>
          {{ $t('header.admin') }}
        </router-link>

        <!-- Not logged in: Login link -->
        <router-link v-if="!auth.user && isFeatureEnabled('authentication')"
                     to="/login"
                     class="btn-ghost text-sm">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
          {{ $t('header.login') }}
        </router-link>
      </nav>

      <!-- Mobile menu button -->
      <button class="sm:hidden btn-ghost p-2" @click="mobileOpen = !mobileOpen">
        <!-- Hamburger / X icon -->
        <svg v-if="!mobileOpen" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
        <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <!-- Mobile dropdown menu -->
    <nav v-if="mobileOpen"
         class="sm:hidden border-t border-surface-700/50 bg-surface-900/95 backdrop-blur-md animate-fade-in">
      <div class="px-4 py-3 space-y-1">
        <router-link to="/"
                     class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-surface-200
                            hover:bg-surface-700/50 hover:text-surface-100 transition-colors">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
          </svg>
          {{ $t('header.upload') }}
        </router-link>

        <router-link v-if="isFeatureEnabled('clients')"
                     to="/clients"
                     class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-surface-200
                            hover:bg-surface-700/50 hover:text-surface-100 transition-colors">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          {{ $t('header.cli') }}
        </router-link>

        <a v-if="isFeatureEnabled('github')"
           href="https://root-gg.github.io/plik/"
           target="_blank"
           rel="noopener noreferrer"
           class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-surface-200
                  hover:bg-surface-700/50 hover:text-surface-100 transition-colors">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
          </svg>
          {{ $t('header.documentation') }}
        </a>

        <a v-if="isFeatureEnabled('github')"
           href="https://github.com/root-gg/plik"
           target="_blank"
           rel="noopener noreferrer"
           class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-surface-200
                  hover:bg-surface-700/50 hover:text-surface-100 transition-colors">
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
          </svg>
          GitHub
        </a>

        <!-- Theme separator + picker (mobile) -->
        <template v-if="showThemePicker">
          <div class="border-t border-surface-700/50 my-1"></div>
          <ThemePicker
              button-class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-surface-200
                            hover:bg-surface-700/50 hover:text-surface-100 transition-colors w-full">
            <span>{{ $t('header.theme') }}</span>
          </ThemePicker>
        </template>

        <!-- Language picker (mobile) -->
        <template v-if="showLanguagePicker">
          <div class="border-t border-surface-700/50 my-1"></div>
          <LanguagePicker
              button-class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-surface-200
                            hover:bg-surface-700/50 hover:text-surface-100 transition-colors w-full">
            <span>{{ $t('header.language') }}</span>
          </LanguagePicker>
        </template>
        <div v-if="showThemePicker || showLanguagePicker || isFeatureEnabled('authentication')" class="border-t border-surface-700/50 my-1"></div>

        <router-link v-if="auth.user"
                     to="/home"
                     class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-surface-200
                            hover:bg-surface-700/50 hover:text-surface-100 transition-colors">
          <img v-if="auth.user.profilePicture"
               :src="auth.user.profilePicture"
               alt="Profile"
               class="w-5 h-5 rounded-full object-cover"
               referrerpolicy="no-referrer" />
          <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
          </svg>
          {{ auth.user.login || auth.user.name || $t('header.account') }}
        </router-link>

        <router-link v-if="auth.user?.admin"
                     to="/admin"
                     class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-surface-200
                            hover:bg-surface-700/50 hover:text-surface-100 transition-colors">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z" />
          </svg>
          {{ $t('header.admin') }}
        </router-link>

        <router-link v-if="!auth.user && isFeatureEnabled('authentication')"
                     to="/login"
                     class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-surface-200
                            hover:bg-surface-700/50 hover:text-surface-100 transition-colors">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
          {{ $t('header.login') }}
        </router-link>
      </div>
    </nav>
  </header>

  <!-- Impersonation Banner -->
  <div v-if="auth.impersonatedUser"
       class="bg-orange-500/20 border-b-2 border-orange-500/50 px-4 py-2 flex items-center justify-center gap-3 text-sm relative z-40">
    <span class="text-orange-500 font-medium">⚠️ {{ $t('header.impersonating') }}
      <strong class="text-orange-400">{{ auth.impersonatedUser.login || auth.impersonatedUser.name || auth.impersonatedUser.id }}</strong>
    </span>
    <button @click="clearImpersonate"
            class="text-xs bg-orange-500/30 hover:bg-orange-500/50 text-orange-400 font-semibold rounded px-3 py-1 transition-colors">
       {{ $t('header.stopImpersonating') }}
    </button>
  </div>
</template>

