# Internationalization (i18n)

Plik supports multiple languages in the web interface, with automatic detection and user preference persistence.

## How It Works

1. **Auto-detection**: By default, Plik detects the user's preferred language from their browser settings
2. **Local storage**: Once a language is selected, it's saved to the browser's localStorage
3. **Server sync**: For authenticated users, the language preference is saved to their account and follows them across devices

## Built-in Languages

| Code | Language |
|------|----------|
| `en` | English |
| `fr` | Français |

## Configuration

Language settings are configured in `settings.json`:

```jsonc
{
  // Default language: "auto" (detect from browser), "en", "fr", etc.
  "language": "auto",

  // Available languages in the picker ("*" = all built-in, [] = English only)
  "languages": ["*"]
}
```

### Language Picker

Users can switch languages via the globe icon in the header. The selected language persists across page reloads via localStorage. For authenticated users, the preference is also saved to their account.

To control which languages appear in the picker:

```jsonc
// All built-in languages (default)
"languages": ["*"]

// No language picker (English only)
"languages": []

// Only specific languages
"languages": ["auto", "en", "fr"]
```

When only one language is configured, the picker is hidden automatically.

## Adding a New Language

Follow these steps to add a new language to Plik:

### 1. Create the Locale File

Copy the English locale as a template:

```bash
cp webapp/src/locales/en.json webapp/src/locales/XX.json
```

Replace `XX` with the [ISO 639-1 language code](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) (e.g., `de` for German, `es` for Spanish).

### 2. Translate All Keys

Open your new `XX.json` file and translate all values. Keep the JSON keys unchanged — only translate the values.

::: tip Punctuation
Different languages have different punctuation rules. For example, French uses a space before colons (` :`), while English does not. Include any language-specific punctuation directly in the translation values.
:::

### 3. Register the Language

Add a flag SVG to `webapp/public/flags/`:

```bash
# Use any SVG flag — e.g. from https://github.com/lipis/flag-icons
cp your-flag.svg webapp/public/flags/XX.svg
```

Then edit `webapp/src/settings.js` and add your language to the `BUILTIN_LANGUAGES` array:

```javascript
export const BUILTIN_LANGUAGES = [
    { name: 'auto', label: 'Auto' },
    { name: 'en', label: 'English', flag: '/flags/en.svg' },
    { name: 'fr', label: 'Français', flag: '/flags/fr.svg' },
    // Add your language:
    { name: 'XX', label: 'Your Language', flag: '/flags/XX.svg' },
]
```

The `flag` field is a path to an SVG file in `webapp/public/flags/`. This allows custom flags (e.g. regional languages) that aren't in any icon library.

### 4. Import the Locale

Edit `webapp/src/i18n.js` and add the import:

```javascript
import en from './locales/en.json'
import fr from './locales/fr.json'
import XX from './locales/XX.json'  // [!code ++]

const i18n = createI18n({
    // ...
    messages: { en, fr, XX },  // [!code ++]
})
```

### 5. Build and Test

```bash
cd webapp
npm test        # Run unit tests
npm run build   # Verify production build
```

### 6. Configure (Optional)

If you want your language available in the picker by default, the `["*"]` wildcard in `settings.json` will automatically include it. For custom deployments, add it explicitly:

```jsonc
"languages": ["auto", "en", "fr", "XX"]
```
