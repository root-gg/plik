# Web UI

Plik serves a web interface on the same port as the API by default.

## Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `NoWebInterface` | `false` | Disable the web UI entirely |
| `WebappDirectory` | `../webapp/dist` | Path to static files |

## Customization

Place or volume-mount files into the webapp root (`server/webapp/dist/`, or `/home/plik/server/webapp/dist/` in Docker). Changes take effect on next page load — no rebuild required.

### Settings

Edit `settings.json` to change the app name, theme, background, and more. The file uses JSONC (supports `//` comments):

```jsonc
{
  // Title displayed in the header logo and browser tab
  "name": "Plik",

  // Path to a logo image (e.g. "/img/logo.png"). When set, replaces the text logo.
  "logo": "",

  // Theme: "dark", "light", "auto" (OS preference), or a custom theme name
  // matching a CSS file in the themes/ directory (e.g. "solarized-dark")
  "theme": "auto",

  // Available themes in the picker ("*" = all built-in themes, [] = dark only)
  // Examples: ["dark", "light"] or ["dark", "catppuccin-mocha", "nord"]
  "themes": ["*"],

  // Theme used by "auto" when the OS prefers dark mode (default: "dark")
  "defaultDarkTheme": "dark",

  // Theme used by "auto" when the OS prefers light mode (default: "light")
  "defaultLightTheme": "light",

  // Language: "auto" (detect from browser), "en", "fr", etc.
  // See [Internationalization](/plik/features/internationalization) for details.
  "language": "auto",

  // Available languages in the picker ("*" = all built-in, [] = English only)
  "languages": ["*"],

  // Background image path (e.g. "/img/background.jpg")
  "backgroundImage": "",

  // Fallback background color (CSS value, e.g. "#1a1a2e")
  "backgroundColor": "",

  // Overlay opacity (0–1) — darkens the background for readability
  "overlayOpacity": 0.2,

  // Path to a custom CSS file (e.g. "/css/custom.css")
  "customCSS": "",

  // Path to a custom JavaScript file (e.g. "/js/custom.js")
  "customJS": ""
}
```

Custom CSS and JS are loaded only when their path is set (empty = disabled, no extra HTTP requests).

To use a background image, place it at `img/background.jpg` and set `"backgroundImage": "/img/background.jpg"`.

To change the favicon, replace `favicon.ico`.

### Docker Example

```bash
docker run -p 8080:8080 \
  -v ./settings.json:/home/plik/server/webapp/dist/settings.json:ro \
  -v ./custom.css:/home/plik/server/webapp/dist/css/custom.css:ro \
  rootgg/plik
```

## Custom Themes

Plik ships with `dark` (default) and `light` themes, plus eight community-inspired themes:

| Theme | Type | Based on |
|-------|------|----------|
| `solarized-dark` | Dark | [Solarized](https://ethanschoonover.com/solarized/) |
| `solarized-light` | Light | [Solarized](https://ethanschoonover.com/solarized/) |
| `nord` | Dark | [Nord](https://www.nordtheme.com/) |
| `nord-light` | Light | [Nord](https://www.nordtheme.com/) |
| `catppuccin-mocha` | Dark | [Catppuccin](https://catppuccin.com/) |
| `catppuccin-latte` | Light | [Catppuccin](https://catppuccin.com/) |
| `matrix` | Dark | The Matrix — neon green on black |
| `hexless` | Dark | Bold primary colors — pure RGB fun |

You can also create your own:

1. Copy `themes/TEMPLATE.css` to `themes/my-theme.css`
2. Replace `THEME_NAME` with `my-theme` in the CSS selectors
3. Customize the color values — each token is documented in the template
4. Add `"my-theme"` to the `themes` array in `settings.json` (e.g. `["*", "my-theme"]`)
5. Set `"theme": "my-theme"` in `settings.json`, or let users pick it from the theme picker

Theme files are lazy-loaded on startup and cached by the browser.

### Theme Picker

Users can switch themes via the palette icon in the header navigation bar. The selected theme is stored in the browser's localStorage and persists across page reloads. For authenticated users, the theme preference is also saved to their account — it follows them across devices and browsers automatically.

To control which themes appear in the picker, set the `themes` array in `settings.json`:

```jsonc
// All built-in themes available (default)
"themes": ["*"]

// No theme picker (dark only)
"themes": []

// Only allow one specific theme (disables the picker)
"themes": ["acme"]

// Only allow specific themes
"themes": ["auto", {"name": "acme-light", "label": "Acme Light"}, {"name": "acme-dark", "label": "Acme Dark"}]

// All built-in themes + your custom ones ("*" expands to all built-ins)
"themes": ["*", {"name": "custom-light", "label": "Custom Light"}, {"name": "custom-dark", "label": "Custom Dark"}]
```

When only one theme is configured, the picker is hidden automatically.

#### Custom Auto Defaults

By default, "auto" mode resolves to the built-in `dark` and `light` themes based on the user's OS preference. You can override which themes "auto" uses:

```jsonc
// Use Solarized as the default dark/light pair
"defaultDarkTheme": "custom-dark",
"defaultLightTheme": "custom-light"
```

This is especially useful when deploying custom themes — users who select "auto" will automatically get your preferred dark/light pair.

### Docker

```bash
docker run -p 8080:8080 \
  -v ./settings.json:/home/plik/server/webapp/dist/settings.json:ro \
  -v ./my-theme.css:/home/plik/server/webapp/dist/themes/my-theme.css:ro \
  rootgg/plik
```

## Features

### Inline File Viewer

The web interface includes an inline file viewer for text files (code, logs, markdown, etc.), images, video and audio.

- **Auto-display**: If an upload contains only one viewable file, the viewer is displayed by default.
- **Syntax Highlighting**: Automatic detection of hundreds of languages.
- **JSON Formatting**: Pretty-print and validation buttons for JSON files.
- **Markdown Preview**: Rendered HTML preview with Code/Preview tabs.
- **Image Preview**: Inline display for all image types (`image/*`).
- **Video Playback**: Native HTML5 player with controls for video files (`video/*`).
- **Audio Playback**: Native HTML5 player with controls for audio files (`audio/*`).
