---
description: Critically review local uncommitted or staged changes (git diff review)
---

# Review Local Changes

A thorough, critical review of local git changes — mimicking a senior engineer's code review.

CRITICAL RULE: NEVER perform any write action on GitHub without explicit user permission. This includes but is not limited to: submitting PR reviews, posting comments, creating/merging pull requests, pushing commits, creating branches, or creating issues. Always draft the content locally and present it to the user for review and approval BEFORE publishing anything to GitHub.

## When to Use

- Before committing, to catch issues early
- When the user asks to review changes, review diff, review local changes, etc.
- Invoked via `/review-changes`

## Steps

### 1. Determine the scope

Run one of the following based on user intent:

```bash
# Unstaged changes
git diff

# Staged changes
git diff --cached

# All local changes (staged + unstaged) vs HEAD
git diff HEAD

# Changes on branch vs main/master
git diff main...HEAD
```

// turbo-all

If the user doesn't specify, default to `git diff HEAD` (all uncommitted changes).

If the diff is very large (>500 lines), use `git diff HEAD --stat` first to get an overview, then review files individually with `git diff HEAD -- <path>`.

### 2. Review each changed file

For every modified file, evaluate against ALL of the following checklist:

#### Correctness
- [ ] Does the logic do what it claims?
- [ ] Are edge cases handled (nil, empty, zero values, boundaries)?
- [ ] Are error paths correct (proper error returns, no swallowed errors)?
- [ ] Are there race conditions or concurrency issues?

#### Consistency
- [ ] Does the change follow existing patterns in the codebase?
- [ ] Are naming conventions consistent (variables, functions, types)?
- [ ] Does it match the code style of surrounding files?

#### Completeness
- [ ] Are all necessary files modified (tests, docs, config)?
- [ ] Are there TODO/FIXME left behind unintentionally?
- [ ] Did the change update ARCHITECTURE.md / AGENTS.md / docs/ if needed? (per project rules)
- [ ] Are imports clean (no unused, no missing)?

#### Robustness
- [ ] Is stdout vs stderr usage correct? (for CLI tools)
- [ ] Are resources properly cleaned up (files, connections, goroutines)?
- [ ] Are there potential panics (nil pointer dereference, out-of-range)?

#### Security
- [ ] Are there hardcoded secrets or credentials?
- [ ] Is user input validated/sanitized?
- [ ] Are there path traversal or injection risks?

#### Tests
- [ ] Are new behaviors covered by tests?
- [ ] Do existing tests still pass with these changes?
- [ ] Are test assertions meaningful (not just "no error")?

#### Documentation (`docs/`, README, `--help`)
- [ ] Do user-facing docs reflect the new/changed behavior?
- [ ] Is the level of detail appropriate? (not too verbose, not too terse)
- [ ] Are copy-paste ready examples provided for new features?
- [ ] Is the language user-friendly and jargon-free where possible?
- [ ] Are options/flags documented consistently with `--help` output?
- [ ] Are there broken or outdated screenshots/examples?

#### CSS / Theming (if webapp styles were changed)
- [ ] Do changes use design tokens (`surface-*`, `accent-*`) instead of hardcoded colors?
- [ ] Do new/modified components look correct in both dark and light themes?
- [ ] Is there no theme flash (FODT) when loading with empty cache?
- [ ] Are theme-specific overrides in the correct place (theme CSS file, not `style.css`)?
- [ ] Are new UI elements accessible (sufficient contrast, focus states, hover states)?
- [ ] Is the user experience consistent across views (Home, Admin, Download)?

#### i18n / Translations (if locale files were changed)
- [ ] Do all locale files have the same key set as `en.json`? (run `npm test -- --run src/__tests__/locales.test.js`)
- [ ] Are `{placeholder}` tokens preserved exactly in every translation?
- [ ] Are semantic distinctions preserved (e.g. "passphrase" ≠ "password")? See `/review-language` workflow.
- [ ] Does each language follow its own punctuation rules? (e.g. French requires spaces before `:`, `?`, `!`, `;`)
- [ ] Are `languagePicker` entries present for every supported language in every locale file?
- [ ] Are new languages registered in both `settings.js` (`BUILTIN_LANGUAGES`) and `i18n.js` (import + messages)?
- [ ] Does each new language have a matching flag SVG in `public/flags/`?
- [ ] Are pipe-separated plurals (`{count} file | {count} files`) correct for the target language's plural rules?

### 3. Cross-check with project context

- Read the relevant ARCHITECTURE.md file(s) for the changed packages
- Verify the change aligns with documented patterns
- Check if AGENTS.md needs updating

### 4. Lint and build

Only if there are some code changes

```bash
make lint 2>&1 | tail -20
make client server 2>&1 | tail -20
```

If webapp files were changed:

```bash
make frontend 2>&1 | tail -20
make test-frontend 2>&1 | tail -40
```

If webapp e2e tests or source files that affect UI behavior were changed (ask for confirmation as this takes ~3 min):

```bash
make test-frontend-e2e 2>&1 | tail -20
```

### 5. Run tests

If there are some code changes (ask the for confirmation as this takes ~3 min)

```bash
make test 2>&1 | tail -40
```

If docs were changed, validate the docs build:

```bash
make docs 2>&1 | tail -20
```

### 5b. Vulnerability check

```bash
make vuln 2>&1 | tail -40
```

### 6. Produce the review report

Structure the output as:

```markdown
## Review Summary

**Scope**: <what was reviewed, e.g. "3 files, 47 insertions, 12 deletions">
**Verdict**: ✅ LGTM / ⚠️ Minor issues / ❌ Changes requested

## Issues Found

### 🔴 Critical (must fix)
- [file:line] Description of the issue

### 🟡 Suggestions (should fix)
- [file:line] Description of the suggestion

### 🔵 Nits (optional)
- [file:line] Description of the nit

## What Looks Good
- Brief mention of things done well

## Checklist
- [ ] Build passes
- [ ] Tests pass
- [ ] Docs updated
- [ ] ARCHITECTURE.md in sync
```

### 7. Offer to fix issues

If issues are found, ask the user:
> Would you like me to fix the [critical/suggested] issues?

Do NOT auto-fix without asking. Present the findings first, let the user decide.

## Important Notes

- Be genuinely critical — the goal is to catch bugs before they ship
- Don't just rubber-stamp changes with "LGTM" unless they're truly clean
- Pay special attention to subtle issues: off-by-one, missing error handling, stdout/stderr confusion, goroutine leaks
- If a change seems incomplete (e.g., missing tests for new behavior), flag it
- Compare against ARCHITECTURE.md and project conventions