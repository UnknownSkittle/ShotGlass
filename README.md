# ShotGlass

ShotGlass is a Go/GTK4 desktop launcher frontend for playing Roblox through
[Sober](https://sober.vinegarhq.org/) on Linux. It lists your Roblox favorite
games and friends, then sends a selected game to Sober using the desktop's
`roblox-player:` URL handler.

## Install on Arch Linux

Run these commands in a terminal.

### 1. Install build tools and Flatpak

```sh
sudo pacman -Syu
sudo pacman -S --needed git go gtk4 gobject-introspection pkgconf flatpak
```

### 2. Install Sober

Add Flathub for your user and install Sober:

```sh
flatpak remote-add --user --if-not-exists flathub https://flathub.org/repo/flathub.flatpakrepo
flatpak install --user flathub org.vinegarhq.Sober
```

Start Sober once. On first launch, let Sober install Roblox using the default
**Automatic** option:

```sh
flatpak run org.vinegarhq.Sober
```

Close Sober after Roblox finishes installing.

### 3. Download ShotGlass

```sh
git clone https://github.com/UnknownSkittle/ShotGlass.git "$HOME/ShotGlass"
cd "$HOME/ShotGlass"
```

### 4. Build and run ShotGlass

```sh
mkdir -p "$HOME/.local/bin"
go build -o "$HOME/.local/bin/shotglass" ./cmd/shotglass
"$HOME/.local/bin/shotglass"
```

If `shotglass` is not found when you run it by name, add the local bin directory
to `PATH`:

```sh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
source "$HOME/.bashrc"
shotglass
```

The same build command can be used later to update ShotGlass:

```sh
cd "$HOME/ShotGlass"
git pull
go build -o "$HOME/.local/bin/shotglass" ./cmd/shotglass
```

### 5. Connect Roblox and launch a game

1. Paste your Roblox `.ROBLOSECURITY` cookie into the masked field. The field
   accepts either the raw cookie value or a value prefixed with
   `.ROBLOSECURITY=`.
2. Click **Connect**. ShotGlass loads your favorite games and friends.
3. Select a favorite game, then click **Launch in Sober**.

Your cookie stays in memory while ShotGlass is running; ShotGlass does not
save it to disk or include it in logs. Treat the cookie like your password:
anyone who obtains it may be able to access your Roblox account.

### Troubleshooting the Sober URL handler

After starting Sober at least once, check whether the desktop knows the
`roblox-player:` handler:

```sh
gio mime x-scheme-handler/roblox-player
```

ShotGlass first asks GLib/GIO to open the URL. If that fails, it tries
`xdg-open` and displays an error if neither method can route the URL. Check
that Sober is installed and has completed its first-run setup. You can also
try opening Sober again:

```sh
flatpak run org.vinegarhq.Sober
```

## Install with Nix (WIP)

The flake is experimental and currently provides a development shell; it does
not install or package ShotGlass. On Arch Linux, install and start the Nix
daemon:

```sh
sudo pacman -S --needed nix
sudo systemctl enable --now nix-daemon.service
```

Then, from the ShotGlass repository, enter the development shell and run the
application:

```sh
nix --extra-experimental-features 'nix-command flakes' develop
go version
pkg-config --modversion gtk4
go test ./...
go run ./cmd/shotglass
```

The development shell supplies Go, GTK4, GObject Introspection, `pkg-config`,
and a C compiler for the GTK Go bindings. Sober still needs to be installed
separately and registered as the `roblox-player:` URL handler on your desktop.
Run ShotGlass from inside `nix develop` so the Nix-provided GTK libraries are
available.

## Requirements and notes

- Go 1.24 or newer, GTK4 development libraries, GObject Introspection, and
  `pkg-config` to build outside Nix.
- A Linux desktop session with a `roblox-player:` URL handler and `xdg-open`.
- Actual transparent-window compositing depends on the desktop: X11 requires
  a compositor, and Wayland compositors control transparency.

Run the tests and start the application from the repository with:

```sh
go test ./...
go run ./cmd/shotglass
```
