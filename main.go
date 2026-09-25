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

// EventMouseNav carries "back" or "forward" from the mouse's side buttons.
const EventMouseNav = "mouse:nav"

// DropEvent tells the UI which files the OS dropped onto which folder.
type DropEvent struct {
	Dir   string   `json:"dir"`
	Files []string `json:"files"`
}

func init() {
	application.RegisterEvent[services.Transfer](services.EventTransfer)
	application.RegisterEvent[string](services.EventChanged)
	application.RegisterEvent[DropEvent]("files:dropped")
	application.RegisterEvent[string](EventMouseNav)
	application.RegisterEvent[CrossDrag](EventCrossDrag)
	application.RegisterEvent[*ItemClipboard](EventClipboard)
	application.RegisterEvent[bool](EventOSClipboardFiles)
	application.RegisterEvent[services.UpdateStatus](services.EventUpdate)
	application.RegisterEvent[services.SystemTheme](services.EventTheme)
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
	theme := services.NewThemeService()
	windows := &WindowService{theme: theme, store: store}
	dragTransfers = transfers
	cleanDragExports()

	app := application.New(application.Options{
		Name:        "Nova",
		Description: "Desktop file manager for nova.storage",
		Services: []application.Service{
			application.NewService(services.NewSessionService(client, store)),
			application.NewService(services.NewFilesService(client)),
			application.NewService(services.NewAccountService(client, store)),
			application.NewService(transfers),
			application.NewService(windows),
			application.NewService(services.NewUpdateService()),
			application.NewService(theme),
			application.NewServiceWithOptions(
				services.NewMediaService(client, icons.NewResolver()),
				application.ServiceOptions{Route: services.MediaRoute},
			),
		},
		// Launching Nova again opens another window in the running app, like
		// Nautilus, so windows can share drags and transfers.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "storage.nova.desktop",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				windows.NewWindow("")
			},
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
	app   *application.App
	theme *services.ThemeService
	store *config.Store
}

// background is the colour a new window shows until its page has painted.
func (s *WindowService) background() application.RGBA {
	r, g, b := s.theme.WindowColour(s.store.Get().Prefs.Theme)
	return application.NewRGB(r, g, b)
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
		BackgroundColour: s.background(),
		URL:              u,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 46,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	enableMouseNav(win)
	enableCrossDrag(win)
	enableOSClipboard(win)

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
