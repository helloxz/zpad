package main

import (
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/plugins/updater/native"
	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/app"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/locales"
)

func main() {
	// The editor's live heap is small, and Go's default target of doubling
	// it buys nothing while a Metal drawable alone is tens of megabytes: a
	// lower target keeps the heap's high-water mark down (24 MB before it,
	// 14 MB after).
	debug.SetGCPercent(50)

	// One instance: on Windows and Linux, opening a file from the file
	// manager starts another process, which hands its command line to this
	// one and exits. macOS delivers the file to the running app itself.
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	// Signed updates from the GitHub releases of helloxz/zpad: the window is
	// drawn in native UI, so the app needs no webview to update itself. The
	// icon comes from the binary, as the resources carry no icon.png.
	mygo.Use(native.New(updater.Options{Icon: appIconPNG}))

	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		log.Fatal(err)
	}

	// The log carries what mygo says of the renderer, which a Windows build
	// without a console would lose, and what the app did with big files.
	logPath := app.SetupLogging(dir)

	s, err := settings.Load(dir)
	a := app.New(s, dir)
	a.LogPath = logPath
	a.SetAppIcon(appIconPNG)
	app.SetVersion(mygo.App.Version())
	if err != nil {
		a.SetErrorToast(locales.LoadSettingsFailed(), err.Error())
		log.Printf("load settings: %v", err)
	}
	if logPath != "" {
		log.Printf("zpad %s %s/%s", app.Version, runtime.GOOS, runtime.GOARCH)
	}
	if err := settings.Ensure(dir, s); err != nil {
		a.SetErrorToast(locales.SaveSettingsFailed(), err.Error())
		log.Printf("initialize settings: %v", err)
	}
	if err := registerFonts(); err != nil {
		log.Printf("register fonts: %v", err)
		a.SetToast(locales.FontLoadFallback())
	}

	// Start in the saved appearance, before the window draws its first
	// frame.
	app.SetThemeSource(s.Theme)

	// Open files the system asks for, from Finder, Explorer or the command
	// line.
	mygo.App.OnOpenFile(a.HandleOpenFile)
	mygo.App.OnSecondInstance(a.HandleSecondInstance)
	mygo.App.OnBeforeQuit(a.BeforeQuit)

	mygo.App.WhenReady(func() {
		a.Prepare()
		mygo.App.SetLocale(s.Language)
		a.RestoreSession()
		opts := mygo.WindowOptions{
			Title:     "ZPad",
			Width:     1200,
			Height:    800,
			MinWidth:  800,
			MinHeight: 520,
			StateKey:  "main",
			Content: ui.View(func(c *ui.Context) {
				start := time.Now()
				a.View(c)
				app.NoteFrame(time.Since(start))
			}),
		}
		if runtime.GOOS == "darwin" {
			// The toolbar doubles as the title bar: the traffic lights float
			// over it, and the view keeps clear of them with
			// ui.Context.TitleBar. Pinning them 14 DIPs from the top centers
			// them in the 45 DIP toolbar of internal/app.
			opts.TitleBarStyle = mygo.TitleBarHidden
			opts.TrafficLightPosition = &mygo.Point{X: 20, Y: 14}
		} else if runtime.GOOS == "windows" {
			// The window draws the application menu as its title bar, in a
			// band the system's minimize, maximize and close buttons share.
			opts.TitleBarStyle = mygo.TitleBarHidden
			opts.TitleBarHeight = app.MenuBarHeight
		}
		win := mygo.NewWindow(opts)
		a.Attach(win)
		a.OpenCommandLine(os.Args[1:])
	})

	defer app.CleanupPreviewCache()
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
