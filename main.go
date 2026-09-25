package main

import (
	"embed"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"nova/internal/config"
	"nova/internal/icons"
	"nova/internal/nova"
	"nova/internal/platform"
	"nova/services"
)

//go:embed all:frontend/dist
var assets embed.FS

// DropEvent tells the UI which files the OS dropped onto which folder.
type DropEvent struct {
	Dir   string   `json:"dir"`
	Files []string `json:"files"`
}

func init() {
	application.RegisterEvent[services.Transfer](services.EventTransfer)
	application.RegisterEvent[string](services.EventChanged)
	application.RegisterEvent[DropEvent]("files:dropped")
	application.RegisterEvent[services.UpdateStatus](services.EventUpdate)
}

func main() {
	services.GuardUpdaterHelper()
	cfgPath, err := config.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	store, err := config.Open(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	cfg := store.Get()
	client := nova.NewClient(nova.DefaultBaseURL, cfg.APIKey)
	// Handy for development: NOVA_API_KEY overrides the stored key for this
	// run only. It is never written to the config file.
	if k := os.Getenv("NOVA_API_KEY"); k != "" {
		client.SetAPIKey(k)
	}

	transfers := services.NewTransferService(client)
	windows := &WindowService{}

	app := application.New(application.Options{
		Name:        "Nova",
		Description: "Desktop file manager for nova.storage",
		Services: []application.Service{
			application.NewService(services.NewSessionService(client, store)),
			application.NewService(services.NewFilesService(client)),
			application.NewService(transfers),
			application.NewService(windows),
			application.NewService(services.NewUpdateService()),
			application.NewServiceWithOptions(
				services.NewMediaService(client, icons.NewResolver()),
				application.ServiceOptions{Route: services.MediaRoute},
			),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Linux: application.LinuxOptions{
			ProgramName: "nova",
		},
		Windows: application.WindowsOptions{
			// WebView2 would otherwise keep its profile in %APPDATA%\<exe
			// name>, a different folder for nova.exe (installer) and
			// nova-windows-amd64.exe (portable).
			WebviewUserDataPath: filepath.Join(platform.CacheDir(), "webview"),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	windows.app = app
	windows.NewWindow("")

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// WindowService opens additional browser windows, like Nautilus' Ctrl+N.
type WindowService struct {
	app *application.App
}

// NewWindow opens a window showing path (the home folder when empty).
func (s *WindowService) NewWindow(path string) {
	u := "/"
	if path != "" {
		u += "?path=" + url.QueryEscape(path)
	}
	win := s.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Nova",
		Width:            1100,
		Height:           700,
		MinWidth:         360,
		MinHeight:        300,
		Frameless:        true,
		EnableFileDrop:   true,
		BackgroundColour: application.NewRGB(36, 36, 36),
		URL:              u,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 46,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	// Deliver OS file drops only to the window they were dropped on.
	win.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		files := e.Context().DroppedFiles()
		if len(files) == 0 {
			return
		}
		dir := ""
		if d := e.Context().DropTargetDetails(); d != nil {
			dir = d.Attributes["data-path"]
		}
		win.DispatchWailsEvent(&application.CustomEvent{Name: "files:dropped", Data: DropEvent{Dir: dir, Files: files}})
	})
}
