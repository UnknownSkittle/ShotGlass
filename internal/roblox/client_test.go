package roblox

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testClient(server *httptest.Server) *Client {
	return &Client{
		httpClient:      server.Client(),
		apiBaseURL:      server.URL,
		gamesBaseURL:    server.URL,
		friendsBaseURL:  server.URL,
		presenceBaseURL: server.URL,
		authBaseURL:     server.URL,
	}
}

func TestAuthenticatedUserSendsCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got != ".ROBLOSECURITY=secret" {
			t.Errorf("Cookie header = %q, want cookie value", got)
		}
		fmt.Fprint(w, `{"id":42,"name":"player","displayName":"Player"}`)
	}))
	defer server.Close()

	user, err := testClient(server).AuthenticatedUser(context.Background(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 42 || user.Name != "player" {
		t.Fatalf("unexpected authenticated user: %#v", user)
	}
}

func TestAuthenticationTicketRetriesCSRFChallenge(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.Method != http.MethodPost || r.Header.Get("Cookie") != ".ROBLOSECURITY=secret" {
			t.Errorf("unexpected request: %s cookie=%q", r.Method, r.Header.Get("Cookie"))
		}
		if attempts == 1 {
			w.Header().Set("X-CSRF-TOKEN", "csrf-token")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if got := r.Header.Get("X-CSRF-TOKEN"); got != "csrf-token" {
			t.Errorf("X-CSRF-TOKEN = %q, want challenge token", got)
		}
		w.Header().Set("rbx-authentication-ticket", "ticket-value")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ticket, err := testClient(server).AuthenticationTicket(context.Background(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	if ticket != "ticket-value" || attempts != 2 {
		t.Fatalf("ticket=%q attempts=%d", ticket, attempts)
	}
}

func TestFavoriteGamesAndFriendPresence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/users/42/favorite/games":
			if r.URL.Query().Get("limit") != "50" {
				t.Errorf("favorite limit = %q, want 50", r.URL.Query().Get("limit"))
			}
			fmt.Fprint(w, `{"data":[{"id":10,"name":"Game","rootPlaceId":11}]}`)
		case "/v1/my/friends":
			fmt.Fprint(w, `{"data":[{"id":100,"name":"friend","displayName":"Friend"}]}`)
		case "/v1/presence/users":
			fmt.Fprint(w, `{"userPresences":[{"userId":100,"userPresenceType":2,"lastLocation":"Experience"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := testClient(server)

	games, err := client.FavoriteGames(context.Background(), "secret", 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 || games[0].RootPlaceID != 11 {
		t.Fatalf("unexpected favorite games: %#v", games)
	}
	friends, err := client.FriendsWithPresence(context.Background(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(friends) != 1 || friends[0].State != "In game" || friends[0].LastLocation != "Experience" {
		t.Fatalf("unexpected friend presence: %#v", friends)
	}
}

func TestInvalidCookieDoesNotReachServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request should not be made for invalid cookie")
	}))
	defer server.Close()

	_, err := testClient(server).AuthenticatedUser(context.Background(), "secret\r\nInjected: value")
	if err == nil || !strings.Contains(err.Error(), "invalid characters") {
		t.Fatalf("expected cookie validation error, got %v", err)
	}
}
