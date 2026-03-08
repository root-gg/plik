<script setup>
import { config } from '../config.js'
import { clampQuota, filterQuotaInput, defaultSizeHint, defaultTTLHint, TTL_UNITS } from '../utils.js'

const props = defineProps({
    modelValue: { type: Boolean, required: true },     // visibility (v-model)
    form: { type: Object, required: true },            // edit form data (v-model:form)
    ttlUnit: { type: Number, default: 60 },            // TTL unit selection (v-model:ttlUnit)
    error: { type: String, default: '' },
    saving: { type: Boolean, default: false },
    title: { type: String, default: 'Edit User' },
    quotaHeader: { type: String, default: 'Quotas' },
    showQuotas: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'update:form', 'update:ttlUnit', 'save'])

function close() {
    emit('update:modelValue', false)
}

function updateField(field, value) {
    emit('update:form', { ...props.form, [field]: value })
}

function updateQuotaField(field, raw, allowDecimal = false) {
    const filtered = filterQuotaInput(raw, allowDecimal)
    emit('update:form', { ...props.form, [field]: filtered })
}

function clampField(field) {
    emit('update:form', { ...props.form, [field]: clampQuota(props.form[field]) })
}
</script>

<template>
  <Teleport to="body">
    <div v-if="modelValue"
         class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
         @mousedown.self="close">
      <div class="glass-card p-4 sm:p-6 max-w-md w-full space-y-5 animate-fade-in max-h-[90vh] overflow-y-auto">
        <h2 class="text-lg font-semibold text-surface-200">{{ title }}</h2>

        <!-- Error -->
        <div v-if="error" class="text-sm text-red-400 bg-red-500/10 rounded-lg px-3 py-2">
          {{ error }}
        </div>

        <!-- Provider & Login (read-only) -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.provider') }}</label>
            <div class="input-field bg-surface-800/50 text-surface-400 cursor-not-allowed">{{ form.provider }}</div>
          </div>
          <div>
            <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.login') }}</label>
            <div class="input-field bg-surface-800/50 text-surface-400 cursor-not-allowed">{{ form.login }}</div>
          </div>
        </div>

        <!-- Name -->
        <div>
          <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.name') }}</label>
          <input type="text" :value="form.name" @input="updateField('name', $event.target.value)"
                 class="input-field w-full" :placeholder="$t('editUser.displayName')" />
        </div>

        <!-- Email -->
        <div>
          <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.email') }}</label>
          <input type="email" :value="form.email" @input="updateField('email', $event.target.value)"
                 class="input-field w-full" :placeholder="$t('editUser.email')" />
        </div>

        <!-- Password (local only) -->
        <div v-if="form.provider === 'local'">
          <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.password') }}</label>
          <input type="password" :value="form.password" @input="updateField('password', $event.target.value)"
                 class="input-field w-full" :placeholder="$t('editUser.leaveBlankToKeep')" />
        </div>

        <!-- Quotas -->
        <template v-if="showQuotas">
          <div class="border-t border-surface-700/50 pt-4 space-y-4">
            <p class="text-xs text-surface-500 uppercase tracking-wider">{{ quotaHeader }}</p>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.maxFileSizeGB') }}</label>
                <input type="text" inputmode="decimal" :value="form.maxFileSize"
                       @input="updateQuotaField('maxFileSize', $event.target.value, true)"
                       @blur="clampField('maxFileSize')"
                       class="input-field w-full" />
                <p class="text-xs text-surface-600 mt-0.5">{{ defaultSizeHint(config.maxFileSize, $t) }}</p>
              </div>
              <div>
                <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.maxUserSizeGB') }}</label>
                <input type="text" inputmode="decimal" :value="form.maxUserSize"
                       @input="updateQuotaField('maxUserSize', $event.target.value, true)"
                       @blur="clampField('maxUserSize')"
                       class="input-field w-full" />
                <p class="text-xs text-surface-600 mt-0.5">{{ defaultSizeHint(config.maxUserSize, $t) }}</p>
              </div>
            </div>

            <div>
              <label class="block text-xs text-surface-500 mb-1">{{ $t('editUser.maxTTL') }}</label>
              <div class="flex gap-2">
                <input type="text" inputmode="numeric" :value="form.maxTTL"
                       @input="updateQuotaField('maxTTL', $event.target.value, false)"
                       @blur="clampField('maxTTL')"
                       class="input-field flex-1" />
                <select :value="ttlUnit" @change="$emit('update:ttlUnit', Number($event.target.value))"
                        class="input-field w-28">
                  <option v-for="u in TTL_UNITS" :key="u.seconds" :value="u.seconds">{{ $t(u.i18nKey) }}</option>
                </select>
              </div>
              <p class="text-xs text-surface-600 mt-0.5">{{ defaultTTLHint(config.maxTTL, $t) }}</p>
            </div>

            <label class="flex items-center gap-2 text-sm text-surface-300 cursor-pointer">
              <input type="checkbox" :checked="form.admin"
                     @change="updateField('admin', $event.target.checked)"
                     class="w-4 h-4 rounded border-surface-600 bg-surface-800
                            text-accent-500 focus:ring-accent-500/30" />
              {{ $t('common.admin') }}
            </label>
          </div>
        </template>

        <!-- Actions -->
        <div class="flex justify-end gap-2 pt-2">
          <button @click="close" class="btn-ghost text-sm px-4 py-2">{{ $t('common.cancel') }}</button>
          <button @click="$emit('save')" :disabled="saving"
                  class="btn-primary px-4 py-2 text-sm">
            {{ saving ? $t('common.saving') : $t('common.save') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
