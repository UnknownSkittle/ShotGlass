package launcher

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
)

type testRouter struct {
	err error
	got string
}

func (r *testRouter) LaunchDefaultForURI(uri string) error {
	r.got = uri
	return r.err
}

func TestBuildPlayerURI(t *testing.T) {
	uri, err := BuildPlayerURI("ticket+with/slash", 12345, 6789)
	if err != nil {
		t.Fatal(err)
	}
	want := "roblox-player:1+launchmode:play+gameinfo:ticket%2Bwith%2Fslash+placeid:12345+browsertrackerid:6789"
	if uri != want {
		t.Fatalf("BuildPlayerURI() = %q, want %q", uri, want)
	}
}

func TestBuildPlayerURIRejectsMissingInputs(t *testing.T) {
	for _, test := range []struct {
		ticket    string
		placeID   int64
		trackerID uint64
	}{
		{"", 1, 1},
		{"ticket", 0, 1},
		{"ticket", 1, 0},
	} {
		if _, err := BuildPlayerURI(test.ticket, test.placeID, test.trackerID); err == nil {
			t.Errorf("BuildPlayerURI(%q, %d, %d) returned no error", test.ticket, test.placeID, test.trackerID)
		}
	}
}

func TestLaunchUsesGLibRouteWhenAvailable(t *testing.T) {
	router := &testRouter{}
	system := &SystemRouter{logger: slog.New(slog.NewTextHandler(io.Discard, nil)), gio: router}
	if err := system.Launch(context.Background(), "roblox-player:test"); err != nil {
		t.Fatal(err)
	}
	if router.got != "roblox-player:test" {
		t.Fatalf("routed URI = %q", router.got)
	}
}

func TestLaunchFallsBackToXDGOpen(t *testing.T) {
	router := &testRouter{err: errors.New("no handler")}
	fallbackCalled := false
	system := &SystemRouter{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		gio:    router,
		fallback: func(_ context.Context, uri string) error {
			fallbackCalled = true
			if uri != "roblox-player:test" {
				t.Errorf("fallback URI = %q", uri)
			}
			return nil
		},
	}
	if err := system.Launch(context.Background(), "roblox-player:test"); err != nil {
		t.Fatal(err)
	}
	if !fallbackCalled {
		t.Fatal("xdg-open fallback was not called")
	}
}

func TestNewTrackerIDIsPositive(t *testing.T) {
	trackerID, err := NewTrackerID()
	if err != nil {
		t.Fatal(err)
	}
	if trackerID == 0 {
		t.Fatal("tracker ID must be positive")
	}
}

func TestLaunchReportsBothRoutingErrors(t *testing.T) {
	router := &testRouter{err: errors.New("no handler")}
	system := &SystemRouter{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		gio:    router,
		fallback: func(context.Context, string) error {
			return errors.New("xdg-open missing")
		},
	}
	err := system.Launch(context.Background(), "roblox-player:test")
	if err == nil || !strings.Contains(err.Error(), "GLib could not route") || !strings.Contains(err.Error(), "xdg-open fallback failed") {
		t.Fatalf("expected GLib and xdg-open errors, got %v", err)
	}
}
