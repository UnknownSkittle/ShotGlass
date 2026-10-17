package main

import (
	"log/slog"
	"os"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/unknownskittle/shotglass/internal/launcher"
	"github.com/unknownskittle/shotglass/internal/roblox"
	"github.com/unknownskittle/shotglass/ui"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	application := gtk.NewApplication("com.unknownskittle.ShotGlass", gio.ApplicationFlagsNone)
	client := roblox.NewClient(nil)
	router := launcher.NewSystemRouter(logger)
	var window *ui.Window
	application.ConnectActivate(func() {
		if window == nil {
			window = ui.New(application, client, router, logger)
		}
		window.Present()
	})
	os.Exit(application.Run(os.Args))
}
