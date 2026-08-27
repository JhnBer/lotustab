package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"lotustab/internal/app"
	"lotustab/internal/core/connection"
	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/mysql"
	"lotustab/internal/drivers/postgres"
	"lotustab/internal/secrets"
	"lotustab/internal/storage"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	registry := driver.NewRegistry()
	if err := registry.Register(postgres.New()); err != nil {
		log.Fatal(err)
	}
	if err := registry.Register(mysql.New()); err != nil {
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
	drivers := app.NewDriversService(registry)

	appl := application.New(application.Options{
		Name:        "LotusTab",
		Description: "Desktop database client",
		Services: []application.Service{
			application.NewService(drivers),
			application.NewService(connections),
			application.NewService(queries),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	appl.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "LotusTab",
		Width:  1280,
		Height: 800,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(10, 10, 10),
		URL:              "/",
	})

	if err := appl.Run(); err != nil {
		log.Fatal(err)
	}
}
