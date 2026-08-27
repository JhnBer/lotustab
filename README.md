# LotusTab 
(imma gonna add cool logo later)

## Develop

```bash
# requires Go 1.25+ (toolchain auto-downloads) and wails3
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
wails3 dev
```

## Architecture

- `internal/core` — driver port, registry, connection manager
- `internal/drivers` — one adapter package per engine
- `internal/app` — Wails services bound to the Vue UI
- `internal/storage` — JSON profiles in the user config directory
- `internal/secrets` — keychain via `go-keyring`
