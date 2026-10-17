package ui

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/unknownskittle/shotglass/internal/launcher"
	"github.com/unknownskittle/shotglass/internal/roblox"
)

//go:embed style.css
var styleCSS string

type Window struct {
	window         *gtk.ApplicationWindow
	api            *roblox.Client
	router         *launcher.SystemRouter
	logger         *slog.Logger
	cookie         *gtk.Entry
	loginButton    *gtk.Button
	refreshButton  *gtk.Button
	launchButton   *gtk.Button
	status         *gtk.Label
	userLabel      *gtk.Label
	selectedLabel  *gtk.Label
	favoriteList   *gtk.ListBox
	friendList     *gtk.ListBox
	selectedButton *gtk.Button
	selectedPlace  int64
	selectedName   string
	loading        bool
}

func New(application *gtk.Application, api *roblox.Client, router *launcher.SystemRouter, logger *slog.Logger) *Window {
	if logger == nil {
		logger = slog.Default()
	}
	applyStyles(logger)
	w := &Window{
		window:        gtk.NewApplicationWindow(application),
		api:           api,
		router:        router,
		logger:        logger,
		cookie:        gtk.NewEntry(),
		loginButton:   gtk.NewButtonWithLabel("Connect"),
		refreshButton: gtk.NewButtonWithLabel("Refresh"),
		launchButton:  gtk.NewButtonWithLabel("Launch in Sober"),
		status:        gtk.NewLabel("Enter your Roblox cookie to load your games and friends."),
		userLabel:     gtk.NewLabel("Not connected"),
		selectedLabel: gtk.NewLabel("Select a favorite game to launch."),
		favoriteList:  gtk.NewListBox(),
		friendList:    gtk.NewListBox(),
	}
	w.build()
	return w
}

func (w *Window) Present() {
	w.window.Present()
}

func (w *Window) build() {
	w.window.SetTitle("ShotGlass")
	w.window.SetDefaultSize(860, 620)

	panel := gtk.NewBox(gtk.OrientationVertical, 16)
	panel.AddCSSClass("launcher-panel")
	panel.SetMarginTop(18)
	panel.SetMarginBottom(18)
	panel.SetMarginStart(18)
	panel.SetMarginEnd(18)

	title := gtk.NewLabel("ShotGlass")
	title.AddCSSClass("title")
	title.SetHAlign(gtk.AlignStart)
	subtitle := gtk.NewLabel("A native Roblox launcher for Sober")
	subtitle.AddCSSClass("subtitle")
	subtitle.SetHAlign(gtk.AlignStart)
	panel.Append(title)
	panel.Append(subtitle)

	accountRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
	w.cookie.SetPlaceholderText("Paste your .ROBLOSECURITY cookie")
	w.cookie.SetVisibility(false)
	w.cookie.SetHExpand(true)
	w.loginButton.AddCSSClass("primary-action")
	accountRow.Append(w.cookie)
	accountRow.Append(w.loginButton)
	accountRow.Append(w.refreshButton)
	panel.Append(accountRow)

	w.userLabel.AddCSSClass("muted")
	w.userLabel.SetHAlign(gtk.AlignStart)
	panel.Append(w.userLabel)

	columns := gtk.NewBox(gtk.OrientationHorizontal, 16)
	columns.SetVExpand(true)

	favoritesColumn := gtk.NewBox(gtk.OrientationVertical, 8)
	favoritesColumn.SetHExpand(true)
	favoritesTitle := gtk.NewLabel("Favorite games")
	favoritesTitle.AddCSSClass("section-title")
	favoritesTitle.SetHAlign(gtk.AlignStart)
	favoritesColumn.Append(favoritesTitle)
	favoritesScroll := gtk.NewScrolledWindow()
	favoritesScroll.SetMinContentHeight(230)
	favoritesScroll.SetVExpand(true)
	w.favoriteList.SetSelectionMode(gtk.SelectionNone)
	favoritesScroll.SetChild(w.favoriteList)
	favoritesColumn.Append(favoritesScroll)

	friendsColumn := gtk.NewBox(gtk.OrientationVertical, 8)
	friendsColumn.SetHExpand(true)
	friendsTitle := gtk.NewLabel("Friends")
	friendsTitle.AddCSSClass("section-title")
	friendsTitle.SetHAlign(gtk.AlignStart)
	friendsColumn.Append(friendsTitle)
	friendsScroll := gtk.NewScrolledWindow()
	friendsScroll.SetMinContentHeight(230)
	friendsScroll.SetVExpand(true)
	w.friendList.SetSelectionMode(gtk.SelectionNone)
	friendsScroll.SetChild(w.friendList)
	friendsColumn.Append(friendsScroll)

	columns.Append(favoritesColumn)
	columns.Append(friendsColumn)
	panel.Append(columns)

	w.selectedLabel.AddCSSClass("muted")
	w.selectedLabel.SetHAlign(gtk.AlignStart)
	panel.Append(w.selectedLabel)
	w.launchButton.AddCSSClass("primary-action")
	w.launchButton.SetSensitive(false)
	panel.Append(w.launchButton)

	w.status.SetHAlign(gtk.AlignStart)
	w.status.SetWrap(true)
	panel.Append(w.status)
	w.window.SetChild(panel)

	w.loginButton.ConnectClicked(w.loadAccount)
	w.refreshButton.ConnectClicked(w.loadAccount)
	w.cookie.ConnectActivate(w.loadAccount)
	w.launchButton.ConnectClicked(w.launchGame)
}

func applyStyles(logger *slog.Logger) {
	display := gdk.DisplayGetDefault()
	if display == nil {
		logger.Error("GTK4 did not provide a display; cannot apply launcher styling")
		return
	}
	provider := gtk.NewCSSProvider()
	provider.ConnectParsingError(func(_ *gtk.CSSSection, err error) {
		logger.Error("Failed to parse launcher CSS", "error", err)
	})
	provider.LoadFromData(styleCSS)
	gtk.StyleContextAddProviderForDisplay(display, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}

func (w *Window) loadAccount() {
	if w.loading {
		return
	}
	cookie := strings.TrimSpace(w.cookie.Text())
	if cookie == "" {
		w.setStatus("Enter a Roblox .ROBLOSECURITY cookie.", true)
		return
	}
	w.loginButton.SetSensitive(false)
	w.refreshButton.SetSensitive(false)
	w.cookie.SetSensitive(false)
	w.loading = true
	w.userLabel.SetText("Connecting to Roblox…")
	w.favoriteList.RemoveAll()
	w.friendList.RemoveAll()
	w.selectedPlace = 0
	w.selectedName = ""
	w.selectedButton = nil
	w.selectedLabel.SetText("Select a favorite game to launch.")
	w.launchButton.SetSensitive(false)
	w.setStatus("Connecting to Roblox…", false)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		user, err := w.api.AuthenticatedUser(ctx, cookie)
		if err != nil {
			w.postMain(func() {
				w.loading = false
				w.loginButton.SetSensitive(true)
				w.refreshButton.SetSensitive(true)
				w.cookie.SetSensitive(true)
				w.setStatus("Could not authenticate with Roblox: "+err.Error(), true)
			})
			return
		}

		var (
			favorites []roblox.Favorite
			friends   []roblox.FriendPresence
			favErr    error
			friendErr error
			wg        sync.WaitGroup
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			favorites, favErr = w.api.FavoriteGames(ctx, cookie, user.ID)
		}()
		go func() {
			defer wg.Done()
			friends, friendErr = w.api.FriendsWithPresence(ctx, cookie)
		}()
		wg.Wait()

		w.postMain(func() {
			w.loading = false
			w.loginButton.SetSensitive(true)
			w.refreshButton.SetSensitive(true)
			w.cookie.SetSensitive(true)
			w.userLabel.SetText(fmt.Sprintf("Signed in as %s (@%s)", displayName(user), user.Name))
			if favErr == nil {
				w.showFavorites(favorites)
			}
			if friendErr == nil {
				w.showFriends(friends)
			}
			switch {
			case favErr != nil && friendErr != nil:
				w.setStatus(fmt.Sprintf("Connected, but loading games failed: %v; loading friends failed: %v", favErr, friendErr), true)
			case favErr != nil:
				w.setStatus("Connected, but loading favorite games failed: "+favErr.Error(), true)
			case friendErr != nil:
				w.setStatus("Connected, but loading friends failed: "+friendErr.Error(), true)
			default:
				w.setStatus(fmt.Sprintf("Loaded %d favorite games and %d friends.", len(favorites), len(friends)), false)
			}
		})
	}()
}

func displayName(user roblox.User) string {
	if user.DisplayName != "" {
		return user.DisplayName
	}
	return user.Name
}

func (w *Window) showFavorites(favorites []roblox.Favorite) {
	w.favoriteList.RemoveAll()
	w.selectedPlace = 0
	w.selectedName = ""
	w.selectedButton = nil
	w.launchButton.SetSensitive(false)
	w.selectedLabel.SetText("Select a favorite game to launch.")
	if len(favorites) == 0 {
		w.favoriteList.Append(gtk.NewLabel("No favorite games found."))
		return
	}
	for _, favorite := range favorites {
		game := favorite
		placeID := game.RootPlaceID
		if placeID == 0 {
			label := gtk.NewLabel(game.Name + " (missing root place ID)")
			label.SetHAlign(gtk.AlignStart)
			w.favoriteList.Append(label)
			continue
		}
		button := gtk.NewButtonWithLabel(game.Name)
		button.AddCSSClass("game-button")
		button.SetHAlign(gtk.AlignFill)
		button.SetTooltipText(game.Description)
		button.ConnectClicked(func() {
			if w.selectedButton != nil {
				w.selectedButton.RemoveCSSClass("selected")
			}
			w.selectedButton = button
			button.AddCSSClass("selected")
			w.selectedPlace = placeID
			w.selectedName = game.Name
			w.selectedLabel.SetText(fmt.Sprintf("Selected: %s (place %d)", game.Name, placeID))
			w.launchButton.SetSensitive(true)
		})
		w.favoriteList.Append(button)
	}
}

func (w *Window) showFriends(friends []roblox.FriendPresence) {
	w.friendList.RemoveAll()
	if len(friends) == 0 {
		w.friendList.Append(gtk.NewLabel("No friends found."))
		return
	}
	for _, friend := range friends {
		name := friend.DisplayName
		if name == "" {
			name = friend.Name
		}
		line := fmt.Sprintf("%s (@%s) · %s", name, friend.Name, friend.State)
		if friend.LastLocation != "" {
			line += " — " + friend.LastLocation
		}
		label := gtk.NewLabel(line)
		label.SetHAlign(gtk.AlignStart)
		label.SetWrap(true)
		w.friendList.Append(label)
	}
}

func (w *Window) launchGame() {
	cookie := strings.TrimSpace(w.cookie.Text())
	placeID := w.selectedPlace
	gameName := w.selectedName
	if cookie == "" || placeID <= 0 {
		w.setStatus("Connect your Roblox account and select a game first.", true)
		return
	}
	w.launchButton.SetSensitive(false)
	w.setStatus("Requesting a Roblox launch ticket for "+gameName+"…", false)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ticket, err := w.api.AuthenticationTicket(ctx, cookie)
		if err == nil {
			var trackerID uint64
			trackerID, err = launcher.NewTrackerID()
			if err == nil {
				var uri string
				uri, err = launcher.BuildPlayerURI(ticket, placeID, trackerID)
				if err == nil {
					err = w.router.Launch(ctx, uri)
				}
			}
		}
		w.postMain(func() {
			w.launchButton.SetSensitive(w.selectedPlace > 0)
			if err != nil {
				w.logger.Error("Failed to launch Roblox experience", "error", err, "place_id", placeID)
				w.setStatus("Could not launch "+gameName+": "+err.Error(), true)
				return
			}
			w.setStatus("Sent "+gameName+" to Sober.", false)
		})
	}()
}

func (w *Window) postMain(callback func()) {
	glib.IdleAdd(func() bool {
		callback()
		return false
	})
}

func (w *Window) setStatus(message string, isError bool) {
	w.status.SetText(message)
	if isError {
		w.status.AddCSSClass("status-error")
		w.status.RemoveCSSClass("status-success")
	} else {
		w.status.RemoveCSSClass("status-error")
		w.status.AddCSSClass("status-success")
	}
}
