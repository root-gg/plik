// Reactive config store
// Fetches server configuration once and provides reactive state

import { reactive, ref } from 'vue'
import { getConfig, getVersion, setDownloadURL } from './api.js'

export const config = reactive({
    loaded: false,
    error: null,

    // Server limits
    maxFileSize: 0,
    maxUserSize: 0,
    maxFilePerUpload: 1000,
    streamTimeout: 0,
    defaultTTL: 0,
    maxTTL: 0,

    // Feature flags (values: "enabled", "disabled", "forced", "default")
    feature_authentication: 'disabled',
    feature_local_login: 'enabled',
    feature_delete_account: 'enabled',
    feature_one_shot: 'default',
    feature_removable: 'default',
    feature_stream: 'default',
    feature_password: 'default',
    feature_comments: 'disabled',
    feature_set_ttl: 'default',
    feature_extend_ttl: 'default',
    feature_clients: 'default',
    feature_api_tokens: 'enabled',
    feature_github: 'default',
    feature_text: 'default',
    feature_e2ee: 'enabled',

    // Download domain / URL
    downloadDomain: '',
    downloadURL: '',

    // OAuth providers (set by server config)
    googleAuthentication: false,
    ovhAuthentication: false,
    oidcAuthentication: false,
    oidcProviderName: 'OpenID',

    // Abuse contact
    abuseContact: '',
})

export const version = ref(null)

export async function loadConfig() {
    try {
        const data = await getConfig()
        Object.assign(config, data)
        config.loaded = true
        // Prefer downloadURL (domain+path) if present; fall back to downloadDomain for old servers
        setDownloadURL(config.downloadURL || config.downloadDomain || '')
    } catch (err) {
        config.error = err.message || 'Failed to load configuration'
        console.error('Failed to load config:', err)
    }
}

export async function loadVersion() {
    try {
        version.value = await getVersion()
    } catch (err) {
        console.error('Failed to load version:', err)
    }
}

// Feature flag helpers
export function isFeatureEnabled(feature) {
    const key = `feature_${feature}`
    const value = config[key]
    return value !== 'disabled'
}

export function isFeatureForced(feature) {
    const key = `feature_${feature}`
    return config[key] === 'forced'
}

// Returns true when the feature should be ON by default (either 'default' or 'forced')
export function isFeatureDefaultOn(feature) {
    const key = `feature_${feature}`
    const value = config[key]
    return value === 'default' || value === 'forced'
}
