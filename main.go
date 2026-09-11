package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"lotustab/internal/app"
	"lotustab/internal/core/connection"
	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/clickhouse"
	"lotustab/internal/drivers/mysql"
	"lotustab/internal/drivers/postgres"
	"lotustab/internal/secrets"
	"lotustab/internal/storage"
)

//go:embed all:frontend/dist
var assets embed.FS

// currentVersion is overwritten at release build via
// -ldflags "-X main.currentVersion=<tag>".
var currentVersion = "dev"

func main() {
	registry := driver.NewRegistry()
	if err := registry.Register(postgres.New()); err != nil {
		log.Fatal(err)
	}
	if err := registry.Register(mysql.New()); err != nil {
		log.Fatal(err)
	}
	if err := registry.Register(clickhouse.New()); err != nil {
		log.Fatal(err)
	}

	store, err := storage.New()
	if err != nil {
		log.Fatal(err)
	}
	secretStore := secrets.New()
	manager := connection.NewManager()

	connections := app.NewConnectionsService(registry, store, secretStore, manager)
	queries := app.NewQueryServiceFromConnections(connections)
	exports := app.NewExportService(connections)
	drivers := app.NewDriversService(registry)
	windows := app.NewWindowsService()

	appl := application.New(application.Options{
		Name:        "LotusTab",
		Description: "Desktop database client",
		Services: []application.Service{
			application.NewService(drivers),
			application.NewService(connections),
			application.NewService(queries),
			application.NewService(exports),
			application.NewService(windows),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	appl.Window.NewWithOptions(app.WindowOptions("main", "LotusTab", "/"))

	if err := appl.Run(); err != nil {
		log.Fatal(err)
	}
}
