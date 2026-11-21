# Development

Run commands from the repo root.

## Build

Needs Go 1.25+ and [Wails v3](https://wails.io/): `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`.

Version comes from the latest git tag (`v0.1.0` → `0.1.0`); override with `APP_VERSION=0.1.0`.

```bash
wails3 dev                              # local app
wails3 task package                     # macOS .app → bin/lotustab.app
wails3 task darwin:package:dmg          # macOS .dmg → bin/lotustab.dmg
wails3 task darwin:package:universal    # Intel + Apple Silicon .app
```

On Windows and Linux, `wails3 task package` builds that platform's installer instead of a `.app`.

## Example database

A local PostgreSQL for trying the client lives in [`.docker/compose.yaml`](../.docker/docker-compose.yaml). It is not part of the app. The password below is only for this container.

```bash
docker compose -f .docker/docker-compose.yaml up -d
```

```bash
docker compose -f .docker/docker-compose.yaml down
```

Connect in LotusTab with:

| Field | Value |
| --- | --- |
| Host | `localhost` |
| Port | `5432` |
| User | `postgres` |
| Password | `postgres` |
| Database | `lotustab` |
| SSL mode | `prefer` |

`down -v` deletes the volume and the data in it:

```bash
docker compose -f .docker/docker-compose.yaml down -v
```

## Seed

[`scripts/seed.go`](../scripts/seed.go) is a separate program in its own module, so it is not compiled into LotusTab. Names and emails come from [gofakeit](https://github.com/brianvoe/gofakeit). It connects with the same defaults as the Compose file and rebuilds the demo tables:

- `users` — 250 rows, unique email, name, active, flexible `settings` jsonb, created_at
- `sessions` — zero to three per user
- `products` — 120 rows, sku, name, price, stock
- `orders` — 800 rows, foreign key to `users`, status, amount
- `order_items` — one to four products on each order

Running it again drops those tables and inserts the rows once more.

```bash
go run -C scripts .
```

Override the connection with `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, and `PGSSLMODE`.
