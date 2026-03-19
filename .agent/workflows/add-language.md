---
description: Add a new language to Plik's i18n system (locale file, flag, registration, docs)
---

# Add a New Language

End-to-end workflow for adding a language to Plik's webapp internationalization.

// turbo-all

## Steps

### 1. Validate prerequisites

Check that the language code doesn't already exist:

```bash
ls webapp/src/locales/*.json
```

Confirm the language code (ISO 639-1, e.g. `ja`, `zh`, `ko`, `ru`) with the user.

### 2. Create the locale file

Use `webapp/src/locales/en.json` as the source of truth. Translate all values into the target language.

**Translation rules:**
- **Abbreviations/product names**: Keep as-is: `CLI`, `TTL`, `E2EE`, `MD5`, `Markdown`, `HTTP`
- **Semantic distinctions**: `passphrase` and `password` are both present in the UI — they MUST translate to **different** words. Using the same word for both breaks UX. Same applies to `token` vs other concepts, `admin` vs `user`, `streaming` vs `upload/download`. Native translations are fine as long as the distinction is clear.
- **Placeholders**: Preserve `{placeholder}` tokens exactly as-is
- **Plurals**: Use pipe-separated forms (`{count} file | {count} files`). Some languages need 3+ forms (e.g. Polish: `1 | few | many`). Languages with no grammatical plurals (e.g. Chinese, Japanese) use a single form with no pipes.
- **Punctuation**: Follow the target language's punctuation rules (e.g. French uses spaces before `:`, `?`, `!`, `;`)
- **languagePicker**: Must include entries for ALL supported languages (check existing locales for the full list)
- **Consistency**: Match the style/register of existing translations (informal but professional). Use the same translated term for a concept everywhere (e.g. don't mix "jeton" and "token" in the same locale).

Save as `webapp/src/locales/XX.json` (2-space indent to match `en.json`).

### 3. Create a flag SVG

Create a simple flag SVG at `webapp/public/flags/XX.svg`. Use the country's official flag colors. Keep it minimal — basic geometric shapes only (rects, paths), no complex emblems.

Example (French tricolor):
```svg
<svg viewBox="0 0 3 2" xmlns="http://www.w3.org/2000/svg">
  <rect fill="#002395" width="1" height="2"/>
  <rect fill="#fff" x="1" width="1" height="2"/>
  <rect fill="#ED2939" x="2" width="1" height="2"/>
</svg>
```

### 4. Register the language

#### `webapp/src/settings.js`
Add an entry to `BUILTIN_LANGUAGES`:
```javascript
{ name: 'XX', label: 'NativeName', flag: '/flags/XX.svg' },
```
The label must be the language's **endonym** (e.g. "Deutsch" not "German", "日本語" not "Japanese").

#### `webapp/src/i18n.js`
Add import and register in messages:
```javascript
import XX from './locales/XX.json'
// ...
messages: { en, fr, ..., XX },
```

### 5. Update languagePicker in ALL existing locales

Every locale file's `languagePicker` section must include an entry for the new language. The value is always the endonym.

```bash
# Quick check — every locale should have the same languagePicker keys
cd webapp/src/locales && for f in *.json; do echo "=== $f ==="; jq '.languagePicker | keys' "$f"; done
```

### 6. Run the key sync test

```bash
cd webapp && npx vitest run src/__tests__/locales.test.js
```

This validates:
- No missing or extra keys vs `en.json`
- No empty translation values
- All `{placeholder}` tokens preserved

### 7. Build and verify

```bash
make frontend 2>&1 | tail -5
make test-frontend 2>&1 | tail -10
```

### 8. Update docs

Add the new language to the table in:
- `docs/features/internationalization.md` — "Built-in Languages" table
- `webapp/ARCHITECTURE.md` — verify the locale wildcard `src/locales/*.json` still covers it (it should)

### 9. Visual check

Start the dev server and verify the new language appears in the language picker, the flag renders correctly, and translations display properly across Upload, Download, Home, Admin, and Login views.

### 10. Update review-language workflow

Propose updates to `.agent/workflows/review-language.md` with language-specific rules for the new locale:
- Add the language to the **Punctuation rules** table (step 4) if it has special punctuation conventions
- Add the language to the **Plural forms** table (step 5) with the correct number of forms and pattern
- Add any language-specific loanword exceptions to step 3 if the language has common false friends

### 11. Run translation review

Run the `/review-language XX` workflow to do a thorough quality review of the new translations (semantic distinction audit, punctuation rules, plural forms, contextual spot-check).
