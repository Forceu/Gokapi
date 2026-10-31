# Contributing to Gokapi

First off, thanks for taking the time to contribute! Gokapi is built to be a lightweight, secure way to share files, and community help is what keeps this project running.

## Security First
If you find a security vulnerability, please **do not** open a public issue. Instead, report it at https://github.com/Forceu/Gokapi/security to allow for a responsible disclosure.

## Getting Started
1. **Fork** the repository and clone it locally.
2. Ensure you have **Go 1.25+** (or the version specified in `go.mod`) installed.
3. Install dependencies: `go mod download`.
4. Create a new branch for your fix or feature: `git checkout -b feat/new-pr-name`.

## Coding Standards
- **Style:** Run `go fmt ./...` before committing. We follow standard Go idioms.
- **Tests:** If you add a feature, please add a corresponding test in the relevant `_test.go` file.
- **Documentation:** Update the `README.md` or `/docs` if you change application behaviours or procedures.

## Translations
Translations live in `internal/languages/translations/` as one JSON file per language, named after its ISO 639-1 code (e.g. `de.json`).

- To add a language, copy `en.json`, set `language_name` to the name of the language in that language itself, and translate the values. Keys must stay unchanged.
- Keep placeholders such as `{0}`, `_TOTAL_`, `_MAX_` and `{{filesize}}` intact — they are substituted at runtime.
- A few values contain HTML. Keep the tags and any `id` attributes; translate only the surrounding text.
- Partial translations are fine: missing or empty keys fall back to English, so no label will ever render blank.

`en.json` is the source of truth. When you add a new user-facing string, add the key there — the other languages will fall back to it until they are updated. See the [Languages section](https://gokapi.readthedocs.io/en/latest/advanced.html#languages) of the documentation for more detail.

## Submitting a PR
- Ensure your PR description follows our [PR Template](.github/PULL_REQUEST_TEMPLATE.md).
- Keep PRs focused. If you have two unrelated fixes, please open two PRs.
- Make sure all CI checks pass before requesting a review.

## Community & Communication
If you're unsure about an implementation detail, feel free to open a "Draft PR" or start a GitHub Discussion to get feedback early!
