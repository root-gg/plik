// Auth store — reactive user session state
// On app init, try to fetch /me to check if already logged in (from cookie).
// Provides login(), logout(), impersonate(), and reactive user object.

import { reactive } from 'vue'
import { login as apiLogin, logout as apiLogout, getUser, setImpersonateUser } from './api.js'
import { syncThemeFromUser, syncLanguageFromUser } from './settings.js'

export const auth = reactive({
    user: null,       // { id, provider, login, admin, ... } or null
    loading: false,
    error: null,
    impersonatedUser: null, // user object being impersonated (admin only)
    originalUser: null,     // the real admin user (preserved during impersonation)
})

/**
 * Check existing session on page load.
 * If a valid plik-session cookie exists, /me returns the user.
 */
export async function checkSession() {
    auth.loading = true
    auth.error = null
    try {
        auth.user = await getUser()
        auth.originalUser = auth.user
        syncThemeFromUser(auth.user)
        syncLanguageFromUser(auth.user)
    } catch (err) {
        // 401 = not logged in, that's fine
        auth.user = null
    } finally {
        auth.loading = false
    }
}

/**
 * Log in with local credentials.
 * @returns {boolean} true if login succeeded
 */
export async function login(loginName, password) {
    auth.loading = true
    auth.error = null
    try {
        await apiLogin(loginName, password)
        // Now fetch user info (the session cookie was just set)
        auth.user = await getUser()
        auth.originalUser = auth.user
        syncThemeFromUser(auth.user)
        syncLanguageFromUser(auth.user)
        return true
    } catch (err) {
        auth.error = err.message || 'Login failed'
        auth.user = null
        return false
    } finally {
        auth.loading = false
    }
}

/**
 * Log out (clears server session + cookies).
 */
export async function logout() {
    try {
        await apiLogout()
    } catch {
        // Ignore errors on logout
    }
    clearImpersonate()
    auth.user = null
    auth.originalUser = null
    auth.error = null
}

/**
 * Impersonate a user (admin only).
 * Sets the X-Plik-Impersonate header for all subsequent API calls.
 */
export async function impersonate(user) {
    if (!user || !auth.originalUser?.admin) return
    if (user.id === auth.originalUser.id) return // can't impersonate yourself

    setImpersonateUser(user.id)
    auth.impersonatedUser = user

    try {
        // Refresh /me — server will return the impersonated user
        auth.user = await getUser()
    } catch (err) {
        // Revert on failure
        clearImpersonate()
    }
}

/**
 * Stop impersonating and restore the real admin user.
 */
export function clearImpersonate() {
    setImpersonateUser(null)
    auth.impersonatedUser = null
    if (auth.originalUser) {
        auth.user = auth.originalUser
    }
}
