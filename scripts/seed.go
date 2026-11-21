// Separate module (scripts/go.mod): not part of the LotusTab build.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/jackc/pgx/v5"
)

const (
	userCount    = 250
	productCount = 120
	orderCount   = 800
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn())
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed: connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed: begin: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	counts, err := seed(ctx, tx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		os.Exit(1)
	}
	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "seed: commit: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("seeded lotustab: %d users, %d sessions, %d products, %d orders, %d order_items\n",
		counts.users, counts.sessions, counts.products, counts.orders, counts.items)
}

func dsn() string {
	host := env("PGHOST", "localhost")
	port := env("PGPORT", "5432")
	user := env("PGUSER", "postgres")
	password := env("PGPASSWORD", "postgres")
	database := env("PGDATABASE", "lotustab")
	sslmode := env("PGSSLMODE", "prefer")

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + database,
	}
	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()
	return u.String()
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type counts struct {
	users, sessions, products, orders, items int
}

func seed(ctx context.Context, tx pgx.Tx) (counts, error) {
	if _, err := tx.Exec(ctx, `
		DROP TABLE IF EXISTS order_items;
		DROP TABLE IF EXISTS sessions;
		DROP TABLE IF EXISTS orders;
		DROP TABLE IF EXISTS products;
		DROP TABLE IF EXISTS users;

		CREATE TABLE users (
			id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			email      text NOT NULL UNIQUE,
			name       text NOT NULL,
			active     boolean NOT NULL,
			settings   jsonb NOT NULL,
			created_at timestamptz NOT NULL
		);

		CREATE TABLE sessions (
			id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id    bigint NOT NULL REFERENCES users (id),
			token      text NOT NULL UNIQUE,
			ip         text NOT NULL,
			user_agent text NOT NULL,
			created_at timestamptz NOT NULL,
			expires_at timestamptz NOT NULL
		);

		CREATE TABLE products (
			id     bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			sku    text NOT NULL UNIQUE,
			name   text NOT NULL,
			price  numeric(12, 2) NOT NULL,
			stock  integer NOT NULL,
			active boolean NOT NULL
		);

		CREATE TABLE orders (
			id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id    bigint NOT NULL REFERENCES users (id),
			status     text NOT NULL,
			amount     numeric(12, 2) NOT NULL,
			created_at timestamptz NOT NULL
		);

		CREATE TABLE order_items (
			id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			order_id   bigint NOT NULL REFERENCES orders (id),
			product_id bigint NOT NULL REFERENCES products (id),
			quantity   integer NOT NULL,
			unit_price numeric(12, 2) NOT NULL
		);

		CREATE INDEX sessions_user_id_idx ON sessions (user_id);
		CREATE INDEX orders_user_id_idx ON orders (user_id);
		CREATE INDEX order_items_order_id_idx ON order_items (order_id);
		CREATE INDEX order_items_product_id_idx ON order_items (product_id);
	`); err != nil {
		return counts{}, err
	}

	fake := gofakeit.New(1)
	rng := rand.New(rand.NewPCG(1, 2026))
	now := time.Now().UTC().Truncate(time.Second)

	users := makeUsers(fake, rng, now)
	userIDs, err := insertUsers(ctx, tx, users)
	if err != nil {
		return counts{}, err
	}

	sessions := makeSessions(fake, rng, now, users, userIDs)
	if err := insertSessions(ctx, tx, sessions); err != nil {
		return counts{}, err
	}

	products := makeProducts(fake, rng)
	productIDs, err := insertProducts(ctx, tx, products)
	if err != nil {
		return counts{}, err
	}

	orders, items := makeOrders(rng, now, users, userIDs, products, productIDs)
	if err := insertOrders(ctx, tx, orders, items); err != nil {
		return counts{}, err
	}

	return counts{
		users:    len(users),
		sessions: len(sessions),
		products: len(products),
		orders:   len(orders),
		items:    len(items),
	}, nil
}

type userRow struct {
	email, name string
	active      bool
	settings    string
	createdAt   time.Time
}

type sessionRow struct {
	userID               int64
	token, ip, userAgent string
	createdAt, expiresAt time.Time
}

type productRow struct {
	sku, name string
	price     float64
	stock     int
	active    bool
}

type orderRow struct {
	userID    int64
	status    string
	amount    float64
	createdAt time.Time
}

type itemRow struct {
	orderIndex int
	productID  int64
	quantity   int
	unitPrice  float64
}

func makeUsers(fake *gofakeit.Faker, rng *rand.Rand, now time.Time) []userRow {
	rows := make([]userRow, userCount)
	for i := range rows {
		name := fake.Name()
		email := uniqueEmail(fake, name, i)
		settings, err := json.Marshal(userSettings(rng))
		if err != nil {
			settings = []byte(`{}`)
		}
		rows[i] = userRow{
			email:     email,
			name:      name,
			active:    rng.IntN(100) < 88,
			settings:  string(settings),
			createdAt: now.AddDate(0, 0, -rng.IntN(800)).Add(time.Duration(rng.IntN(20)) * time.Hour),
		}
	}
	return rows
}

func uniqueEmail(fake *gofakeit.Faker, name string, n int) string {
	local := strings.ToLower(strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r == ' ' || r == '-' || r == '\'':
			return '.'
		default:
			return -1
		}
	}, name))
	local = strings.Trim(local, ".")
	if local == "" {
		local = "user"
	}
	return fmt.Sprintf("%s.%d@%s", local, n+1, strings.ToLower(fake.DomainName()))
}

func userSettings(rng *rand.Rand) map[string]any {
	locales := []string{"en", "de", "fr", "es", "ru", "pt"}
	themes := []string{"light", "dark", "system"}
	densities := []string{"comfortable", "compact"}
	timezones := []string{
		"Europe/Berlin", "Europe/Moscow", "Europe/Paris",
		"America/New_York", "America/Los_Angeles", "Asia/Tokyo",
	}
	settings := map[string]any{
		"locale":   locales[rng.IntN(len(locales))],
		"timezone": timezones[rng.IntN(len(timezones))],
	}
	if rng.IntN(5) > 0 {
		settings["theme"] = themes[rng.IntN(len(themes))]
	}
	if rng.IntN(2) == 0 {
		settings["density"] = densities[rng.IntN(len(densities))]
	}
	if rng.IntN(3) == 0 {
		settings["marketing"] = rng.IntN(4) > 0
	}
	if rng.IntN(2) == 0 {
		settings["notifications"] = map[string]any{
			"email": rng.IntN(5) > 0,
			"push":  rng.IntN(2) == 0,
		}
	}
	if rng.IntN(4) == 0 {
		settings["sidebar"] = []string{"expanded", "collapsed"}[rng.IntN(2)]
	}
	return settings
}

func makeSessions(fake *gofakeit.Faker, rng *rand.Rand, now time.Time, users []userRow, userIDs []int64) []sessionRow {
	rows := make([]sessionRow, 0, userCount*2)
	for i, user := range users {
		n := 1
		switch roll := rng.IntN(10); {
		case roll < 2:
			n = 0
		case roll < 7:
			n = 1
		case roll < 9:
			n = 2
		default:
			n = 3
		}
		for range n {
			created := user.createdAt.Add(time.Duration(rng.IntN(24*40)) * time.Hour)
			if created.After(now) {
				created = now.Add(-time.Duration(rng.IntN(48)) * time.Hour)
			}
			rows = append(rows, sessionRow{
				userID:    userIDs[i],
				token:     fake.UUID(),
				ip:        fake.IPv4Address(),
				userAgent: fake.UserAgent(),
				createdAt: created,
				expiresAt: created.Add(time.Duration(7+rng.IntN(24)) * 24 * time.Hour),
			})
		}
	}
	return rows
}

func makeProducts(fake *gofakeit.Faker, rng *rand.Rand) []productRow {
	rows := make([]productRow, productCount)
	seen := map[string]struct{}{}
	for i := range rows {
		name := fake.ProductName()
		if _, ok := seen[name]; ok {
			name = fmt.Sprintf("%s %d", name, i+1)
		}
		seen[name] = struct{}{}
		cents := 199 + rng.IntN(49800)
		rows[i] = productRow{
			sku:    fmt.Sprintf("SKU-%05d", i+1),
			name:   name,
			price:  float64(cents) / 100,
			stock:  rng.IntN(400),
			active: rng.IntN(100) < 90,
		}
	}
	return rows
}

func makeOrders(rng *rand.Rand, now time.Time, users []userRow, userIDs []int64, products []productRow, productIDs []int64) ([]orderRow, []itemRow) {
	statuses := []string{"pending", "paid", "shipped", "cancelled", "refunded"}
	orders := make([]orderRow, orderCount)
	items := make([]itemRow, 0, orderCount*3)
	for i := range orders {
		userIdx := rng.IntN(len(users))
		created := users[userIdx].createdAt.Add(time.Duration(rng.IntN(24*60)) * time.Hour)
		if created.After(now) {
			created = now.Add(-time.Duration(rng.IntN(72)) * time.Hour)
		}
		nItems := 1 + rng.IntN(4)
		used := map[int]struct{}{}
		var amount float64
		for len(used) < nItems {
			productIdx := rng.IntN(len(products))
			if _, ok := used[productIdx]; ok {
				continue
			}
			used[productIdx] = struct{}{}
			qty := 1 + rng.IntN(5)
			price := products[productIdx].price
			amount += price * float64(qty)
			items = append(items, itemRow{
				orderIndex: i,
				productID:  productIDs[productIdx],
				quantity:   qty,
				unitPrice:  price,
			})
		}
		orders[i] = orderRow{
			userID:    userIDs[userIdx],
			status:    statuses[rng.IntN(len(statuses))],
			amount:    amount,
			createdAt: created,
		}
	}
	return orders, items
}

func insertUsers(ctx context.Context, tx pgx.Tx, rows []userRow) ([]int64, error) {
	emails := make([]string, len(rows))
	names := make([]string, len(rows))
	active := make([]bool, len(rows))
	settings := make([]string, len(rows))
	created := make([]time.Time, len(rows))
	for i, row := range rows {
		emails[i] = row.email
		names[i] = row.name
		active[i] = row.active
		settings[i] = row.settings
		created[i] = row.createdAt
	}
	return queryIDs(ctx, tx, `
		INSERT INTO users (email, name, active, settings, created_at)
		SELECT email, name, active, settings::jsonb, created_at
		FROM unnest($1::text[], $2::text[], $3::bool[], $4::text[], $5::timestamptz[])
			AS t(email, name, active, settings, created_at)
		RETURNING id
	`, emails, names, active, settings, created)
}

func insertSessions(ctx context.Context, tx pgx.Tx, rows []sessionRow) error {
	userIDs := make([]int64, len(rows))
	tokens := make([]string, len(rows))
	ips := make([]string, len(rows))
	agents := make([]string, len(rows))
	created := make([]time.Time, len(rows))
	expires := make([]time.Time, len(rows))
	for i, row := range rows {
		userIDs[i] = row.userID
		tokens[i] = row.token
		ips[i] = row.ip
		agents[i] = row.userAgent
		created[i] = row.createdAt
		expires[i] = row.expiresAt
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO sessions (user_id, token, ip, user_agent, created_at, expires_at)
		SELECT user_id, token, ip, user_agent, created_at, expires_at
		FROM unnest($1::bigint[], $2::text[], $3::text[], $4::text[], $5::timestamptz[], $6::timestamptz[])
			AS t(user_id, token, ip, user_agent, created_at, expires_at)
	`, userIDs, tokens, ips, agents, created, expires)
	return err
}

func insertProducts(ctx context.Context, tx pgx.Tx, rows []productRow) ([]int64, error) {
	skus := make([]string, len(rows))
	names := make([]string, len(rows))
	prices := make([]float64, len(rows))
	stock := make([]int32, len(rows))
	active := make([]bool, len(rows))
	for i, row := range rows {
		skus[i] = row.sku
		names[i] = row.name
		prices[i] = row.price
		stock[i] = int32(row.stock)
		active[i] = row.active
	}
	return queryIDs(ctx, tx, `
		INSERT INTO products (sku, name, price, stock, active)
		SELECT sku, name, price, stock, active
		FROM unnest($1::text[], $2::text[], $3::numeric[], $4::int[], $5::bool[])
			AS t(sku, name, price, stock, active)
		RETURNING id
	`, skus, names, prices, stock, active)
}

func insertOrders(ctx context.Context, tx pgx.Tx, orders []orderRow, items []itemRow) error {
	userIDs := make([]int64, len(orders))
	statuses := make([]string, len(orders))
	amounts := make([]float64, len(orders))
	created := make([]time.Time, len(orders))
	for i, row := range orders {
		userIDs[i] = row.userID
		statuses[i] = row.status
		amounts[i] = row.amount
		created[i] = row.createdAt
	}
	orderIDs, err := queryIDs(ctx, tx, `
		INSERT INTO orders (user_id, status, amount, created_at)
		SELECT user_id, status, amount, created_at
		FROM unnest($1::bigint[], $2::text[], $3::numeric[], $4::timestamptz[])
			AS t(user_id, status, amount, created_at)
		RETURNING id
	`, userIDs, statuses, amounts, created)
	if err != nil {
		return err
	}

	orderIDsCol := make([]int64, len(items))
	productIDs := make([]int64, len(items))
	qty := make([]int32, len(items))
	prices := make([]float64, len(items))
	for i, item := range items {
		orderIDsCol[i] = orderIDs[item.orderIndex]
		productIDs[i] = item.productID
		qty[i] = int32(item.quantity)
		prices[i] = item.unitPrice
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO order_items (order_id, product_id, quantity, unit_price)
		SELECT order_id, product_id, quantity, unit_price
		FROM unnest($1::bigint[], $2::bigint[], $3::int[], $4::numeric[])
			AS t(order_id, product_id, quantity, unit_price)
	`, orderIDsCol, productIDs, qty, prices)
	return err
}

func queryIDs(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]int64, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
