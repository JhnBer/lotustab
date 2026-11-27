---
name: work-with-locale
description: >-
  Add or update LotusTab UI languages (vue-i18n catalogs, settings switcher,
  driver label keys). Use when the user asks to add a language, translate the
  UI, edit frontend/src/locales, or change locale fallback and coverage.
---

# Work with locale

English (`en`) is the source catalog and the fallback. The other shipped catalogs are the remaining files in `frontend/src/locales/`. The app does not follow the OS language. The default locale is English.

A missing key resolves to the English string, not to the raw key (`fallbackLocale: "en"` in `frontend/src/i18n.ts`). Catalogs must still contain the same key set. `TestLocaleCatalogsMatchDriverKeys` fails when they diverge or when a driver `LabelKey` is absent from `en.json`.

## How text is chosen

- UI chrome uses `t()` / `$t()` with keys from `frontend/src/locales/<id>.json`. Nested objects become dotted keys (`field.common.host`).
- Connection forms take `labelKey` from `ConnectionSpec`. Shared keys (`field.common.host`, `port`, `user`, `password`, `database`) only when the English wording and the meaning match. Everything else is `driver.<id>.field.*`. SSH copy is `ssh.*`, not `field.common.*`.
- Locale is `lotustab.locale` in the settings store. It is part of the settings snapshot and `settings:changed`, so a second window switches too. `document.documentElement.lang` follows it.
- The General settings switcher lists `LOCALE_OPTIONS` names in their own script. Those labels are not translation keys.
- Selects whose option text is translated are keyed with `:key="locale"` so the closed trigger does not keep the previous language. Keep that key when adding such a select.
- Do not translate: product names (LotusTab, PostgreSQL, MySQL, ClickHouse, GitHub, GPL-3.0), SQL operators, cell values, table and column names, Go `fmt.Errorf` text, driver and SQL errors, the native File/Export menu. Format-example placeholders stay as written (`localhost`, `optional`, `~/.ssh/id_ed25519`). A required-field toast uses `field.Key`, so it reads `host is required`.
- Inter has no CJK glyphs. `frontend/src/style.css` falls back to PingFang SC, Noto Sans SC, Microsoft YaHei. Japanese (`html[lang="ja"]`) prefers Hiragino Sans, Yu Gothic, and Noto Sans CJK JP before that Chinese fallback. Do not add a downloaded CJK webfont. For a new script Inter cannot draw, add a system font to that stack.

## Add a language

1. Copy `frontend/src/locales/en.json` to `frontend/src/locales/<id>.json` (`<id>` is a BCP 47 tag, like `ru` or `zh-CN`). Translate the values. Keep the key set identical to English. Strings that stay in English (product names, `TLS`, units) may match the source.
2. Add `{ id: "<id>", name: "<native name>" }` to `LOCALE_OPTIONS` in `frontend/src/i18n.ts`. `LOCALES`, `isAppLocale`, `messages`, `AppLocale`, and the settings switcher follow that array. The name is the language's own name, not a `$t` key. The module throws if the array and the catalog files disagree. If the new script needs glyphs Inter cannot draw, add a system font in `frontend/src/style.css`.
3. `TestLocaleCatalogsMatchDriverKeys` already compares every catalog file in `frontend/src/locales/` with `en.json`.
4. From `frontend/`, run `pnpm run locales`. `pnpm run locales -- --compact` prints one percentage per language and nothing else. A key counts as filled when the locale has a non-empty string. Identical-to-English strings count as filled; the full report lists them.

## Add a string

Put the key in every file in `frontend/src/locales/` in the same change. Use an existing `field.common.*` key only when both the English text and the meaning already match. Driver-specific wording gets `driver.<id>.field.*`. Then run `pnpm run locales -- --compact` and `go test ./internal/core/driver/ -run TestLocaleCatalogsMatchDriverKeys`.
