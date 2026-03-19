---
description: Review translation quality for one or more locale files
---

# Review Language Translations

A thorough quality review of one or more locale files, checking for translation accuracy, consistency, and technical correctness.

// turbo-all

## When to Use

- After generating AI-assisted translations
- When a contributor submits a translation PR
- Periodic quality audit of existing translations
- Invoked via `/review-language` (optionally with a language code, e.g. `/review-language de`)

## Steps

### 1. Determine scope

If the user specifies a language code, review that locale. Otherwise, review all non-English locales.

```bash
ls webapp/src/locales/*.json
```

### 2. Run automated checks

```bash
cd webapp && npx vitest run src/__tests__/locales.test.js
```

This catches: missing/extra keys, empty values, placeholder mismatches.

### 3. Semantic distinction audit

The critical rule is not "keep English words" — it's **preserve semantic distinctions**. Some terms can be translated natively as long as they remain distinct from related concepts. Other terms (abbreviations, product names) should stay as-is.

#### Must stay as-is (abbreviations / product names)

| Term | Context | Why |
|------|---------|-----|
| CLI | Label, section titles | Abbreviation — universally understood |
| TTL | Labels, badges | Abbreviation — no standard translation |
| E2EE | Error messages only | Abbreviation — labels can translate "End-to-End Encryption" natively |
| MD5 | File details | Hash algorithm name |
| Markdown | Comment help text | Product name |

#### Must be distinct from related terms (can be translated or kept as loanword)

| Term | Must be distinct from | Why |
|------|----------------------|-----|
| passphrase | password | **Critical** — these are separate fields in the UI; confusing them breaks UX |
| token | other concepts | Auth concept; can be translated (fr→"jeton", pl→"tokeny") as long as it's consistent |
| streaming | download/upload | Distinct mode; can be translated if the meaning is clear |
| admin | user | Role distinction; can use native word (zh→"管理员") |

Check passphrase/password distinction (the most common mistake):
```bash
cd webapp/src/locales && python3 -c "
import json, os
for fname in sorted(os.listdir('.')):
    if not fname.endswith('.json'):
        continue
    lang = fname.replace('.json', '')
    with open(fname) as f:
        data = json.load(f)
    pp = data.get('uploadSidebar', {}).get('passphrase', '???')
    pw = data.get('uploadSidebar', {}).get('password', '???')
    ok = pp.lower() != pw.lower()
    print(f'{lang}: passphrase={pp!r} vs password={pw!r} {\"✅\" if ok else \"❌ CONFLICT!\"}')
"
```

Check consistency — terms that can be translated must use the **same** translation everywhere:
```bash
cd webapp/src/locales && python3 -c "
import json, os, re
# For each translatable term, collect all translations used across a locale
terms = {
    'token': ['homeView.tokens', 'homeView.createToken', 'uploadCard.token'],
    'admin': ['common.admin', 'header.admin'],
    'streaming': ['uploadSidebar.streaming', 'badges.stream', 'downloadView.streamingUpload'],
}
for fname in sorted(os.listdir('.')):
    if not fname.endswith('.json') or fname == 'en.json':
        continue
    lang = fname.replace('.json', '')
    with open(fname) as f:
        data = json.load(f)
    for term, paths in terms.items():
        vals = set()
        for p in paths:
            parts = p.split('.')
            v = data
            for part in parts:
                v = v.get(part, None) if isinstance(v, dict) else None
            if v:
                # Extract the root word (strip punctuation, case)
                clean = re.sub(r'[:\s….]', '', v).lower()
                vals.add(clean)
        if len(vals) > 1:
            print(f'{lang}: {term} has inconsistent translations: {vals}')
"
```


### 4. Punctuation rules

Check language-specific punctuation conventions:

| Language | Rule |
|----------|------|
| French (fr) | Space before `:`, `?`, `!`, `;` |
| German (de) | No special rules |
| Spanish (es) | Opening `¡` and `¿` for exclamations/questions (optional in short UI strings) |
| Chinese (zh) | No spaces before punctuation; uses full-width punctuation marks（：、？、！） |
| Others | Standard punctuation |

### 5. Plural forms

Verify pipe-separated plurals match the language's plural rules:

| Language | Plural forms | Pattern |
|----------|-------------|---------|
| en, de, es, it, pt, nl | 2 | `singular \| plural` |
| fr | 2 | `singular \| plural` (0 is singular in French) |
| pl | 3 | `one \| few \| many` |
| ru | 3 | `one \| few \| many` |
| ar | 6 | `zero \| one \| two \| few \| many \| other` |
| ja, zh, ko | 1 | No plural form needed |

```bash
# Count pipe-separated forms per locale
cd webapp/src/locales && for f in *.json; do
  max=$(grep -o '|' "$f" | wc -l)
  echo "$f: $max pipe characters"
done
```

### 6. languagePicker completeness

Every locale must list every supported language in its `languagePicker` section:

```bash
cd webapp/src/locales && for f in *.json; do
  count=$(jq '.languagePicker | length' "$f")
  echo "$f: $count entries in languagePicker"
done
```

All files should have the same count.

### 7. Documentation freshness

Verify that docs reflect the current set of supported languages:
- `docs/features/internationalization.md` — "Built-in Languages" table lists all locales
- `webapp/ARCHITECTURE.md` — locale file references are up to date (wildcard `src/locales/*.json` should still cover everything)

### 8. Contextual review (spot-check)

For each locale, spot-check 5-10 keys across different namespaces for:
- **Natural phrasing** — does it sound like a native speaker wrote it? (not machine-translated)
- **Consistent register** — matching the informal-but-professional tone of the app
- **UI fit** — are translations reasonably short? (long translations can break layouts)
- **Correct gender/case** — for gendered languages (German, French, etc.)

Focus on high-visibility strings:
- `uploadView.dropPasteOrClick` (first thing users see)
- `header.*` (always visible)
- `downloadView.deleteUploadMessage` (important confirmation)
- `loginView.signInToAccount`
- Error messages in `api.*`

### 9. Produce review report

```markdown
## Translation Review: XX.json

**Verdict**: ✅ Good / ⚠️ Needs fixes / ❌ Major issues

### Automated checks
- [ ] Key sync test passes
- [ ] No empty values
- [ ] Placeholders preserved

### Semantic distinctions
- [ ] passphrase ≠ password (distinct translations)
- [ ] Abbreviations preserved (CLI, TTL, MD5, Markdown)
- [ ] Translatable terms used consistently (token, admin, streaming)

### Quality
- [ ] Natural phrasing (not machine-translated feel)
- [ ] Consistent register
- [ ] Correct punctuation for the language
- [ ] Plural forms correct

### Issues found
- [key.path] Description of issue
```

### 10. Offer fixes

If issues are found, ask the user before making changes.
