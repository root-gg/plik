<script setup>
import { ref, watch, onMounted, onBeforeUnmount, computed, nextTick } from 'vue'
import { EditorState, Compartment } from '@codemirror/state'
import { EditorView, keymap, placeholder as cmPlaceholder, lineNumbers, highlightActiveLineGutter, highlightActiveLine } from '@codemirror/view'
import { defaultKeymap, indentWithTab, history, historyKeymap } from '@codemirror/commands'
import { bracketMatching, foldGutter, indentOnInput, syntaxHighlighting, defaultHighlightStyle } from '@codemirror/language'
import { oneDark } from '@codemirror/theme-one-dark'
import { languages } from '@codemirror/language-data'

const props = defineProps({
  modelValue: { type: String, default: '' },
  filename: { type: String, default: '' },
  readonly: { type: Boolean, default: false },
  placeholder: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'language-detected'])

const editorContainer = ref(null)
let view = null
const languageCompartment = new Compartment()
const themeCompartment = new Compartment()
let themeObserver = null

// Map file extension to language
function getLanguageFromFilename(filename) {
  if (!filename) return null
  const ext = filename.split('.').pop()?.toLowerCase()
  if (!ext) return null

  // CodeMirror language-data stores extensions with a leading dot (e.g. ".xml")
  const dotExt = `.${ext}`

  // Find matching language from CodeMirror's language-data
  for (const lang of languages) {
    if (lang.extensions && lang.extensions.includes(dotExt)) {
      return lang
    }
    // Also check alias-based matching
    if (lang.alias && lang.alias.includes(ext)) {
      return lang
    }
  }
  return null
}

const detectedLanguage = computed(() => {
  const lang = getLanguageFromFilename(props.filename)
  if (lang) return lang.name
  if (props.filename) {
    const ext = props.filename.split('.').pop()?.toLowerCase()
    if (ext) return ext.toUpperCase()
  }
  return 'Plain Text'
})

// JSON prettify / validate
const isJson = computed(() => detectedLanguage.value === 'JSON')
const jsonError = ref(null)
const justValidated = ref(false)
const justPrettified = ref(false)
let feedbackResetTimeout = null

function showFeedback(which) {
  if (which === 'validate') justValidated.value = true
  else justPrettified.value = true
  if (feedbackResetTimeout) clearTimeout(feedbackResetTimeout)
  feedbackResetTimeout = setTimeout(() => {
    justValidated.value = false
    justPrettified.value = false
  }, 1500)
}

function validateJson() {
  const content = view ? view.state.doc.toString() : props.modelValue
  if (!content) return
  try {
    JSON.parse(content)
  } catch (e) {
    jsonError.value = e.message
    return
  }
  jsonError.value = null
  showFeedback('validate')
}

function prettifyJson() {
  const content = view ? view.state.doc.toString() : props.modelValue
  if (!content) return

  let parsed
  try {
    parsed = JSON.parse(content)
  } catch (e) {
    jsonError.value = e.message
    return
  }

  jsonError.value = null
  const pretty = JSON.stringify(parsed, null, 2)

  // In read-only mode we can't dispatch to CM directly — emit so parent updates its ref
  if (props.readonly) {
    emit('update:modelValue', pretty)
  } else if (view) {
    view.dispatch({
      changes: { from: 0, to: view.state.doc.length, insert: pretty }
    })
  }

  showFeedback('prettify')
}

// Shared CodeMirror style overrides (font, sizing, gutters)
const plikBaseTheme = EditorView.theme({
  '&': { fontSize: '13px', backgroundColor: 'transparent' },
  '.cm-content': {
    fontFamily: 'ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, "DejaVu Sans Mono", monospace',
    minHeight: '120px',
    caretColor: 'var(--color-accent-400)',
  },
  '&.cm-focused .cm-content': { caretColor: 'var(--color-accent-400)' },
  '.cm-scroller': { overflow: 'auto' },
  '.cm-gutters': {
    backgroundColor: 'color-mix(in srgb, var(--color-surface-800) 40%, transparent)',
    color: 'var(--color-surface-500)',
    border: 'none',
    borderRight: '1px solid color-mix(in srgb, var(--color-surface-700) 50%, transparent)',
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'color-mix(in srgb, var(--color-surface-700) 40%, transparent)',
    color: 'var(--color-surface-300)',
  },
  '.cm-activeLine': {
    backgroundColor: 'color-mix(in srgb, var(--color-surface-700) 25%, transparent)',
  },
  '&.cm-focused .cm-cursor': { borderLeftColor: 'var(--color-accent-400)' },
  '&.cm-focused .cm-selectionBackground, ::selection': {
    backgroundColor: 'color-mix(in srgb, var(--color-accent-500) 20%, transparent)',
    color: 'inherit',
  },
  '.cm-selectionBackground': {
    backgroundColor: 'color-mix(in srgb, var(--color-accent-500) 15%, transparent)',
  },
  '.cm-foldGutter': { color: 'var(--color-surface-500)' },
  '.cm-tooltip': {
    backgroundColor: 'var(--color-surface-800)',
    border: '1px solid var(--color-surface-700)',
    color: 'var(--color-surface-100)',
  },
  '.cm-placeholder': { color: 'var(--color-surface-500)', fontStyle: 'italic' },
})

function isDarkTheme() {
  const cs = getComputedStyle(document.documentElement).colorScheme
  return cs !== 'light'
}

function getThemeExtensions() {
  if (isDarkTheme()) {
    return [oneDark]
  }
  return [syntaxHighlighting(defaultHighlightStyle)]
}

async function createEditor() {
  if (!editorContainer.value) return

  // Build extensions
  const extensions = [
    lineNumbers(),
    highlightActiveLineGutter(),
    highlightActiveLine(),
    history(),
    bracketMatching(),
    foldGutter(),
    indentOnInput(),
    plikBaseTheme,
    themeCompartment.of(getThemeExtensions()),
    keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
    EditorView.lineWrapping,
  ]

  if (props.placeholder) {
    extensions.push(cmPlaceholder(props.placeholder))
  }

  if (props.readonly) {
    extensions.push(EditorState.readOnly.of(true))
    extensions.push(EditorView.editable.of(false))
  } else {
    // Listen for changes and emit v-model updates
    extensions.push(EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        emit('update:modelValue', update.state.doc.toString())
      }
    }))
  }

  // Load language support via compartment (allows dynamic reconfiguration)
  const langDesc = getLanguageFromFilename(props.filename)
  let langExtension = []
  if (langDesc) {
    try {
      langExtension = await langDesc.load()
    } catch (e) {
      console.error('Failed to load language support:', e)
    }
  }
  extensions.push(languageCompartment.of(langExtension))

  const state = EditorState.create({
    doc: props.modelValue || '',
    extensions,
  })

  view = new EditorView({
    state,
    parent: editorContainer.value,
  })
}

function destroyEditor() {
  if (view) {
    view.destroy()
    view = null
  }
}

// Reconfigure language when filename changes (without destroying the editor)
watch(() => props.filename, async () => {
  if (!view) return
  const langDesc = getLanguageFromFilename(props.filename)
  let langExtension = []
  if (langDesc) {
    try {
      langExtension = await langDesc.load()
    } catch (e) {
      console.error('Failed to load language support:', e)
    }
  }
  view.dispatch({
    effects: languageCompartment.reconfigure(langExtension)
  })
})

let detectionTimeout = null

// Content-based language detection via highlight.js (lazy-loaded)
let hljs = null

// Restrict auto-detection to common languages to avoid false positives
const DETECT_LANGUAGES = [
  'json', 'xml', 'html', 'css',
  'javascript', 'typescript',
  'python', 'ruby', 'perl', 'php',
  'java', 'c', 'cpp', 'csharp',
  'go', 'rust', 'swift', 'kotlin',
  'bash', 'shell', 'powershell',
  'sql', 'yaml', 'markdown',
  'dockerfile', 'ini', 'toml',
]

// Minimum relevance score to accept a detection (0–∞, higher = more confident)
const MIN_RELEVANCE = 5
let hasDetected = false

async function detectLanguageFromContent(content) {
  if (hasDetected) return
  if (!content || content.length < 10) return

  // Lazy-load highlight.js on first detection call
  if (!hljs) {
    try {
      const mod = await import('highlight.js')
      hljs = mod.default
    } catch (e) {
      console.error('Failed to load highlight.js:', e)
      return
    }
  }

  // Cap detection input to avoid perf issues on very large files
  const sample = content.length > 50000 ? content.slice(0, 50000) : content
  const result = hljs.highlightAuto(sample, DETECT_LANGUAGES)
  if (!result.language || result.relevance < MIN_RELEVANCE) return

  // Map hljs language name to a CodeMirror language descriptor
  const lang = languages.find(l =>
    l.name.toLowerCase() === result.language ||
    (l.alias && l.alias.includes(result.language))
  )
  if (lang?.extensions?.length) {
    hasDetected = true
    emit('language-detected', { language: lang.name, extension: lang.extensions[0].replace(/^\./, '') })
  }
}

// Sync external changes to modelValue (e.g. clipboard paste before editor is ready)
watch(() => props.modelValue, (newVal) => {
  if (view && newVal !== view.state.doc.toString()) {
    view.dispatch({
      changes: { from: 0, to: view.state.doc.length, insert: newVal || '' }
    })
  }

  // Debounce detection
  if (detectionTimeout) clearTimeout(detectionTimeout)
  detectionTimeout = setTimeout(() => {
    detectLanguageFromContent(newVal)
  }, 1000)
})


onMounted(async () => {
  await createEditor()
  // Detect language for initial content (e.g. paste from main page opens editor pre-filled)
  if (props.modelValue) {
    detectionTimeout = setTimeout(() => {
      detectLanguageFromContent(props.modelValue)
    }, 500)
  }

  // Watch for theme changes and reconfigure editor
  themeObserver = new MutationObserver(() => {
    if (view) {
      view.dispatch({ effects: themeCompartment.reconfigure(getThemeExtensions()) })
    }
  })
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme'],
  })
})

onBeforeUnmount(() => {
  if (detectionTimeout) clearTimeout(detectionTimeout)
  if (feedbackResetTimeout) clearTimeout(feedbackResetTimeout)
  if (themeObserver) { themeObserver.disconnect(); themeObserver = null }
  destroyEditor()
})
</script>

<template>
  <div class="code-editor-wrapper">
    <!-- Language badge + actions -->
    <div class="flex items-center justify-between px-3 py-1.5 border-b border-surface-700/50">
      <div class="flex items-center gap-2">
        <svg class="w-3.5 h-3.5 text-accent-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
        </svg>
        <span class="text-xs text-surface-400 font-medium">{{ detectedLanguage }}</span>
      </div>
      <div class="flex items-center gap-2">
        <!-- JSON Validate button -->
        <button v-if="isJson"
                class="flex items-center gap-1 text-xs transition-colors px-2 py-0.5 rounded"
                :class="justValidated
                  ? 'text-green-400'
                  : 'text-surface-400 hover:text-accent-400 hover:bg-surface-700/50'"
                @click="validateJson">
          <svg v-if="justValidated" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {{ justValidated ? $t('common.valid') : $t('common.validate') }}
        </button>
        <!-- JSON Prettify button -->
        <button v-if="isJson"
                class="flex items-center gap-1 text-xs transition-colors px-2 py-0.5 rounded"
                :class="justPrettified
                  ? 'text-green-400'
                  : 'text-surface-400 hover:text-accent-400 hover:bg-surface-700/50'"
                @click="prettifyJson">
          <svg v-if="justPrettified" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M4 6h16M4 12h8m-8 6h16" />
          </svg>
          {{ justPrettified ? $t('common.prettified') : $t('common.prettify') }}
        </button>
        <div v-if="readonly" class="flex items-center gap-1">
          <svg class="w-3 h-3 text-surface-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          <span class="text-xs text-surface-500">{{ $t('common.readOnly') }}</span>
        </div>
      </div>
    </div>
    <!-- JSON error banner -->
    <div v-if="jsonError"
         class="flex items-center gap-2 px-3 py-1.5 bg-danger-500/10 border-b border-danger-500/30 text-xs text-danger-400">
      <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span class="truncate">{{ jsonError }}</span>
      <button class="ml-auto text-danger-400 hover:text-surface-100 shrink-0" @click="jsonError = null">
        <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
    <!-- Editor mount point -->
    <div ref="editorContainer" class="code-editor-container" />
  </div>
</template>

<style>
.code-editor-wrapper {
  border-radius: 0.5rem;
  overflow: hidden;
  background-color: color-mix(in srgb, var(--color-surface-900) 80%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-surface-700) 50%, transparent);
}

.code-editor-container .cm-editor {
  max-height: 60vh;
  min-height: 150px;
}

.code-editor-container .cm-editor.cm-focused {
  outline: none;
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-accent-500) 30%, transparent);
}

.code-editor-container .cm-scroller {
  overflow: auto;
}
</style>
