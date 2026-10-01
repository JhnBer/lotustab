package main

import (
	"embed"
	"log"
	"runtime"

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
	about := app.NewAboutService(currentVersion, true, secretStore)

	appl := application.New(application.Options{
		Name:        "LotusTab",
		Description: "Desktop database client",
		Services: []application.Service{
			application.NewService(drivers),
			application.NewService(connections),
			application.NewService(queries),
			application.NewService(exports),
			application.NewService(windows),
			application.NewService(about),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	applyAppMenu(appl)
	appl.Window.NewWithOptions(app.WindowOptions("main", "LotusTab", "/"))

	if err := appl.Run(); err != nil {
		log.Fatal(err)
	}
}

func applyAppMenu(appl *application.App) {
	menu := appl.NewMenu()
	menu.AddRole(application.AppMenu)
	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("Export").OnClick(func(*application.Context) {
		appl.Event.Emit("open:export-data")
	})
	fileMenu.AddSeparator()
	if runtime.GOOS == "darwin" {
		fileMenu.AddRole(application.CloseWindow)
	} else {
		fileMenu.AddRole(application.Quit)
	}
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.ViewMenu)
	menu.AddRole(application.WindowMenu)
	menu.AddRole(application.HelpMenu)
	appl.Menu.Set(menu)
}
