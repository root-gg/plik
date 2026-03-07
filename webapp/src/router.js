import { createRouter, createWebHashHistory } from 'vue-router'
import { config } from './config.js'
import { auth } from './authStore.js'
import RootView from './views/RootView.vue'
import LoginView from './views/LoginView.vue'
import HomeView from './views/HomeView.vue'
import AdminView from './views/AdminView.vue'
import ClientsView from './views/ClientsView.vue'
import CLIAuthView from './views/CLIAuthView.vue'

const routes = [
    {
        path: '/',
        name: 'root',
        component: RootView,
    },
    {
        path: '/login',
        name: 'login',
        component: LoginView,
    },
    {
        path: '/home',
        redirect: '/home/stats',
    },
    {
        path: '/home/:tab(stats|uploads|tokens)',
        name: 'home',
        component: HomeView,
        meta: { requiresAuth: true },
    },
    {
        path: '/admin',
        redirect: '/admin/stats',
    },
    {
        path: '/admin/:tab(stats|users|uploads|events)',
        name: 'admin',
        component: AdminView,
        meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
        path: '/clients',
        name: 'clients',
        component: ClientsView,
    },
    {
        path: '/cli-auth',
        name: 'cli-auth',
        component: CLIAuthView,
    },
    {
        path: '/upload/:id',
        redirect: to => {
            // Redirect old URLs to new format
            return { path: '/', query: { id: to.params.id, ...to.query } }
        }
    },
]

const router = createRouter({
    history: createWebHashHistory(),
    routes,
})

// Redirect to login when authentication is required and user is not logged in.
// The intended destination is saved in sessionStorage so it survives OAuth round-trips.
router.beforeEach((to) => {
    // If already authenticated and visiting login, redirect to the intended destination
    if (to.name === 'login' && auth.user) {
        const redirect = sessionStorage.getItem('plik-auth-redirect') || '/'
        sessionStorage.removeItem('plik-auth-redirect')
        return redirect
    }

    // CLI auth approval always requires authentication (regardless of auth mode)
    if (to.name === 'cli-auth' && !auth.user) {
        sessionStorage.setItem('plik-auth-redirect', to.fullPath)
        return { name: 'login' }
    }

    // Pages marked requiresAuth need a logged-in user
    if (to.meta.requiresAuth && !auth.user) {
        sessionStorage.setItem('plik-auth-redirect', to.fullPath)
        return { name: 'login' }
    }

    // Admin pages require an admin user (user is authenticated at this point)
    if (to.meta.requiresAdmin && !auth.user?.admin) {
        return '/'
    }

    // Forced authentication: redirect everything else to login unless exempted
    if (config.feature_authentication !== 'forced') return true
    if (auth.user) return true
    if (to.name === 'login') return true
    // Allow CLI client download page without authentication
    if (to.name === 'clients') return true
    // Allow download pages — they have ?id= query
    if (to.name === 'root' && to.query.id) return true
    sessionStorage.setItem('plik-auth-redirect', to.fullPath)
    return { name: 'login' }
})

export default router
