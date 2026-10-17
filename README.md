# ShotGlass

ShotGlass is a native Go/GTK4 frontend for launching Roblox experiences in
[Sober](https://sober.vinegarhq.org/) on Linux. It reads Roblox favorite games
and friend presence, requests an authentication ticket when launching, and
routes the resulting `roblox-player:` URI through the desktop's registered
protocol handler.

## Requirements

- Go 1.24 or newer
- GTK 4 and GObject Introspection development libraries and `pkg-config`
- A desktop session with a registered `roblox-player:` handler (normally
  Sober's Flatpak integration)
- `xdg-open` for the launch-routing fallback

On Arch Linux, install the build/runtime dependencies with:

```sh
sudo pacman -S go gtk4 gobject-introspection pkgconf
```

Install Sober and register its `roblox-player:` URL scheme using the Sober
Flatpak setup instructions. ShotGlass does not hard-code the Flatpak path; it
uses the desktop URI association and falls back to `xdg-open`.

## Build and run

```sh
go test ./...
go run ./cmd/shotglass
```

The Go GTK bindings use cgo and compile against the system GTK4 development
libraries.

## Use

1. Paste the value of your Roblox `.ROBLOSECURITY` cookie in the masked field.
   The field accepts either the raw value or a `.ROBLOSECURITY=`-prefixed
   cookie.
2. Select **Connect** to load up to 50 favorite games and friend presence.
3. Select a favorite game, then choose **Launch in Sober**.

The cookie is kept in memory only and is not written to disk or included in
logs. Avoid sharing it; anyone with this cookie can access your Roblox
account. ShotGlass requests a short-lived authentication ticket from Roblox
when you launch an experience. The GTK interface displays API and launch
errors rather than silently treating failed requests as empty results.

## Transparency and desktop integration

The window uses GTK4 CSS with an alpha-transparent window background and a
translucent content panel. As with other GTK4 applications, actual per-pixel
transparency depends on the display backend and compositor: X11 requires a
compositor, and Wayland compositors control alpha compositing. GTK3's
`GdkScreen.GetRGBAVisual()` and app-paintable APIs are not part of GTK4, so the
application uses GTK4's surface and CSS handling instead.

ShotGlass first asks GIO to launch the complete `roblox-player:` URI. If no
desktop application is associated with the scheme or GLib routing fails, it
tries `xdg-open` and reports both errors if the fallback also fails.

## Roblox endpoints

The HTTP client uses Roblox's authenticated-user, favorite-games,
friends-list, presence, and authentication-ticket endpoints. The favorite
games display requests the latest 50 entries. Friend presence is fetched in
batches of up to 100 user IDs. Network calls have timeouts and validate
response status and JSON before updating the interface.
