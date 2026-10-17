package launcher

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
)

type URIRouter interface {
	LaunchDefaultForURI(uri string) error
}

type SystemRouter struct {
	logger   *slog.Logger
	gio      URIRouter
	fallback func(context.Context, string) error
}

type gioRouter struct{}

func (gioRouter) LaunchDefaultForURI(uri string) error {
	return gio.AppInfoLaunchDefaultForURI(uri, nil)
}

func NewSystemRouter(logger *slog.Logger) *SystemRouter {
	if logger == nil {
		logger = slog.Default()
	}
	return &SystemRouter{logger: logger, gio: gioRouter{}, fallback: launchWithXDGOpen}
}

func (r *SystemRouter) Launch(ctx context.Context, uri string) error {
	if err := r.gio.LaunchDefaultForURI(uri); err == nil {
		return nil
	} else {
		r.logger.Warn("GLib could not route the Roblox player URI; trying xdg-open", "error", err)
		fallback := r.fallback
		if fallback == nil {
			fallback = launchWithXDGOpen
		}
		if fallbackErr := fallback(ctx, uri); fallbackErr != nil {
			r.logger.Error("No working handler is registered for the roblox-player URI scheme", "glib_error", err, "xdg_open_error", fallbackErr)
			return errors.Join(
				fmt.Errorf("GLib could not route the Roblox link: %w", err),
				fmt.Errorf("xdg-open fallback failed: %w", fallbackErr),
			)
		}
	}
	return nil
}

func launchWithXDGOpen(ctx context.Context, uri string) error {
	return exec.CommandContext(ctx, "xdg-open", uri).Run()
}

func BuildPlayerURI(ticket string, placeID int64, trackerID uint64) (string, error) {
	if ticket == "" {
		return "", errors.New("Roblox authentication ticket is empty")
	}
	if placeID <= 0 {
		return "", errors.New("Roblox place ID must be positive")
	}
	if trackerID == 0 {
		return "", errors.New("browser tracker ID must be positive")
	}
	encodedTicket := strings.ReplaceAll(url.PathEscape(ticket), "+", "%2B")
	return "roblox-player:1+launchmode:play+gameinfo:" + encodedTicket +
		"+placeid:" + strconv.FormatInt(placeID, 10) +
		"+browsertrackerid:" + strconv.FormatUint(trackerID, 10), nil
}

func NewTrackerID() (uint64, error) {
	max := big.NewInt(1<<31 - 1)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, fmt.Errorf("generate browser tracker ID: %w", err)
	}
	return uint64(value.Int64()) + 1, nil
}
